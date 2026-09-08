package handler

import (
	"context"
	"testing"

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
