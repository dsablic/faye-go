package memory

import (
	"sync"

	"go.uber.org/atomic"
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
		if subscribers, ok := sr.subscriberByPattern[pattern]; ok {
			delete(subscribers, subscriber)
			if len(subscribers) == 0 {
				delete(sr.subscriberByPattern, pattern)
			}
		}
	}
	sr.updateCounts()
}

func (sr *SubscriptionRegister) GetSubscribers(patterns []string) []interface{} {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	arr := make([]interface{}, 0)
	var seen interfaceMap
	for _, pattern := range patterns {
		subscribers, ok := sr.subscriberByPattern[pattern]
		if !ok {
			continue
		}
		if seen == nil && len(arr) == 0 {
			for subscriber := range subscribers {
				arr = append(arr, subscriber)
			}
			continue
		}
		if seen == nil {
			seen = make(interfaceMap, len(arr))
			for _, s := range arr {
				seen[s] = struct{}{}
			}
		}
		for subscriber := range subscribers {
			if _, exists := seen[subscriber]; !exists {
				seen[subscriber] = struct{}{}
				arr = append(arr, subscriber)
			}
		}
	}
	return arr
}
