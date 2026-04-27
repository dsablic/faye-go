package memory

import (
	"errors"
	"sync"

	"github.com/dsablic/faye-go/protocol"
)

var ErrMaxClientsReached = errors.New("maximum number of clients reached")

type ClientRegisterCounters struct {
	TotalFailed              uint64
	TotalSent                uint64
	Clients                  uint
	SubscriberByPatternCount uint64
}

type ClientRegister struct {
	mutex         sync.RWMutex
	clients       map[uint32]*protocol.Client
	subscriptions *SubscriptionRegister
	maxClients    int
}

func NewClientRegister() *ClientRegister {
	return &ClientRegister{
		clients:       make(map[uint32]*protocol.Client),
		subscriptions: NewSubscriptionRegister(),
		maxClients:    0,
	}
}

func (cr *ClientRegister) SetMaxClients(max int) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	cr.maxClients = max
}

func (cr *ClientRegister) AddClient(client *protocol.Client) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	id := client.Id()
	if old, ok := cr.clients[id]; ok {
		old.Close()
		delete(cr.clients, id)
	} else if cr.maxClients > 0 && len(cr.clients) >= cr.maxClients {
		return ErrMaxClientsReached
	}
	cr.clients[id] = client
	return nil
}

func (cr *ClientRegister) GetClient(clientId uint32) *protocol.Client {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	client, ok := cr.clients[clientId]
	if ok {
		return client
	}
	return nil
}

func (cr *ClientRegister) RemoveClient(client *protocol.Client) {
	cr.subscriptions.RemoveSubscription(client, client.Subscriptions())
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	delete(cr.clients, client.Id())
}

func (cr *ClientRegister) AddSubscription(client *protocol.Client, patterns []string) {
	cr.subscriptions.AddSubscription(client, patterns)
}

func (cr *ClientRegister) RemoveSubscription(client *protocol.Client, patterns []string) {
	cr.subscriptions.RemoveSubscription(client, patterns)
}

func (cr *ClientRegister) Publish(msg protocol.Message) {
	patterns := msg.Channel().Expand()
	subscribers := cr.subscriptions.GetSubscribers(patterns)
	if len(subscribers) == 0 {
		return
	}

	clients := make([]*protocol.Client, 0, len(subscribers))
	for _, sub := range subscribers {
		if client, ok := sub.(*protocol.Client); ok {
			clients = append(clients, client)
		}
	}

	if len(clients) == 0 {
		return
	}

	go func(clients []*protocol.Client, msg protocol.Message) {
		for _, client := range clients {
			client.Send(msg, "")
		}
	}(clients, msg)
}

func (cr *ClientRegister) Reap() *ClientRegisterCounters {
	totals := ClientRegisterCounters{0, 0, 0, 0}
	cr.mutex.RLock()
	totals.SubscriberByPatternCount = cr.subscriptions.SubscriberByPatternCount.Load()
	dead := []uint32{}
	for id, client := range cr.clients {
		if client.ShouldReap() {
			cr.subscriptions.RemoveSubscription(client, client.Subscriptions())
			dead = append(dead, id)
		}
		c := client.ResetCounters()
		totals.TotalFailed += c.Failed
		totals.TotalSent += c.Sent
	}
	totals.Clients = uint(len(cr.clients) - len(dead))
	cr.mutex.RUnlock()
	if len(dead) > 0 {
		cr.mutex.Lock()
		for _, id := range dead {
			delete(cr.clients, id)
		}
		cr.mutex.Unlock()
	}
	return &totals
}
