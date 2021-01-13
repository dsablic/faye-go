package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dsablic/faye-go/protocol"
	"github.com/uber-go/atomic"
)

type LongPollingConnection struct {
	responseChan chan []protocol.Message
	Closed       *atomic.Bool
	jsonp        *atomic.String
}

func NewLongPollingConnection() *LongPollingConnection {
	return &LongPollingConnection{make(chan []protocol.Message, 1), atomic.NewBool(false), atomic.NewString("")}
}

func (lp *LongPollingConnection) enqueueMessages(msgs []protocol.Message) error {
	select {
	case lp.responseChan <- msgs:
		return nil
	default:
		return errors.New("Response channel is full")
	}
}

func (lp *LongPollingConnection) Send(msgs []protocol.Message) error {
	lp.Close()
	return lp.enqueueMessages(msgs)
}

func (lp *LongPollingConnection) SendJsonp(msgs []protocol.Message, jsonp string) error {
	lp.Close()
	lp.jsonp.Store(jsonp)
	return lp.enqueueMessages(msgs)
}

func (lp *LongPollingConnection) IsConnected() bool {
	return !lp.Closed.Load()
}

func (lp *LongPollingConnection) Close() {
	lp.Closed.Store(true)
}

func (lp *LongPollingConnection) IsSingleShot() bool {
	return true
}

func MakeLongPoll(msgs interface{}, server Server, w http.ResponseWriter) {
	conn := NewLongPollingConnection()
	done := make(chan bool, 1)
	go func() {
		server.HandleRequest(msgs, conn)
		done <- true
	}()

	select {
	case responseMsgs := <-conn.responseChan:
		if bs, err := json.Marshal(responseMsgs); err != nil {
			server.Logger().Warnf("While encoding response msgs: %s", err)
		} else {
			connJsonp := conn.jsonp.Load()
			if connJsonp != "" {
				jsonp := fmt.Sprintf("/**/%v(%v)", connJsonp, string(bs))
				bs = []byte(jsonp)
				w.Header().Add("Content-Type", "text/javascript")
			} else {
				w.Header().Add("Content-Type", "application/json")
			}
			if _, err := w.Write(bs); err != nil {
				server.Logger().Warnf("While writing HTTP response: %s", err)
			}
		}
	case <-done:
		server.Logger().Debugf("No response")
	}
}
