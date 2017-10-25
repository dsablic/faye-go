package transport

import (
	"errors"
	"io"
	"sync"

	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
	"github.com/gorilla/websocket"
)

const WebSocketConnectionPriority = 10

type Server interface {
	HandleRequest(interface{}, protocol.Connection)
	Logger() utils.Logger
}

type WebSocketConnection struct {
	ws     *websocket.Conn
	failed bool
	mutex  sync.RWMutex
}

func (wc *WebSocketConnection) Send(msgs []protocol.Message) error {
	wc.mutex.Lock()
	defer wc.mutex.Unlock()
	err := wc.ws.WriteJSON(msgs)
	if err != nil {
		wc.failed = true
	}
	return err
}

func (wc *WebSocketConnection) SendJsonp(msgs []protocol.Message, _ string) error {
	return errors.New("Jsonp is not supported over websockets")
}

func (wc *WebSocketConnection) IsConnected() bool {
	wc.mutex.RLock()
	defer wc.mutex.RUnlock()
	return !wc.failed
}

func (wc *WebSocketConnection) Close() {
	wc.ws.Close()
}

func (wc *WebSocketConnection) Priority() int {
	return WebSocketConnectionPriority
}

func (wc *WebSocketConnection) IsSingleShot() bool {
	return false
}

func WebsocketServer(m Server) func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		var data interface{}
		wsConn := WebSocketConnection{ws: ws, failed: false}
		for {
			err := ws.ReadJSON(&data)
			if err != nil {
				wsConn.mutex.Lock()
				wsConn.failed = true
				wsConn.mutex.Unlock()
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
				ws.WriteJSON([]string{})
				wsConn.mutex.Unlock()
			} else {
				m.HandleRequest(data, &wsConn)
			}
		}
	}
}
