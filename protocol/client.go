package protocol

import (
	"reflect"
	"sync"
	"time"

	"github.com/dsablic/faye-go/utils"
)

type Session struct {
	conn     Connection
	timeout  int
	response Message
	client   *Client
	started  time.Time
	logger   utils.Logger
}

func NewSession(client *Client, conn Connection, timeout int, response Message, logger utils.Logger) *Session {
	session := Session{conn, timeout, response, client, time.Now(), logger}
	if timeout > 0 {
		go func() {
			time.Sleep(time.Duration(timeout) * time.Millisecond)
			session.End()
		}()
	}
	return &session
}

func (s *Session) End() {
	if s.client.isConnected() {
		s.conn.Send([]Message{s.response})
	} else {
		s.logger.Debugf("No longer connected %s", s.client.clientId)
	}
}

type CounterChannels struct {
	Failed    chan uint
	Sent      chan uint
	Published chan uint
}

type Client struct {
	clientId      string
	subscriptions *utils.StringSet
	Messages      chan Message
	connection    Connection
	responseMsg   Message
	mutex         sync.RWMutex
	lastSession   *Session
	created       time.Time
	logger        utils.Logger
	quit          chan bool
}

func NewClient(clientId string, logger utils.Logger) *Client {
	return &Client{
		subscriptions: utils.NewStringSet(),
		clientId:      clientId,
		created:       time.Now(),
		logger:        logger,
		Messages:      make(chan Message),
		quit:          make(chan bool),
	}
}

func (c *Client) Id() string {
	return c.clientId
}

func (c *Client) Connect(timeout int, interval int, responseMsg Message, connection Connection, ch *CounterChannels) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lastSession = NewSession(c, connection, timeout, responseMsg, c.logger)
	c.responseMsg = responseMsg
	go func() {
		for {
			select {
			case m := <-c.Messages:
				{
					if c.isSubscribed(m.Channel().Expand()) {
						if c.Send(m) {
							ch.Sent <- 1
						} else {
							ch.Failed <- 1
						}
					}
				}
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *Client) Release() {
	defer c.mutex.RUnlock()
	c.mutex.RLock()
	c.quit <- true
}

func (c *Client) SetConnection(connection Connection) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connection == nil || connection.Priority() > c.connection.Priority() {
		c.connection = connection
	}
}

func (c *Client) ShouldReap() bool {
	if !c.isConnected() {
		return true
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().Sub(c.created) > time.Duration(1*time.Minute) {
		if c.lastSession != nil &&
			time.Now().Sub(c.lastSession.started) > time.Duration(2*time.Hour) {
			return true
		}
	}
	return false
}

func (c *Client) AddSubscriptions(patterns []string) {
	c.logger.Infof("SUBSCRIBE %s subscription: %v", c.clientId, patterns)
	defer c.mutex.RUnlock()
	c.mutex.RLock()
	c.subscriptions.AddMany(patterns)
}

func (c *Client) isConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connection != nil && c.connection.IsConnected()
}

func (c *Client) isSubscribed(patterns []string) bool {
	defer c.mutex.RUnlock()
	c.mutex.RLock()
	for _, p := range patterns {
		if c.subscriptions.Has(p) {
			return true
		}
	}
	return false
}

func (c *Client) Send(msg Message) bool {
	if c.isConnected() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		msgs := []Message{msg}
		if c.connection.IsSingleShot() {
			msgs = append(msgs, c.responseMsg)
		}
		c.logger.Debugf("Sending %d msgs to %s on %s", len(msgs), c.clientId, reflect.TypeOf(c.connection))

		err := c.connection.Send(msgs)

		if err != nil {
			c.logger.Debugf("Was unable to send to %s requeued %d messages", c.clientId, len(msgs))
			c.connection.Close()
			return false
		}

		return true
	}

	c.logger.Debugf("Not connected for %s", c.clientId)
	return false
}
