package memory

import (
	"sync"

	"github.com/dsablic/faye-go/utils"
)

type SubscriptionRegister struct {
	subscriberByPattern  map[string]*utils.ValueSet
	patternsBySubscriber map[interface{}]*utils.StringSet
	mutex                sync.RWMutex
}

func NewSubscriptionRegister() *SubscriptionRegister {
	return &SubscriptionRegister{
		subscriberByPattern:  make(map[string]*utils.ValueSet),
		patternsBySubscriber: make(map[interface{}]*utils.StringSet),
	}
}

func (sr *SubscriptionRegister) AddSubscription(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		if _, ok := sr.subscriberByPattern[pattern]; !ok {
			sr.subscriberByPattern[pattern] = utils.NewValueSet()
		}
		sr.subscriberByPattern[pattern].Add(subscriber)
	}

	if _, ok := sr.patternsBySubscriber[subscriber]; !ok {
		sr.patternsBySubscriber[subscriber] = utils.NewStringSet()
	}

	sr.patternsBySubscriber[subscriber].AddMany(patterns)
}

func (sr *SubscriptionRegister) RemoveSubscription(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		if p, ok := sr.subscriberByPattern[pattern]; ok {
			p.Remove(subscriber)
		}
		if s, ok := sr.patternsBySubscriber[subscriber]; ok {
			s.Remove(pattern)
		}
	}
}

func (sr *SubscriptionRegister) GetSubscribers(patterns []string) []interface{} {
	set := utils.NewValueSet()
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	for _, pattern := range patterns {
		if p, ok := sr.subscriberByPattern[pattern]; ok {
			set.AddMany(p.GetAll())
		}
	}
	return set.GetAll()
}

func (sr *SubscriptionRegister) RemoveClient(subscriber interface{}) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if p, ok := sr.patternsBySubscriber[subscriber]; ok {
		for _, pattern := range p.GetAll() {
			if s, ok := sr.subscriberByPattern[pattern]; ok {
				s.Remove(subscriber)
			}
		}
	}
	delete(sr.patternsBySubscriber, subscriber)
}
