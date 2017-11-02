package protocol

import (
	"strings"
)

type MetaChannel interface{}

const (
	MetaPrefix            string = "/meta/"
	MetaService                  = "/service"
	MetaHandshakeChannel         = "handshake"
	MetaSubscribeChannel         = "subscribe"
	MetaConnectChannel           = "connect"
	MetaDisconnectChannel        = "disconnect"
	MetaUnknownChannel           = "unknown"
)

func NewChannel(name string) Channel {
	return Channel{name}
}

type Channel struct {
	name string
}

type Subscription Channel

func (c Channel) Name() string {
	return c.name
}

func (c Channel) IsMeta() bool {
	return strings.HasPrefix(c.name, MetaPrefix)
}

func (c Channel) IsService() bool {
	return strings.HasPrefix(c.name, MetaService)
}

func (c Channel) MetaType() MetaChannel {
	if !c.IsMeta() {
		return nil
	} else {
		switch c.name[len(MetaPrefix):] {
		case MetaConnectChannel:
			return MetaConnectChannel
		case MetaSubscribeChannel:
			return MetaSubscribeChannel
		case MetaDisconnectChannel:
			return MetaDisconnectChannel
		case MetaHandshakeChannel:
			return MetaHandshakeChannel
		default:
			return MetaUnknownChannel
		}
	}
}

// Returns all the channels patterns that could match this channel
/*
For:
/foo/bar
We should return these:
/**
/foo/**
/foo/*
/foo/bar
*/
func (c Channel) Expand() []string {
	segments := strings.Split(c.name, "/")
	num_segments := len(segments)
	patterns := make([]string, num_segments+1)
	patterns[0] = "/**"
	for i := 1; i < len(segments); i = i + 2 {
		patterns[i] = strings.Join(segments[:i+1], "/") + "/**"
	}
	patterns[len(patterns)-2] = strings.Join(segments[:num_segments-1], "/") + "/*"
	patterns[len(patterns)-1] = c.name
	return patterns
}
