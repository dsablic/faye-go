package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dsablic/faye-go/protocol"
)

const LongPollingConnectionPriority = 1

type LongPollingConnection struct {
	responseChan chan []protocol.Message
	Closed       bool
	jsonp        string
}

func NewLongPollingConnection() *LongPollingConnection {
	return &LongPollingConnection{make(chan []protocol.Message, 1), false, ""}
}

func (lp *LongPollingConnection) Send(msgs []protocol.Message) error {
	lp.Closed = true
	lp.responseChan <- msgs
	return nil
}

func (lp *LongPollingConnection) SendJsonp(msgs []protocol.Message, jsonp string) error {
	lp.Closed = true
	lp.jsonp = jsonp
	lp.responseChan <- msgs
	return nil
}

func (lp *LongPollingConnection) IsConnected() bool {
	return !lp.Closed
}

func (lp *LongPollingConnection) Close() {
	lp.Closed = true
}

func (lp LongPollingConnection) Priority() int {
	return LongPollingConnectionPriority
}

func (lp LongPollingConnection) IsSingleShot() bool {
	return true
}

func MakeLongPoll(msgs interface{}, server Server, w http.ResponseWriter) {
	conn := NewLongPollingConnection()
	go func() {
		server.HandleRequest(msgs, conn)
	}()

	responseMsgs := <-conn.responseChan
	bs, err := json.Marshal(responseMsgs)
	if err != nil {
		server.Logger().Warnf("While encoding response msgs: %s", err)
	}

	if conn.jsonp != "" {
		jsonp := fmt.Sprintf("/**/%v(%v)", conn.jsonp, string(bs))
		bs = []byte(jsonp)
		w.Header().Add("Content-Type", "text/javascript")
	} else {
		w.Header().Add("Content-Type", "application/json")
	}

	_, err = w.Write(bs)
	if err != nil {
		server.Logger().Warnf("While writing HTTP response: %s", err)
	}
}
