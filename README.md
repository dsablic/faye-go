# Faye + Go

Major websocket-oriented rewrite of [roncohen/faye-go](https://github.com/roncohen/faye-go). Still experimental.

## Usage

```go

package main

import (
	"net/http"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/dsablic/faye-go"
	"github.com/dsablic/faye-go/adapters"
	"github.com/dsablic/faye-go/protocol"
)

type extLogger struct {
	*log.Logger
}

func (l extLogger) Panicf(msg string, v ...interface{}) {
	l.Errorf(msg, v)
	panic(msg)
}

type callbacks struct{}

func (c callbacks) SubscribeValid(m *protocol.Message) bool {
	return true
}

func (c callbacks) PublishValid(m *protocol.Message) bool {
	return true
}

func main() {
	duration := time.Duration(10) * time.Second
	ctx := log.WithFields(log.Fields{})
	logger := extLogger{ctx.Logger}
	log.SetHandler(text.New(os.Stdout))
	statistics := make(chan faye.Counters)
	go func() {
    interval := uint(duration.Seconds())
		for c := range statistics {
			logger.Infof("Clients = %v, Publish = %v/s, Send = %v/s, Fail = %v/s",
				c.Clients, c.Published/interval, c.Sent/interval, c.Failed/interval)
		}
	}()
	engine := faye.NewEngine(logger, duration, statistics)
	server := faye.NewServer(logger, engine, callbacks{})
	http.Handle("/bayeux", adapters.FayeHandler(server))
	if err := http.ListenAndServe(":8000", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

```
