package memory

import (
	"sync"

	"github.com/uber-go/atomic"
)

type void struct{}
type interfaceMap map[interface{}]void
type stringMap map[string]void

type SubscriptionRegister struct {
	subscriberByPattern       map[string]interfaceMap
	patternsBySubscriber      map[interface{}]stringMap
	SubscriberByPatternCount  *atomic.Uint64
	PatternsBySubscriberCount *atomic.Uint64
	mutex                     sync.RWMutex
}

func NewSubscriptionRegister() *SubscriptionRegister {
	return &SubscriptionRegister{
		subscriberByPattern:       make(map[string]interfaceMap),
		SubscriberByPatternCount:  atomic.NewUint64(0),
		patternsBySubscriber:      make(map[interface{}]stringMap),
		PatternsBySubscriberCount: atomic.NewUint64(0),
	}
}

func (sr *SubscriptionRegister) updateCounts() {
	sr.PatternsBySubscriberCount.Store(uint64(len(sr.patternsBySubscriber)))
	sr.SubscriberByPatternCount.Store(uint64(len(sr.subscriberByPattern)))
}

func (sr *SubscriptionRegister) AddSubscription(subscriber interface{}, patterns []string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, ok := sr.patternsBySubscriber[subscriber]; !ok {
		sr.patternsBySubscriber[subscriber] = make(stringMap)
	}

	for _, pattern := range patterns {
		if _, ok := sr.subscriberByPattern[pattern]; !ok {
			sr.subscriberByPattern[pattern] = make(interfaceMap)
		}
		sr.subscriberByPattern[pattern][subscriber] = void{}
		sr.patternsBySubscriber[subscriber][pattern] = void{}
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
		if s, ok := sr.patternsBySubscriber[subscriber]; ok {
			delete(s, pattern)
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

func (sr *SubscriptionRegister) RemoveClient(subscriber interface{}) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if patterns, ok := sr.patternsBySubscriber[subscriber]; ok {
		for pattern := range patterns {
			if subscribers, ok := sr.subscriberByPattern[pattern]; ok {
				delete(subscribers, subscriber)
				if len(subscribers) == 0 {
					delete(sr.subscriberByPattern, pattern)
				}
			}
		}
	}
	delete(sr.patternsBySubscriber, subscriber)
	sr.updateCounts()
}
