package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

const BayeuxVersion = "1.0"

type Advice struct {
	Reconnect string `json:"reconnect"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
}

var DefaultAdvice = Advice{Reconnect: "retry", Interval: 0, Timeout: 25000}

type Message map[string]interface{}

func (m Message) Channel() Channel {
	if ch, ok := m["channel"].(string); ok {
		return Channel{ch}
	}
	return Channel{}
}

func (m Message) ClientId() uint32 {
	if clientId, ok := m["clientId"].(string); ok {
		id, _ := strconv.ParseInt(strings.Replace(clientId, "client-", "", 1), 10, 32)
		return uint32(id)
	}
	return 0
}

func (m Message) Jsonp() string {
	if jsonp, ok := m["jsonp"].(string); ok {
		return jsonp
	}
	return ""
}

func (m Message) SetClientId(clientId uint32) {
	m["clientId"] = fmt.Sprintf("client-%d", clientId)
}

func (m Message) Update(update Message) {
	for k, v := range update {
		m[k] = v
	}
}
