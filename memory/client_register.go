package memory

import (
	"sync"

	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
)

type ClientRegister struct {
	logger  utils.Logger
	mutex   sync.RWMutex
	clients map[string]*protocol.Client
}

func NewClientRegister(logger utils.Logger) *ClientRegister {
	return &ClientRegister{
		logger:  logger,
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

func (cr *ClientRegister) Count() int {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	return len(cr.clients)
}

func (cr *ClientRegister) Reap(onRemove func(id string)) []string {
	cr.mutex.RLock()
	dead := []string{}
	for k, v := range cr.clients {
		if v.ShouldReap() {
			dead = append(dead, k)
		}
	}
	cr.mutex.RUnlock()
	if len(dead) > 0 {
		cr.mutex.Lock()
		for _, id := range dead {
			onRemove(id)
			delete(cr.clients, id)
		}
		cr.mutex.Unlock()
	}

	return dead
}
