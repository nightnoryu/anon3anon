package handler

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevokeLinkRotatesTokenAndCutsOffSenders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))
	require.NoError(t, store.SetSession(ctx, 51, owner.TgUserID))

	c := &fakeClient{}
	d.revokeLink(ctx, c, testMsg(10, 1000, "/revoke"))

	// A fresh token was issued.
	updated, ok, err := store.UserByID(ctx, 10)
	require.NoError(t, err)
	require.True(t, ok)
	assert.NotEqual(t, owner.LinkToken, updated.LinkToken)
	assert.Contains(t, c.lastSend(), updated.LinkToken)

	// Senders who already opened the old link are cut off.
	for _, sender := range []int64{50, 51} {
		_, ok, err := store.GetSession(ctx, sender)
		require.NoError(t, err)
		assert.Falsef(t, ok, "sender %d must lose their session", sender)
	}
}

func TestRevokeLinkRegistersUnknownUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	c := &fakeClient{}
	d.revokeLink(ctx, c, testMsg(77, 77, "/revoke"))

	// An unregistered user gets registered instead.
	_, ok, err := store.UserByID(ctx, 77)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, c.lastSend(), "https://t.me/testbot?start=")
}

func TestRevokeLinkCutsOffRepliesToDeliveredMessages(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello"))
	deliveredID := c.nextMsgID

	// Owner answers, so the sender's chat holds a relay of its own.
	ownerReply := testMsg(owner.TgUserID, owner.ChatID, "hi")
	ownerReply.ReplyToMessage = &models.Message{ID: deliveredID}
	require.True(t, d.tryRouteReply(ctx, c, ownerReply))
	relayedToSenderID := c.nextMsgID
	require.Len(t, c.copies, 2)

	d.revokeLink(ctx, c, testMsg(10, 1000, "/revoke"))

	// Dropping the session alone would not cut this sender off: answering an
	// already delivered message is routed from the relay map, which no session
	// is consulted for.
	senderReply := testMsg(5, 50, "still here?")
	senderReply.ReplyToMessage = &models.Message{ID: relayedToSenderID}
	assert.False(t, d.tryRouteReply(ctx, c, senderReply), "the relay must no longer resolve")

	// What the router does next once the reply path declines the message.
	d.routeToOwner(ctx, c, senderReply)

	assert.Len(t, c.copies, 2, "nothing may reach the owner after /revoke")
	assert.Equal(t, noSessionMessage, c.lastSend())
}
