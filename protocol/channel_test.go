package protocol

import (
	"reflect"
	"testing"
)

func TestChannelIsMeta(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		expected bool
	}{
		{"meta handshake", "/meta/handshake", true},
		{"meta connect", "/meta/connect", true},
		{"regular channel", "/foo/bar", false},
		{"empty", "", false},
		{"service", "/service/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChannel(tt.channel)
			if got := c.IsMeta(); got != tt.expected {
				t.Errorf("Channel(%q).IsMeta() = %v, want %v", tt.channel, got, tt.expected)
			}
		})
	}
}

func TestChannelIsService(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		expected bool
	}{
		{"service channel", "/service/test", true},
		{"meta channel", "/meta/connect", false},
		{"regular channel", "/foo/bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChannel(tt.channel)
			if got := c.IsService(); got != tt.expected {
				t.Errorf("Channel(%q).IsService() = %v, want %v", tt.channel, got, tt.expected)
			}
		})
	}
}

func TestChannelMetaType(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		expected MetaChannel
	}{
		{"handshake", "/meta/handshake", MetaHandshakeChannel},
		{"connect", "/meta/connect", MetaConnectChannel},
		{"disconnect", "/meta/disconnect", MetaDisconnectChannel},
		{"subscribe", "/meta/subscribe", MetaSubscribeChannel},
		{"unsubscribe", "/meta/unsubscribe", MetaUnsubscribeChannel},
		{"unknown meta", "/meta/unknown", MetaUnknownChannel},
		{"not meta", "/foo/bar", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChannel(tt.channel)
			if got := c.MetaType(); got != tt.expected {
				t.Errorf("Channel(%q).MetaType() = %v, want %v", tt.channel, got, tt.expected)
			}
		})
	}
}

func TestChannelExpand(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		expected []string
	}{
		{
			name:     "simple two segment",
			channel:  "/foo/bar",
			expected: []string{"/**", "/foo/**", "/foo/*", "/foo/bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChannel(tt.channel)
			got := c.Expand()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Channel(%q).Expand() = %v, want %v", tt.channel, got, tt.expected)
			}
		})
	}
}
