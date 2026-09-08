package handler

import (
	"context"
	"testing"

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
