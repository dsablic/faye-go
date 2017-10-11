package protocol

const BAYEUX_VERSION = "1.0"

type Advice struct {
	Reconnect string `json:"reconnect"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
}

var DEFAULT_ADVICE = Advice{Reconnect: "retry", Interval: 0, Timeout: 10000}

type Message map[string]interface{}

func (m Message) Channel() Channel {
	return Channel{m["channel"].(string)}
}

func (m Message) ClientId() string {
	return m["clientId"].(string)
}

func (m Message) SetClientId(clientId string) {
	m["clientId"] = clientId
}

func (m Message) Update(update map[string]interface{}) {
	for k, v := range update {
		m[k] = v
	}
}
