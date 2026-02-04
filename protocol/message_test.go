package protocol

import (
	"testing"
)

func TestMessageChannel(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name:     "has channel",
			message:  Message{"channel": "/foo/bar"},
			expected: "/foo/bar",
		},
		{
			name:     "no channel",
			message:  Message{},
			expected: "",
		},
		{
			name:     "wrong type",
			message:  Message{"channel": 123},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.Channel().Name(); got != tt.expected {
				t.Errorf("Message.Channel().Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMessageClientId(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected uint32
	}{
		{
			name:     "valid client id",
			message:  Message{"clientId": "client-123"},
			expected: 123,
		},
		{
			name:     "no client id",
			message:  Message{},
			expected: 0,
		},
		{
			name:     "invalid format",
			message:  Message{"clientId": "invalid"},
			expected: 0,
		},
		{
			name:     "wrong type",
			message:  Message{"clientId": 123},
			expected: 0,
		},
		{
			name:     "large client id within int32",
			message:  Message{"clientId": "client-2147483647"},
			expected: 2147483647,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.ClientId(); got != tt.expected {
				t.Errorf("Message.ClientId() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestMessageJsonp(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name:     "has jsonp",
			message:  Message{"jsonp": "callback"},
			expected: "callback",
		},
		{
			name:     "no jsonp",
			message:  Message{},
			expected: "",
		},
		{
			name:     "wrong type",
			message:  Message{"jsonp": 123},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.Jsonp(); got != tt.expected {
				t.Errorf("Message.Jsonp() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMessageSetClientId(t *testing.T) {
	m := Message{}
	m.SetClientId(42)

	if got, ok := m["clientId"].(string); !ok || got != "client-42" {
		t.Errorf("SetClientId(42) = %v, want \"client-42\"", m["clientId"])
	}
}

func TestMessageUpdate(t *testing.T) {
	m := Message{"a": 1, "b": 2}
	m.Update(Message{"b": 3, "c": 4})

	if m["a"] != 1 {
		t.Errorf("m[\"a\"] = %v, want 1", m["a"])
	}
	if m["b"] != 3 {
		t.Errorf("m[\"b\"] = %v, want 3", m["b"])
	}
	if m["c"] != 4 {
		t.Errorf("m[\"c\"] = %v, want 4", m["c"])
	}
}
