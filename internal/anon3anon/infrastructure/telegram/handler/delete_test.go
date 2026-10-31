package handler

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAccountErasesUserAndRelatedRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))
	require.NoError(t, store.Block(ctx, owner.TgUserID, 60))

	c := &fakeClient{}
	d.deleteAccount(ctx, c, testMsg(10, 10, "/delete"))
	assert.Equal(t, accountDeletedMessage, c.lastSend())

	_, ok, err := store.UserByID(ctx, 10)
	require.NoError(t, err)
	assert.False(t, ok, "user record must be gone")

	_, ok, err = store.GetSession(ctx, 50)
	require.NoError(t, err)
	assert.False(t, ok, "sessions pointed at the owner must be gone")

	blocked, err := store.IsBlocked(ctx, 10, 60)
	require.NoError(t, err)
	assert.False(t, blocked, "owner's blocks must be gone")
}

func TestDeleteAccountWithoutRegistration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)

	c := &fakeClient{}
	d.deleteAccount(ctx, c, testMsg(77, 77, "/delete"))

	assert.Equal(t, noAccountToDeleteMessage, c.lastSend())
}

func TestDeleteAccountClearsSenderSideSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	// User 20 is a registered recipient who is also a sender in owner 10's link.
	_, err = store.UpsertUser(ctx, 20, 20)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 20, owner.TgUserID))

	c := &fakeClient{}
	d.deleteAccount(ctx, c, testMsg(20, 20, "/delete"))
	assert.Equal(t, accountDeletedMessage, c.lastSend())

	_, ok, err := store.GetSession(ctx, 20)
	require.NoError(t, err)
	assert.False(t, ok, "the deleted user's own outgoing session must be gone")
}

func TestRelayRejectsContactAndLocation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	for name, mut := range map[string]func(*models.Message){
		"contact":  func(m *models.Message) { m.Contact = &models.Contact{PhoneNumber: "+123", FirstName: "A"} },
		"location": func(m *models.Message) { m.Location = &models.Location{Latitude: 1, Longitude: 2} },
		"venue":    func(m *models.Message) { m.Venue = &models.Venue{Title: "X"} },
		"story":    func(m *models.Message) { m.Story = &models.Story{ID: 7} },
		"users_shared": func(m *models.Message) {
			m.UsersShared = &models.UsersShared{RequestID: 1}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := &fakeClient{}
			msg := testMsg(5, 50, "")
			mut(msg)
			d.routeToOwner(ctx, c, msg)

			assert.Equal(t, unsupportedContentMessage, c.lastSend())
			assert.Empty(t, c.copies, "unsupported content must not be relayed")
		})
	}
}

func TestRelayRejectsContactWithoutBurningQuota(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t) // rate max 3

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	for range 5 {
		msg := testMsg(5, 50, "")
		msg.Location = &models.Location{Latitude: 1, Longitude: 2}
		d.routeToOwner(ctx, c, msg)
	}
	assert.Equal(t, unsupportedContentMessage, c.lastSend())

	// Quota is untouched: three plain messages still go through.
	for range 3 {
		d.routeToOwner(ctx, c, testMsg(5, 50, "hi"))
	}
	assert.Equal(t, messageSentMessage, c.lastSend())
	assert.Len(t, c.copies, 3)
}
