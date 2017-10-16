package faye

import (
	"fmt"
	"time"

	"github.com/dsablic/faye-go/memory"
	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
	"github.com/satori/go.uuid"
)

type Counters struct {
	Published uint
	Sent      uint
	Clients   uint
	Failed    uint
}

type Engine struct {
	clients      *memory.ClientRegister
	register     *memory.SubscriptionRegister
	logger       utils.Logger
	counters     Counters
	Statistics   chan Counters
	reapInterval time.Duration
	ticker       *time.Ticker
	quit         chan struct{}
}

func NewEngine(logger utils.Logger, reapInterval time.Duration) *Engine {
	engine := &Engine{
		clients:      memory.NewClientRegister(),
		register:     memory.NewSubscriptionRegister(),
		counters:     Counters{0, 0, 0, 0},
		logger:       logger,
		Statistics:   make(chan Counters),
		reapInterval: reapInterval,
		ticker:       time.NewTicker(reapInterval),
		quit:         make(chan struct{}),
	}
	go engine.reap()
	return engine
}

func (m *Engine) GetClient(clientId string) *protocol.Client {
	return m.clients.GetClient(clientId)
}

func (m *Engine) NewClient(conn protocol.Connection) *protocol.Client {
	newClient := protocol.NewClient(
		uuid.NewV4().String(),
		m.logger)
	m.clients.AddClient(newClient)
	return newClient
}

func (m *Engine) Connect(request *protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	timeout := protocol.DEFAULT_ADVICE.Timeout

	response.Update(protocol.Message{
		"advice": protocol.DEFAULT_ADVICE,
	})
	client.Connect(timeout, 0, response, conn)
}

func (m *Engine) SubscribeClient(request *protocol.Message, client *protocol.Client) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	subscription := (*request)["subscription"]
	response["subscription"] = subscription

	var subs []string
	switch subscription.(type) {
	case []string:
		subs = subscription.([]string)
	case string:
		subs = []string{subscription.(string)}
	}

	for _, s := range subs {
		// Do not register clients subscribing to a service channel
		// They will be answered directly instead of through the normal subscription system
		if !protocol.NewChannel(s).IsService() {
			m.addSubscription(client.Id(), []string{s})
		}
	}

	client.Send(response)
}

func (m *Engine) Disconnect(request *protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true
	clientId := request.ClientId()
	m.logger.Debugf("Client %s disconnected", clientId)
}

func (m *Engine) Publish(request *protocol.Message, conn protocol.Connection) {
	requestingClient := m.clients.GetClient(request.ClientId())

	response := m.responseFromRequest(request)
	response["successful"] = true
	data := (*request)["data"]
	channel := request.Channel()

	if requestingClient == nil {
		conn.Send([]protocol.Message{response})
		conn.Close()
	} else {
		requestingClient.Send(response)
	}

	go func() {
		msg := protocol.Message{}
		msg["channel"] = channel.Name()
		msg["data"] = data
		// TODO: Missing ID

		msg.SetClientId(request.ClientId())

		recipients := m.register.GetClients(channel.Expand())
		m.counters.Published++
		m.logger.Debugf("PUBLISH from %s on %s to %d recipients", request.ClientId(), channel, len(recipients))
		for _, c := range recipients {
			if client := m.clients.GetClient(c); client != nil {
				if client.Send(msg) {
					m.counters.Sent++
				} else {
					m.counters.Failed++
				}
			}
		}
	}()
}

func (m *Engine) Handshake(request *protocol.Message, conn protocol.Connection) string {
	newClientId := ""
	version := (*request)["version"].(string)

	response := m.responseFromRequest(request)
	response["successful"] = false

	if version == protocol.BAYEUX_VERSION {
		newClientId = m.NewClient(conn).Id()

		response.Update(protocol.Message{
			"clientId":                 newClientId,
			"channel":                  protocol.META_PREFIX + protocol.META_HANDSHAKE_CHANNEL,
			"version":                  protocol.BAYEUX_VERSION,
			"advice":                   protocol.DEFAULT_ADVICE,
			"supportedConnectionTypes": []string{"websocket"},
			"successful":               true,
		})

	} else {
		response["error"] = fmt.Sprintf("Only supported version is '%s'", protocol.BAYEUX_VERSION)
	}

	conn.Send([]protocol.Message{response})
	return newClientId
}

func (m *Engine) reap() {
	for {
		select {
		case <-m.ticker.C:
			m.counters.Clients = uint(m.clients.Reap(func(id string) {
				m.logger.Debugf("Reaping client %s", id)
				m.register.RemoveClient(id)
			}))
			m.Statistics <- m.counters
			m.counters = Counters{0, 0, 0, 0}
		case <-m.quit:
			m.ticker.Stop()
			return
		}
	}
}

func (m *Engine) responseFromRequest(request *protocol.Message) protocol.Message {
	response := protocol.Message{}
	response["channel"] = request.Channel().Name()
	if reqId, ok := (*request)["id"]; ok {
		response["id"] = reqId.(string)
	}

	return response
}

func (m *Engine) addSubscription(clientId string, subscriptions []string) {
	m.logger.Infof("SUBSCRIBE %s subscription: %v", clientId, subscriptions)
	m.register.AddSubscription(clientId, subscriptions)
}
