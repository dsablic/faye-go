package faye

import (
	"fmt"
	"sync/atomic"
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
	statistics   chan Counters
	clients      *memory.ClientRegister
	logger       utils.Logger
	published    uint64
	reapInterval time.Duration
	ticker       *time.Ticker
}

func NewEngine(logger utils.Logger, reapInterval time.Duration, statistics chan Counters) *Engine {
	engine := &Engine{
		statistics:   statistics,
		clients:      memory.NewClientRegister(),
		logger:       logger,
		published:    0,
		reapInterval: reapInterval,
		ticker:       time.NewTicker(reapInterval),
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
		if !protocol.NewChannel(s).IsService() {
			m.logger.Infof("SUBSCRIBE %s subscription: %v", client.Id(), s)
			m.clients.AddSubscription(client, []string{s})
		}
	}

	client.Send(response, request.Jsonp())
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
	} else {
		requestingClient.Send(response, request.Jsonp())
	}

	msg := protocol.Message{}
	msg["channel"] = channel.Name()
	msg["data"] = data
	msg.SetClientId(request.ClientId())
	m.logger.Debugf("PUBLISH from %s on %s", request.ClientId(), channel)
	go func() {
		m.clients.Publish(msg)
		atomic.AddUint64(&m.published, 1)
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

	if jsonp := request.Jsonp(); jsonp != "" {
		conn.SendJsonp([]protocol.Message{response}, jsonp)
	} else {
		conn.Send([]protocol.Message{response})
	}
	return newClientId
}

func (m *Engine) reap() {
	for range m.ticker.C {
		registerCounters := m.clients.Reap()
		c := Counters{}
		c.Clients = registerCounters.Clients
		c.Failed = uint(registerCounters.TotalFailed)
		c.Sent = uint(registerCounters.TotalSent)
		c.Published = uint(atomic.SwapUint64(&m.published, 0))
		select {
		case m.statistics <- c:
		default:
			m.logger.Errorf("Statistics channel full")
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
