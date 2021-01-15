package memory

import (
	"sync"

	"github.com/uber-go/atomic"
)

type interfaceMap map[interface{}]struct{}

type SubscriptionRegister struct {
	subscriberByPattern      map[string]interfaceMap
	SubscriberByPatternCount *atomic.Uint64
	mutex                    sync.RWMutex
}

func NewSubscriptionRegister() *SubscriptionRegister {
	return &SubscriptionRegister{
		subscriberByPattern:      make(map[string]interfaceMap),
		SubscriberByPatternCount: atomic.NewUint64(0),
	}
}

func (sr *SubscriptionRegister) updateCounts() {
	sr.SubscriberByPatternCount.Store(uint64(len(sr.subscriberByPattern)))
}

func (sr *SubscriptionRegister) AddSubscription(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		if _, ok := sr.subscriberByPattern[pattern]; !ok {
			sr.subscriberByPattern[pattern] = make(interfaceMap)
		}
		sr.subscriberByPattern[pattern][subscriber] = struct{}{}
	}

	sr.updateCounts()
}

func (sr *SubscriptionRegister) RemoveSubscription(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		if p, ok := sr.subscriberByPattern[pattern]; ok {
			delete(p, subscriber)
		}
	}
	sr.updateCounts()
}

func (sr *SubscriptionRegister) GetSubscribers(patterns []string) []interface{} {
	arr := make([]interface{}, 0)
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	for _, pattern := range patterns {
		if subscribers, ok := sr.subscriberByPattern[pattern]; ok {
			for subscriber := range subscribers {
				arr = append(arr, subscriber)
			}
		}
	}
	return arr
}

func (sr *SubscriptionRegister) RemoveClient(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	for _, pattern := range patterns {
		if subscribers, ok := sr.subscriberByPattern[pattern]; ok {
			delete(subscribers, subscriber)
			if len(subscribers) == 0 {
				delete(sr.subscriberByPattern, pattern)
			}
		}
	}
	sr.updateCounts()
}
