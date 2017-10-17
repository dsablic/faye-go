package memory

import (
	"sync"

	"github.com/dsablic/faye-go/protocol"
)

type ClientRegister struct {
	mutex   sync.RWMutex
	clients map[string]*protocol.Client
}

func NewClientRegister() *ClientRegister {
	return &ClientRegister{
		clients: make(map[string]*protocol.Client),
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

func (cr *ClientRegister) Reap() uint {
	cr.mutex.RLock()
	dead := []string{}
	for k, v := range cr.clients {
		if v.ShouldReap() {
			dead = append(dead, k)
		}
	}
	count := uint(len(cr.clients) - len(dead))
	cr.mutex.RUnlock()
	if len(dead) > 0 {
		cr.mutex.Lock()
		for _, id := range dead {
			delete(cr.clients, id)
		}
		cr.mutex.Unlock()
	}
	return count
}
