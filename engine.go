package faye

import (
	"fmt"
	"runtime"
	"time"

	"github.com/roncohen/faye-go/memory"
	"github.com/roncohen/faye-go/protocol"
	"github.com/roncohen/faye-go/utils"
	"github.com/satori/go.uuid"
)

type Engine struct {
	clients  *memory.ClientRegister
	register *memory.SubscriptionRegister
	logger   utils.Logger
	ticker   *time.Ticker
	quit     chan struct{}
}

func NewEngine(logger utils.Logger, reapInterval time.Duration) *Engine {
	engine := &Engine{
		clients:  memory.NewClientRegister(logger),
		register: memory.NewSubscriptionRegister(),
		logger:   logger,
		ticker:   time.NewTicker(reapInterval),
		quit:     make(chan struct{}),
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

func (m *Engine) Connect(request protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	timeout := protocol.DEFAULT_ADVICE.Timeout

	response.Update(protocol.Message{
		"advice": protocol.DEFAULT_ADVICE,
	})
	client.Connect(timeout, 0, response, conn)
}

func (m *Engine) SubscribeClient(request protocol.Message, client *protocol.Client) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	subscription := request["subscription"]
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

	client.Queue(response)
}

func (m *Engine) Disconnect(request protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true
	clientId := request.ClientId()
	m.logger.Debugf("Client %s disconnected", clientId)
}

func (m *Engine) Publish(request protocol.Message) {
	requestingClient := m.clients.GetClient(request.ClientId())

	if requestingClient == nil {
		m.logger.Warnf("PUBLISH from unknown client %s", request)
	} else {
		response := m.responseFromRequest(request)
		response["successful"] = true
		data := request["data"]
		channel := request.Channel()

		m.clients.GetClient(request.ClientId()).Queue(response)

		go func() {
			msg := protocol.Message{}
			msg["channel"] = channel.Name()
			msg["data"] = data
			// TODO: Missing ID

			msg.SetClientId(request.ClientId())

			recipients := m.register.GetClients(channel.Expand())
			m.logger.Debugf("PUBLISH from %s on %s to %d recipients", request.ClientId(), channel, len(recipients))
			for _, c := range recipients {
				m.clients.GetClient(c).Queue(msg)
			}
		}()
	}
}

// Publish message directly to client
// msg should have "channel" which the client is expecting, e.g. "/service/echo"
func (m *Engine) PublishFromService(recipientId string, msg protocol.Message) {
	// response["successful"] = true
	m.clients.GetClient(recipientId).Queue(msg)
}

func (m *Engine) Handshake(request protocol.Message, conn protocol.Connection) string {
	newClientId := ""
	version := request["version"].(string)

	response := m.responseFromRequest(request)
	response["successful"] = false

	if version == protocol.BAYEUX_VERSION {
		newClientId = m.NewClient(conn).Id()

		response.Update(map[string]interface{}{
			"clientId":                 newClientId,
			"channel":                  protocol.META_PREFIX + protocol.META_HANDSHAKE_CHANNEL,
			"version":                  protocol.BAYEUX_VERSION,
			"advice":                   protocol.DEFAULT_ADVICE,
			"supportedConnectionTypes": []string{"websocket", "long-polling"},
			"successful":               true,
		})

	} else {
		response["error"] = fmt.Sprintf("Only supported version is '%s'", protocol.BAYEUX_VERSION)
	}

	// Answer directly
	conn.Send([]protocol.Message{response})
	return newClientId
}

func (m *Engine) reap() {
	for {
		select {
		case <-m.ticker.C:
			for _, id := range m.clients.Reap(func(id string) { m.register.RemoveClient(id) }) {
				m.logger.Debugf("Reaping client %s", id)
			}
			m.logStats()
		case <-m.quit:
			m.ticker.Stop()
			return
		}
	}
}

func (m *Engine) logStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.logger.Infof("Clients = %v, Alloc = %v, TotalAlloc = %v, nSys = %v, nNumGC = %v",
		m.clients.Count(), ms.Alloc/1024, ms.TotalAlloc/1024, ms.Sys/1024, ms.NumGC)
}

func (m *Engine) responseFromRequest(request protocol.Message) protocol.Message {
	response := protocol.Message{}
	response["channel"] = request.Channel().Name()
	if reqId, ok := request["id"]; ok {
		response["id"] = reqId.(string)
	}

	return response
}

func (m *Engine) addSubscription(clientId string, subscriptions []string) {
	m.logger.Infof("SUBSCRIBE %s subscription: %v", clientId, subscriptions)
	m.register.AddSubscription(clientId, subscriptions)
}
