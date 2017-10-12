package protocol

const BAYEUX_VERSION = "1.0"
const UNKNOWN_CLIENT = "__UNKNOWN__"
const UNKNOWN_CHANNEL = "__UNKNOWN__"

type Advice struct {
	Reconnect string `json:"reconnect"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
}

var DEFAULT_ADVICE = Advice{Reconnect: "retry", Interval: 0, Timeout: 10000}

type Message map[string]interface{}

func (m Message) Channel() Channel {
	if ch, ok := m["channel"].(string); ok {
		return Channel{ch}
	}
	return Channel{UNKNOWN_CHANNEL}
}

func (m Message) ClientId() string {
	if id, ok := m["clientId"].(string); ok {
		return id
	}
	return UNKNOWN_CLIENT
}

func (m Message) SetClientId(clientId string) {
	m["clientId"] = clientId
}

func (m Message) Update(update Message) {
	for k, v := range update {
		m[k] = v
	}
}
