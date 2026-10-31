package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessagesForLanguage(t *testing.T) {
	t.Parallel()

	messages, err := MessagesForLanguage(LanguageEnglish)
	require.NoError(t, err)
	assert.Equal(t, "Message sent!", messages.messageSent)

	_, err = MessagesForLanguage("de")
	assert.Error(t, err)
}

func TestRegistrationNotAllowedOmitsOwnerContactWhenUnset(t *testing.T) {
	t.Parallel()

	messages, err := MessagesForLanguage(LanguageEnglish)
	require.NoError(t, err)
	assert.Equal(t, "Registration of new recipients is restricted.", messages.registrationNotAllowed(""))
}

func TestRegisterRecipientUsesEnglishMessages(t *testing.T) {
	t.Parallel()

	d, _ := newTestDeps(t)
	messages, err := MessagesForLanguage(LanguageEnglish)
	require.NoError(t, err)
	d.Messages = messages
	c := &fakeClient{}

	d.registerRecipient(context.Background(), c, testMsg(100, 100, "/start"))

	assert.Contains(t, c.lastSend(), "Your personal link:")
}
