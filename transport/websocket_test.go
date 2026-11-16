package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dsablic/faye-go/protocol"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
)

func TestSendFailsWhenPeerStopsReading(t *testing.T) {
	original := writeTimeout
	writeTimeout = 200 * time.Millisecond
	defer func() { writeTimeout = original }()

	sendErr := make(chan error, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			sendErr <- err
			return
		}
		conn := &WebSocketConnection{ws: ws, failed: atomic.NewBool(false)}
		payload := []protocol.Message{{"data": strings.Repeat("x", 64*1024)}}
		for {
			if err := conn.Send(payload); err != nil {
				sendErr <- err
				return
			}
		}
	}))
	defer server.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	select {
	case err := <-sendErr:
		if !strings.Contains(err.Error(), "timeout") {
			t.Fatalf("expected a write timeout, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Send kept blocking on a peer that stopped reading")
	}
}
