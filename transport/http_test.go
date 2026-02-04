package transport

import (
	"strings"
	"testing"
)

func TestIsValidJSONPCallback(t *testing.T) {
	tests := []struct {
		name     string
		callback string
		expected bool
	}{
		{"simple callback", "callback", true},
		{"with underscore", "my_callback", true},
		{"with numbers", "callback123", true},
		{"namespaced", "jQuery.callback", true},
		{"deep namespaced", "my.app.callback", true},
		{"starts with underscore", "_private", true},
		{"empty", "", false},
		{"starts with number", "123callback", false},
		{"contains special chars", "callback<script>", false},
		{"contains parentheses", "callback()", false},
		{"contains semicolon", "callback;alert", false},
		{"contains space", "call back", false},
		{"too long", strings.Repeat("a", 129), false},
		{"max length", strings.Repeat("a", 128), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidJSONPCallback(tt.callback); got != tt.expected {
				t.Errorf("isValidJSONPCallback(%q) = %v, want %v", tt.callback, got, tt.expected)
			}
		})
	}
}
