package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMyLink(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"https://t.me/anon3anon_bot?start=abc123",
		buildMyLink("anon3anon_bot", "abc123"),
	)
}

func TestCommandPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{"no payload", "/start", ""},
		{"empty string", "", ""},
		{"plain payload", "/start token123", "token123"},
		{"command@bot form", "/start@anon3anon_bot token123", "token123"},
		{"trailing junk ignored", "/start token123 extra words", "token123"},
		{"surrounding whitespace", "  /start   token123  ", "token123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, commandPayload(tt.text))
		})
	}
}
