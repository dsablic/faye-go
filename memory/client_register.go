package memory

import (
	"sync"

	"github.com/dsablic/faye-go/protocol"
)

type ClientRegisterCounters struct {
	TotalFailed uint64
	TotalSent   uint64
	Clients     uint
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

func (cr *ClientRegister) AddSubscription(clientId string, patterns []string) {
	cr.subscriptions.AddSubscription(clientId, patterns)
}

func (cr *ClientRegister) Publish(msg protocol.Message) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	for _, clientId := range cr.subscriptions.GetClients(msg.Channel().Expand()) {
		cr.clients[clientId].Send(msg)
	}
}

func (cr *ClientRegister) Reap() *ClientRegisterCounters {
	totals := ClientRegisterCounters{0, 0, 0}
	cr.mutex.RLock()
	dead := []string{}
	for k, v := range cr.clients {
		if v.ShouldReap() {
			dead = append(dead, k)
		}
		c := v.ResetCounters()
		totals.TotalFailed += c.Failed
		totals.TotalSent += c.Sent
	}
	totals.Clients = uint(len(cr.clients) - len(dead))
	cr.mutex.RUnlock()
	if len(dead) > 0 {
		cr.mutex.Lock()
		for _, id := range dead {
			cr.subscriptions.RemoveClient(id)
			delete(cr.clients, id)
		}
		cr.mutex.Unlock()
	}
	return &totals
}
