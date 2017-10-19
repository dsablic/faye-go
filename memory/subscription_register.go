package memory

import (
	"sync"

	"github.com/dsablic/faye-go/utils"
)

type SubscriptionRegister struct {
	clientByPattern  map[string]*utils.StringSet
	patternsByClient map[string]*utils.StringSet
	mutex            sync.RWMutex
}

func NewSubscriptionRegister() *SubscriptionRegister {
	return &SubscriptionRegister{
		clientByPattern:  make(map[string]*utils.StringSet),
		patternsByClient: make(map[string]*utils.StringSet),
	}
}

func (sr *SubscriptionRegister) AddSubscription(clientId string, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	for _, pattern := range patterns {
		_, ok := sr.clientByPattern[pattern]
		if !ok {
			sr.clientByPattern[pattern] = utils.NewStringSet()
		}
		sr.clientByPattern[pattern].Add(clientId)
	}

	_, ok := sr.patternsByClient[clientId]
	if !ok {
		sr.patternsByClient[clientId] = utils.NewStringSet()
	}
	sr.patternsByClient[clientId].AddMany(patterns)
}

func (sr *SubscriptionRegister) RemoveSubscription(clientId string, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		sr.clientByPattern[pattern].Remove(clientId)
		sr.patternsByClient[clientId].Remove(pattern)
	}
}

func (sr *SubscriptionRegister) GetClients(patterns []string) []string {
	set := utils.NewStringSet()
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	for _, pattern := range patterns {
		if p := sr.clientByPattern[pattern]; p != nil {
			set.AddMany(p.GetAll())
		}
	}
	return set.GetAll()
}

func (sr *SubscriptionRegister) RemoveClient(clientId string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range sr.patternsByClient[clientId].GetAll() {
		sr.clientByPattern[pattern].Remove(clientId)
	}
	delete(sr.patternsByClient, clientId)
}
