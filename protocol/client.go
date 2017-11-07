package protocol

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dsablic/faye-go/utils"
)

type Session struct {
	conn     Connection
	timeout  int
	response Message
	client   *Client
	started  time.Time
	logger   utils.Logger
}

func NewSession(client *Client, conn Connection, timeout int, response Message, logger utils.Logger) *Session {
	session := Session{conn, timeout, response, client, time.Now(), logger}
	if timeout > 0 {
		go func() {
			time.Sleep(time.Duration(timeout) * time.Millisecond)
			session.End()
		}()
	}
	return &session
}

func (s *Session) End() {
	if s.client.isConnected() {
		s.conn.Send([]Message{s.response})
	} else {
		s.logger.Debugf("No longer connected %s", s.client.clientId)
	}
}

type ClientCounters struct {
	Failed uint64
	Sent   uint64
}

type Client struct {
	clientId    string
	connection  Connection
	responseMsg Message
	mutex       sync.RWMutex
	lastSession *Session
	created     time.Time
	logger      utils.Logger
	counters    ClientCounters
}

func NewClient(clientId string, logger utils.Logger) *Client {
	return &Client{
		clientId: clientId,
		created:  time.Now(),
		logger:   logger,
		counters: ClientCounters{0, 0},
	}
}

func (c *Client) Id() string {
	return c.clientId
}

func (c *Client) Connect(timeout int, interval int, responseMsg Message, connection Connection) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lastSession = NewSession(c, connection, timeout, responseMsg, c.logger)
	c.responseMsg = responseMsg
}

func (c *Client) SetConnection(connection Connection) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connection == nil || connection.Priority() > c.connection.Priority() {
		c.connection = connection
	}
}

func (c *Client) ShouldReap() bool {
	if !c.isConnected() {
		return true
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().Sub(c.created) > time.Duration(1*time.Minute) {
		if c.lastSession != nil &&
			time.Now().Sub(c.lastSession.started) > time.Duration(2*time.Hour) {
			return true
		}
	}
	return false
}

func (c *Client) ResetCounters() ClientCounters {
	return ClientCounters{
		Sent:   atomic.SwapUint64(&c.counters.Sent, 0),
		Failed: atomic.SwapUint64(&c.counters.Failed, 0),
	}
}

func (c *Client) Send(msg Message, jsonp string) bool {
	if c.isConnected() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		msgs := []Message{msg}
		if c.connection.IsSingleShot() {
			msgs = append(msgs, c.responseMsg)
		}
		c.logger.Debugf("Sending %d msgs to %s on %s", len(msgs), c.clientId, reflect.TypeOf(c.connection))

		var err error

		if jsonp != "" {
			err = c.connection.SendJsonp(msgs, jsonp)
		} else {
			err = c.connection.Send(msgs)
		}

		if err != nil {
			c.logger.Debugf("Was unable to send %d messages to %s", len(msgs), c.clientId)
			c.connection.Close()
			atomic.AddUint64(&c.counters.Failed, 1)
			return false
		}

		atomic.AddUint64(&c.counters.Sent, 1)
		return true
	}

	c.logger.Debugf("Not connected for %s", c.clientId)
	atomic.AddUint64(&c.counters.Failed, 1)
	return false
}

func (c *Client) isConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connection != nil && c.connection.IsConnected()
}
