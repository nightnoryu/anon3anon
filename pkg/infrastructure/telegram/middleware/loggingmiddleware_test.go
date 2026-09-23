package middleware

import (
	"context"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/pseudonym"
)

func TestLoggingMiddlewareEmitsCompletionEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		message     *models.Message
		wantEvent   string
		wantCommand string
	}{
		{
			name:      "anonymous message",
			message:   testMessage("hello"),
			wantEvent: eventTypeAnonymousMessage,
		},
		{
			name: "reply",
			message: &models.Message{
				Chat:           models.Chat{ID: 42, Type: models.ChatTypePrivate},
				ID:             99,
				Text:           "reply",
				ReplyToMessage: &models.Message{ID: 98},
			},
			wantEvent: eventTypeReply,
		},
		{
			name:        "known command",
			message:     testCommand("/start token", 6),
			wantEvent:   eventTypeCommandCall,
			wantCommand: "start",
		},
		{
			name:        "command addressed to bot",
			message:     testCommand("/help@anon3anon_bot", 19),
			wantEvent:   eventTypeCommandCall,
			wantCommand: "help",
		},
		{
			name: "command takes precedence over reply",
			message: func() *models.Message {
				message := testCommand("/block", 6)
				message.ReplyToMessage = &models.Message{ID: 98}
				return message
			}(),
			wantEvent:   eventTypeCommandCall,
			wantCommand: "block",
		},
		{
			name:        "unknown command is bounded",
			message:     testCommand("/unregistered", 13),
			wantEvent:   eventTypeCommandCall,
			wantCommand: unknownCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := newCaptureLogger()
			completed := false
			wrapped := NewLoggingMiddleware(logger, testKeyring(t), []string{"start", "help", "block"})(
				func(context.Context, *bot.Bot, *models.Update) { completed = true },
			)

			wrapped(context.Background(), nil, &models.Update{ID: 7, Message: tt.message})

			require.True(t, completed)
			require.Len(t, logger.state.entries, 1)
			entry := logger.state.entries[0]
			assert.Equal(t, "telegram event processed", entry.message)
			assert.Equal(t, tt.wantEvent, entry.fields[eventTypeField])
			if tt.wantCommand == "" {
				assert.NotContains(t, entry.fields, commandField)
			} else {
				assert.Equal(t, tt.wantCommand, entry.fields[commandField])
			}
			assert.IsType(t, float64(0), entry.fields[durationMSField])
			durationMS, ok := entry.fields[durationMSField].(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, durationMS, float64(0))
			assert.NotContains(t, entry.fields, chatIDField)
			assert.NotContains(t, entry.fields, userIDField)
			assert.NotContains(t, entry.fields, usernameField)
			assert.NotContains(t, entry.fields, textField)
		})
	}
}

func TestLoggingMiddlewareSkipsUpdatesWithoutMessages(t *testing.T) {
	t.Parallel()

	logger := newCaptureLogger()
	handled := false
	wrapped := NewLoggingMiddleware(logger, testKeyring(t), nil)(
		func(context.Context, *bot.Bot, *models.Update) { handled = true },
	)

	wrapped(context.Background(), nil, &models.Update{ID: 7})

	assert.False(t, handled)
	assert.Empty(t, logger.state.entries)
}

func testMessage(text string) *models.Message {
	return &models.Message{
		Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate},
		ID:   99,
		Text: text,
	}
}

func testCommand(text string, length int) *models.Message {
	message := testMessage(text)
	message.Entities = []models.MessageEntity{{
		Type:   models.MessageEntityTypeBotCommand,
		Offset: 0,
		Length: length,
	}}
	return message
}

func testKeyring(t *testing.T) *pseudonym.Keyring {
	t.Helper()

	keys, err := pseudonym.NewKeyring(make([]byte, pseudonym.MinKeyLen))
	require.NoError(t, err)
	return keys
}

type capturedEntry struct {
	fields  log.Fields
	message string
}

type captureLogger struct {
	state  *captureLogState
	fields log.Fields
}

type captureLogState struct {
	entries []capturedEntry
}

func newCaptureLogger() *captureLogger {
	return &captureLogger{state: &captureLogState{}}
}

func (l *captureLogger) WithFields(fields log.Fields) log.Logger {
	merged := make(log.Fields, len(l.fields)+len(fields))
	for key, value := range l.fields {
		merged[key] = value
	}
	for key, value := range fields {
		merged[key] = value
	}
	return &captureLogger{state: l.state, fields: merged}
}

func (l *captureLogger) Debug(...any) {}

func (l *captureLogger) Info(args ...any) {
	l.state.entries = append(l.state.entries, capturedEntry{fields: l.fields, message: stringify(args)})
}

func (l *captureLogger) Warn(...any) {}

func (l *captureLogger) Error(error, ...any) {}

func stringify(args []any) string {
	if len(args) == 1 {
		if value, ok := args[0].(string); ok {
			return value
		}
	}
	return ""
}
