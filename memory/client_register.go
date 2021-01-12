package memory

import (
	"sync"

	"github.com/dsablic/faye-go/protocol"
)

type ClientRegisterCounters struct {
	TotalFailed               uint64
	TotalSent                 uint64
	Clients                   uint
	PatternsBySubscriberCount uint64
	SubscriberByPatternCount  uint64
}

type ClientRegister struct {
	mutex         sync.RWMutex
	clients       map[string]*protocol.Client
	subscriptions *SubscriptionRegister
}

func NewClientRegister() *ClientRegister {
	return &ClientRegister{
		clients:       make(map[string]*protocol.Client),
		subscriptions: NewSubscriptionRegister(),
	}
}

func (cr *ClientRegister) AddClient(client *protocol.Client) {
	cr.mutex.Lock()
	cr.clients[client.Id()] = client
	cr.mutex.Unlock()
}

func (cr *ClientRegister) GetClient(clientId string) *protocol.Client {
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
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	for _, client := range cr.subscriptions.GetSubscribers(msg.Channel().Expand()) {
		client.(*protocol.Client).Send(msg, "")
	}
}

func (cr *ClientRegister) Reap() *ClientRegisterCounters {
	totals := ClientRegisterCounters{0, 0, 0, 0, 0}
	cr.mutex.RLock()
	totals.PatternsBySubscriberCount = cr.subscriptions.PatternsBySubscriberCount.Load()
	totals.SubscriberByPatternCount = cr.subscriptions.SubscriberByPatternCount.Load()
	dead := []string{}
	for id, client := range cr.clients {
		if client.ShouldReap() {
			cr.subscriptions.RemoveClient(client)
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
