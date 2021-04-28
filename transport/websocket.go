package transport

import (
	"errors"
	"io"
	"sync"

	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
)

type Server interface {
	HandleRequest(interface{}, protocol.Connection)
	Logger() utils.Logger
}

type WebSocketConnection struct {
	ws     *websocket.Conn
	failed *atomic.Bool
	mutex  sync.RWMutex
}

func (wc *WebSocketConnection) Send(msgs []protocol.Message) error {
	if !wc.IsConnected() {
		return errors.New("Not connected")
	}
	wc.mutex.Lock()
	err := wc.ws.WriteJSON(msgs)
	wc.mutex.Unlock()
	if err != nil {
		wc.failed.Store(true)
	}
	return err
}

func (wc *WebSocketConnection) SendJsonp(msgs []protocol.Message, _ string) error {
	return errors.New("Jsonp is not supported over websockets")
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

			if arr := data.([]interface{}); len(arr) == 0 {
				wsConn.mutex.Lock()
				if ws.WriteJSON([]string{}) != nil {
					wsConn.Close()
				}
				wsConn.mutex.Unlock()
			} else {
				m.HandleRequest(data, &wsConn)
			}
		}
	}
}
