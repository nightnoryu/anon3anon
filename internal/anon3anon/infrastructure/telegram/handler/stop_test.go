package handler

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopSessionClearsActiveSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.stopSession(ctx, c, testMsg(5, 50, "/stop"))

	assert.Equal(t, stoppedMessage, c.lastSend())
	_, ok, err := store.GetSession(ctx, 50)
	require.NoError(t, err)
	assert.False(t, ok, "session must be cleared")
}

func TestStopSessionWithoutSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)

	c := &fakeClient{}
	d.stopSession(ctx, c, testMsg(5, 50, "/stop"))

	assert.Equal(t, noSessionToStopMessage, c.lastSend())
}

func TestStopSessionThenMessageHasNoTarget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.stopSession(ctx, c, testMsg(5, 50, "/stop"))
	d.routeToOwner(ctx, c, testMsg(5, 50, "still there?"))

	assert.Equal(t, noSessionMessage, c.lastSend())
	assert.Empty(t, c.copies, "message must not be delivered after /stop")
}

func TestStopSessionCutsOffRepliesToDeliveredMessages(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello"))
	deliveredID := c.nextMsgID

	ownerReply := testMsg(owner.TgUserID, owner.ChatID, "hi")
	ownerReply.ReplyToMessage = &models.Message{ID: deliveredID}
	require.True(t, d.tryRouteReply(ctx, c, ownerReply))
	relayedToSenderID := c.nextMsgID
	require.Len(t, c.copies, 2)

	d.stopSession(ctx, c, testMsg(5, 50, "/stop"))
	require.Equal(t, stoppedMessage, c.lastSend())

	// Leaving the conversation must also close the reply path, which is routed
	// from the relay map rather than from the session that was just cleared.
	senderReply := testMsg(5, 50, "still here?")
	senderReply.ReplyToMessage = &models.Message{ID: relayedToSenderID}
	assert.False(t, d.tryRouteReply(ctx, c, senderReply), "the relay must no longer resolve")

	d.routeToOwner(ctx, c, senderReply)

	assert.Len(t, c.copies, 2, "nothing may reach the owner after /stop")
	assert.Equal(t, noSessionMessage, c.lastSend())
}

func TestStopSessionLeavesOtherOwnersThreadsAlone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	first, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	second, err := store.UpsertUser(ctx, 20, 2000)
	require.NoError(t, err)

	c := &fakeClient{}

	// Sender 50 opens the first owner's link and gets an answer.
	require.NoError(t, store.SetSession(ctx, 50, first.TgUserID))
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello first"))
	firstReply := testMsg(first.TgUserID, first.ChatID, "hi")
	firstReply.ReplyToMessage = &models.Message{ID: c.nextMsgID}
	require.True(t, d.tryRouteReply(ctx, c, firstReply))
	firstThreadID := c.nextMsgID

	// Then opens the second owner's link - sessions are last-link-wins.
	require.NoError(t, store.SetSession(ctx, 50, second.TgUserID))
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello second"))
	secondReply := testMsg(second.TgUserID, second.ChatID, "hi")
	secondReply.ReplyToMessage = &models.Message{ID: c.nextMsgID}
	require.True(t, d.tryRouteReply(ctx, c, secondReply))
	secondThreadID := c.nextMsgID
	require.Len(t, c.copies, 4)

	d.stopSession(ctx, c, testMsg(5, 50, "/stop"))

	// Only the conversation that was left is closed.
	toSecond := testMsg(5, 50, "still there?")
	toSecond.ReplyToMessage = &models.Message{ID: secondThreadID}
	assert.False(t, d.tryRouteReply(ctx, c, toSecond), "the stopped pair must be cut")

	toFirst := testMsg(5, 50, "still talking")
	toFirst.ReplyToMessage = &models.Message{ID: firstThreadID}
	assert.True(t, d.tryRouteReply(ctx, c, toFirst), "the other owner's thread must survive")

	require.Len(t, c.copies, 5)
	assert.EqualValues(t, first.ChatID, c.copies[4].ChatID)
}
