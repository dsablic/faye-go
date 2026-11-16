package transport

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
)

type Server interface {
	HandleRequest(interface{}, protocol.Connection, *http.Request)
	Logger() utils.Logger
}

type WebSocketConnection struct {
	ws     *websocket.Conn
	failed *atomic.Bool
	mutex  sync.RWMutex
}

var writeTimeout = 10 * time.Second

func (wc *WebSocketConnection) writeJSON(v interface{}) error {
	wc.mutex.Lock()
	defer wc.mutex.Unlock()
	if err := wc.ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return fmt.Errorf("failed to set websocket write deadline: %w", err)
	}
	if err := wc.ws.WriteJSON(v); err != nil {
		return fmt.Errorf("failed to write to websocket: %w", err)
	}
	return nil
}

func (wc *WebSocketConnection) Send(msgs []protocol.Message) error {
	if !wc.IsConnected() {
		return errors.New("not connected")
	}
	err := wc.writeJSON(msgs)
	if err != nil {
		wc.failed.Store(true)
	}
	return err
}

func (wc *WebSocketConnection) SendJsonp(msgs []protocol.Message, _ string) error {
	return errors.New("jsonp is not supported over websockets")
}

func (wc *WebSocketConnection) IsConnected() bool {
	return !wc.failed.Load()
}

func (wc *WebSocketConnection) Close() {
	wc.failed.Store(true)
	wc.ws.Close()
}

func (wc *WebSocketConnection) IsSingleShot() bool {
	return false
}

func WebsocketServer(m Server) func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		var data interface{}
		wsConn := WebSocketConnection{ws: ws, failed: atomic.NewBool(false)}
		for {
			err := ws.ReadJSON(&data)
			if err != nil {
				wsConn.failed.Store(true)
				ws.Close()
				if err == io.EOF {
					m.Logger().Debugf("EOF while reading from socket")
					return
				}
				m.Logger().Debugf("While reading from socket: %s", err)
				return
			}

			arr, ok := data.([]interface{})
			if !ok {
				m.HandleRequest(data, &wsConn, nil)
				continue
			}

			if len(arr) == 0 {
				if wsConn.writeJSON([]string{}) != nil {
					wsConn.Close()
				}
			} else {
				m.HandleRequest(data, &wsConn, nil)
			}
		}
	}
}
