package protocol

import (
	"strings"
)

type MetaChannel interface{}

const (
	MetaPrefix             string = "/meta/"
	MetaService                   = "/service"
	MetaHandshakeChannel          = "handshake"
	MetaSubscribeChannel          = "subscribe"
	MetaUnsubscribeChannel        = "unsubscribe"
	MetaConnectChannel            = "connect"
	MetaDisconnectChannel         = "disconnect"
	MetaUnknownChannel            = "unknown"
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
		case MetaUnsubscribeChannel:
			return MetaUnsubscribeChannel
		case MetaDisconnectChannel:
			return MetaDisconnectChannel
		case MetaHandshakeChannel:
			return MetaHandshakeChannel
		default:
			return MetaUnknownChannel
		}
	}
}

func (c Channel) Expand() []string {
	segments := strings.Split(c.name, "/")
	numSegments := len(segments)
	patterns := make([]string, 0, numSegments+2)
	patterns = append(patterns, "/**")
	for i := 1; i < numSegments-1; i++ {
		patterns = append(patterns, strings.Join(segments[:i+1], "/")+"/**")
	}
	patterns = append(patterns, strings.Join(segments[:numSegments-1], "/")+"/*")
	patterns = append(patterns, c.name)
	return patterns
}
