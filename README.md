# faye-go

A Bayeux protocol server implementation in Go, optimized for WebSocket connections.

## Installation

```bash
go get github.com/dsablic/faye-go
```

## Usage

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dsablic/faye-go"
	"github.com/dsablic/faye-go/adapters"
	"github.com/dsablic/faye-go/protocol"
)

type logger struct{}

func (l logger) Debugf(msg string, v ...interface{}) { log.Printf("[DEBUG] "+msg, v...) }
func (l logger) Infof(msg string, v ...interface{})  { log.Printf("[INFO] "+msg, v...) }
func (l logger) Warnf(msg string, v ...interface{})  { log.Printf("[WARN] "+msg, v...) }
func (l logger) Errorf(msg string, v ...interface{}) { log.Printf("[ERROR] "+msg, v...) }
func (l logger) Panicf(msg string, v ...interface{}) { log.Panicf(msg, v...) }

type validator struct{}

func (v validator) SubscribeValid(m *protocol.Message) bool { return true }
func (v validator) PublishValid(m *protocol.Message) bool   { return true }

func main() {
	l := logger{}
	statistics := make(chan faye.Counters)

	go func() {
		for c := range statistics {
			log.Printf("Clients=%d Published=%d Sent=%d Failed=%d",
				c.Clients, c.Published, c.Sent, c.Failed)
		}
	}()

	engine := faye.NewEngine(l, 10*time.Second, statistics)
	server := faye.NewServer(l, engine, validator{})

	http.Handle("/bayeux", adapters.FayeHandler(server))
	log.Fatal(http.ListenAndServe(":8000", nil))
}
```

## WebSocket CORS

To allow cross-origin WebSocket connections, use `FayeHandlerWithCheckOrigin`:

```go
checkOrigin := func(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Allow specific origins
	return origin == "https://example.com"
}

http.Handle("/bayeux", adapters.FayeHandlerWithCheckOrigin(server, checkOrigin))
```

## Interfaces

### Logger

```go
type Logger interface {
	Debugf(msg string, v ...interface{})
	Infof(msg string, v ...interface{})
	Warnf(msg string, v ...interface{})
	Errorf(msg string, v ...interface{})
	Panicf(msg string, v ...interface{})
}
```

### Validator

```go
type Validator interface {
	SubscribeValid(*protocol.Message) bool
	PublishValid(*protocol.Message) bool
}
```

## Testing

```bash
go test ./...
```

## License

MIT
