package memory

import (
	"sync"

	"github.com/dsablic/faye-go/protocol"
)

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
}

func NewClientRegister() *ClientRegister {
	return &ClientRegister{
		clients:       make(map[uint32]*protocol.Client),
		subscriptions: NewSubscriptionRegister(),
	}
}

func (cr *ClientRegister) AddClient(client *protocol.Client) {
	cr.mutex.Lock()
	id := client.Id()
	if old, ok := cr.clients[id]; ok {
		old.Close()
		delete(cr.clients, id)
	}
	cr.clients[id] = client
	cr.mutex.Unlock()
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
	go func() {
		cr.mutex.RLock()
		defer cr.mutex.RUnlock()
		for _, client := range subscribers {
			client.(*protocol.Client).Send(msg, "")
		}
	}()
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
