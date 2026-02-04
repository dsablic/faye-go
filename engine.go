package faye

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/dsablic/faye-go/memory"
	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
)

type Counters struct {
	Published           uint
	Sent                uint
	Clients             uint
	Failed              uint
	SubscriberByPattern uint
}

type Engine struct {
	statistics      chan Counters
	clients         *memory.ClientRegister
	logger          utils.Logger
	published       uint64
	reapInterval    time.Duration
	ticker          *time.Ticker
	currentClientID uint32
}

func NewEngine(logger utils.Logger, reapInterval time.Duration, statistics chan Counters) *Engine {
	engine := &Engine{
		statistics:      statistics,
		clients:         memory.NewClientRegister(),
		logger:          logger,
		published:       0,
		reapInterval:    reapInterval,
		ticker:          time.NewTicker(reapInterval),
		currentClientID: 0,
	}
	go engine.reap()
	return engine
}

func (m *Engine) GetClient(clientId uint32) *protocol.Client {
	return m.clients.GetClient(clientId)
}

func (m *Engine) NewClient(conn protocol.Connection) *protocol.Client {
	atomic.CompareAndSwapUint32(&m.currentClientID, math.MaxUint32, 0)
	newClient := protocol.NewClient(
		atomic.AddUint32(&m.currentClientID, 1),
		m.logger)
	m.clients.AddClient(newClient)
	return newClient
}

func (m *Engine) Connect(request *protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	timeout := protocol.DefaultAdvice.Timeout

	response.Update(protocol.Message{
		"advice": protocol.DefaultAdvice,
	})
	client.Connect(timeout, 0, response, conn)
}

func (m *Engine) subscriptionResponse(request *protocol.Message) (protocol.Message, []string) {
	response := m.responseFromRequest(request)
	response["successful"] = true

	subscription := (*request)["subscription"]
	response["subscription"] = subscription

	var subs []string
	switch v := subscription.(type) {
	case []string:
		subs = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				subs = append(subs, s)
			}
		}
	case string:
		subs = []string{v}
	}

	return response, subs
}

func (m *Engine) SubscribeClient(request *protocol.Message, client *protocol.Client) {
	response, subs := m.subscriptionResponse(request)
	patterns := []string{}
	for _, s := range subs {
		if !protocol.NewChannel(s).IsService() {
			m.logger.Debugf("SUBSCRIBE %d subscription: %v", client.Id(), s)
			patterns = append(patterns, s)
		}
	}
	client.Subscribe(patterns)
	m.clients.AddSubscription(client, patterns)
	client.Send(response, request.Jsonp())
}

func (m *Engine) UnsubscribeClient(request *protocol.Message, client *protocol.Client) {
	response, subs := m.subscriptionResponse(request)
	patterns := []string{}
	for _, s := range subs {
		if !protocol.NewChannel(s).IsService() {
			m.logger.Debugf("UNSUBSCRIBE %d subscription: %v", client.Id(), s)
			patterns = append(patterns, s)
		}
	}
	client.Unsubscribe(patterns)
	m.clients.RemoveSubscription(client, patterns)
	client.Send(response, request.Jsonp())
}

func (m *Engine) Disconnect(request *protocol.Message, client *protocol.Client, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true
	clientId := request.ClientId()
	m.logger.Debugf("Client %d disconnected", clientId)
	_ = response
}

func (m *Engine) Publish(request *protocol.Message, conn protocol.Connection) {
	response := m.responseFromRequest(request)
	response["successful"] = true
	data := (*request)["data"]
	channel := request.Channel()

	if jsonp := request.Jsonp(); jsonp != "" {
		conn.SendJsonp([]protocol.Message{response}, jsonp)
	} else {
		conn.Send([]protocol.Message{response})
	}

	msg := protocol.Message{}
	msg["channel"] = channel.Name()
	msg["data"] = data
	msg.SetClientId(request.ClientId())
	m.logger.Debugf("PUBLISH from %d on %s", request.ClientId(), channel)
	m.clients.Publish(msg)
	atomic.AddUint64(&m.published, 1)
}

func (m *Engine) Handshake(request *protocol.Message, conn protocol.Connection) uint32 {
	var newClientId uint32

	version, _ := (*request)["version"].(string)

	response := m.responseFromRequest(request)
	response["successful"] = false
	if version == protocol.BayeuxVersion {
		newClientId = m.NewClient(conn).Id()
		update := protocol.Message{
			"channel":                  protocol.MetaPrefix + protocol.MetaHandshakeChannel,
			"version":                  protocol.BayeuxVersion,
			"advice":                   protocol.DefaultAdvice,
			"supportedConnectionTypes": []string{"websocket"},
			"successful":               true,
		}
		update.SetClientId(newClientId)
		response.Update(update)
	} else {
		response["error"] = fmt.Sprintf("Only supported version is '%s'", protocol.BayeuxVersion)
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
		c.SubscriberByPattern = uint(registerCounters.SubscriberByPatternCount)
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
	if reqId, ok := (*request)["id"].(string); ok {
		response["id"] = reqId
	}

	return response
}
