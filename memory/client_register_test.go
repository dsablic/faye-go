package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/dsablic/faye-go/protocol"
)

type discardLogger struct{}

func (discardLogger) Debugf(string, ...interface{}) {}
func (discardLogger) Infof(string, ...interface{})  {}
func (discardLogger) Warnf(string, ...interface{})  {}
func (discardLogger) Errorf(string, ...interface{}) {}
func (discardLogger) Fatalf(string, ...interface{}) {}
func (discardLogger) Panicf(string, ...interface{}) {}

type stalledConnection struct {
	sendStarted chan struct{}
	release     chan struct{}
	startOnce   sync.Once
}

func newStalledConnection() *stalledConnection {
	return &stalledConnection{sendStarted: make(chan struct{}), release: make(chan struct{})}
}

func (c *stalledConnection) Send([]protocol.Message) error {
	c.startOnce.Do(func() { close(c.sendStarted) })
	<-c.release
	return nil
}

func (c *stalledConnection) SendJsonp(msgs []protocol.Message, _ string) error { return c.Send(msgs) }
func (c *stalledConnection) IsConnected() bool                                 { return true }
func (c *stalledConnection) IsSingleShot() bool                                { return false }
func (c *stalledConnection) Close()                                            {}

func completesWithin(t *testing.T, what string, timeout time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("%s did not complete within %s", what, timeout)
	}
}

func TestStalledSendDoesNotFreezeRegister(t *testing.T) {
	register := NewClientRegister()
	conn := newStalledConnection()
	defer close(conn.release)

	stalled := protocol.NewClient(1, discardLogger{})
	stalled.SetConnection(conn)
	if err := register.AddClient(stalled); err != nil {
		t.Fatalf("AddClient: %v", err)
	}

	go stalled.Send(protocol.Message{"channel": "/stuck"}, "")
	<-conn.sendStarted

	completesWithin(t, "Reap", time.Second, func() { register.Reap() })
	completesWithin(t, "AddClient", time.Second, func() {
		if err := register.AddClient(protocol.NewClient(2, discardLogger{})); err != nil {
			t.Errorf("AddClient: %v", err)
		}
	})
	completesWithin(t, "GetClient", time.Second, func() { register.GetClient(2) })
}

func TestReapKeepsReplacementClient(t *testing.T) {
	register := NewClientRegister()

	original := protocol.NewClient(1, discardLogger{})
	if err := register.AddClient(original); err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	register.Reap()

	if got := register.GetClient(1); got != nil {
		t.Fatalf("disconnected client was not reaped")
	}

	replacement := protocol.NewClient(1, discardLogger{})
	replacement.SetConnection(newStalledConnection())
	if err := register.AddClient(replacement); err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	register.Reap()

	if got := register.GetClient(1); got != replacement {
		t.Fatalf("connected client was reaped")
	}
}
