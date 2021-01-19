package protocol

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dsablic/faye-go/utils"
)

type ClientCounters struct {
	Failed uint64
	Sent   uint64
}

type stringMap map[string]struct{}

type Client struct {
	clientId      uint32
	connection    Connection
	responseMsg   Message
	mutex         sync.RWMutex
	created       time.Time
	logger        utils.Logger
	counters      ClientCounters
	subscriptions stringMap
}

func NewClient(clientId uint32, logger utils.Logger) *Client {
	return &Client{
		clientId:      clientId,
		created:       time.Now(),
		logger:        logger,
		counters:      ClientCounters{0, 0},
		subscriptions: stringMap{},
	}
}

func (c *Client) Id() uint32 {
	return c.clientId
}

func (c *Client) Connect(timeout int, interval int, responseMsg Message, connection Connection) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if timeout > 0 {
		go func() {
			time.Sleep(time.Duration(timeout) * time.Millisecond)
			if c.isConnected() {
				connection.Send([]Message{responseMsg})
			} else {
				c.logger.Debugf("No longer connected %d", c.clientId)
			}
		}()
	}
	c.responseMsg = responseMsg
}

func (c *Client) SetConnection(connection Connection) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.connection = connection
}

func (c *Client) ShouldReap() bool {
	return !c.isConnected()
}

func (c *Client) ResetCounters() ClientCounters {
	return ClientCounters{
		Sent:   atomic.SwapUint64(&c.counters.Sent, 0),
		Failed: atomic.SwapUint64(&c.counters.Failed, 0),
	}
}

func (c *Client) Close() {
	if c.isConnected() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.connection.Close()
	}
}

func (c *Client) Send(msg Message, jsonp string) bool {
	if c.isConnected() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		msgs := []Message{msg}
		if c.connection.IsSingleShot() {
			msgs = append(msgs, c.responseMsg)
		}
		c.logger.Debugf("Sending %d msgs to %d on %s", len(msgs), c.clientId, reflect.TypeOf(c.connection))

		var err error

		if jsonp != "" {
			err = c.connection.SendJsonp(msgs, jsonp)
		} else {
			err = c.connection.Send(msgs)
		}

		if err != nil {
			c.logger.Debugf("Was unable to send %d messages to %d", len(msgs), c.clientId)
			c.connection.Close()
			atomic.AddUint64(&c.counters.Failed, 1)
			return false
		}

		atomic.AddUint64(&c.counters.Sent, 1)
		return true
	}

	c.logger.Debugf("Not connected for %d", c.clientId)
	atomic.AddUint64(&c.counters.Failed, 1)
	return false
}

func (c *Client) Subscribe(patterns []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, pattern := range patterns {
		c.subscriptions[pattern] = struct{}{}
	}
}

func (c *Client) Unsubscribe(patterns []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, pattern := range patterns {
		if _, ok := c.subscriptions[pattern]; ok {
			delete(c.subscriptions, pattern)
		}
	}
}

func (c *Client) Subscriptions() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	patterns := make([]string, 0, len(c.subscriptions))
	for k := range c.subscriptions {
		patterns = append(patterns, k)
	}

	return patterns
}

func (c *Client) isConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connection != nil && c.connection.IsConnected()
}
