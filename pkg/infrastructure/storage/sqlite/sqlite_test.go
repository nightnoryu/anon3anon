package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/domain"
	"anon3anon/pkg/infrastructure/storage/sqlite"
)

const (
	testRateWindow = time.Hour
	testRateMax    = 3
)

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	return newStoreWithRate(t, testRateWindow, testRateMax)
}

func newStoreWithRate(t *testing.T, window time.Duration, maxRate int) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"), window, maxRate)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close()) })
	return store
}

func TestUpsertUserIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	first, err := store.UpsertUser(ctx, 100, 100)
	require.NoError(t, err)
	assert.NotEmpty(t, first.LinkToken)

	// Same user, new chat id: keeps the token, refreshes the chat id.
	second, err := store.UpsertUser(ctx, 100, 200)
	require.NoError(t, err)
	assert.Equal(t, first.LinkToken, second.LinkToken)
	assert.Equal(t, int64(200), second.ChatID)

	byID, ok, err := store.UserByID(ctx, 100)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(200), byID.ChatID)
}

func TestUserByTokenAndRotate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	user, err := store.UpsertUser(ctx, 1, 1)
	require.NoError(t, err)

	found, ok, err := store.UserByToken(ctx, user.LinkToken)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(1), found.TgUserID)

	newToken, err := store.RotateToken(ctx, 1)
	require.NoError(t, err)
	assert.NotEqual(t, user.LinkToken, newToken)

	_, ok, err = store.UserByToken(ctx, user.LinkToken)
	require.NoError(t, err)
	assert.False(t, ok, "old token must stop resolving")

	_, ok, err = store.UserByToken(ctx, newToken)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUnknownLookups(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	_, ok, err := store.UserByToken(ctx, "nope")
	require.NoError(t, err)
	assert.False(t, ok)

	_, ok, err = store.UserByID(ctx, 999)
	require.NoError(t, err)
	assert.False(t, ok)

	_, ok, err = store.GetSession(ctx, 999)
	require.NoError(t, err)
	assert.False(t, ok)

	_, ok, err = store.LookupRelay(ctx, 1, 1)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSessionSetGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	other, err := store.UpsertUser(ctx, 20, 20)
	require.NoError(t, err)

	require.NoError(t, store.SetSession(ctx, 555, owner.TgUserID))
	got, ok, err := store.GetSession(ctx, 555)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, owner.TgUserID, got)

	// Last link opened wins.
	require.NoError(t, store.SetSession(ctx, 555, other.TgUserID))
	got, ok, err = store.GetSession(ctx, 555)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, other.TgUserID, got)
}

func TestClearSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 555, owner.TgUserID))

	require.NoError(t, store.ClearSession(ctx, 555))
	_, ok, err := store.GetSession(ctx, 555)
	require.NoError(t, err)
	assert.False(t, ok)

	// Idempotent: clearing a missing session is not an error.
	require.NoError(t, store.ClearSession(ctx, 555))
}

func TestRelayPutLookupAndUpsert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	r := domain.Relay{DestChatID: 7, DestMsgID: 42, OriginChatID: 9, OwnerUserID: 3}
	require.NoError(t, store.PutRelay(ctx, r))

	got, ok, err := store.LookupRelay(ctx, 7, 42)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(9), got.OriginChatID)
	assert.Equal(t, int64(3), got.OwnerUserID)

	// Re-delivering the same (chat, msg) overwrites cleanly.
	require.NoError(t, store.PutRelay(ctx, domain.Relay{DestChatID: 7, DestMsgID: 42, OriginChatID: 11, OwnerUserID: 3}))
	got, ok, err = store.LookupRelay(ctx, 7, 42)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(11), got.OriginChatID)
}

func TestAllowMessageQuotaPerSenderRecipientBucket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t) // window 1h, max 3

	for i := 1; i <= testRateMax; i++ {
		ok, err := store.AllowMessage(ctx, 1, 2)
		require.NoError(t, err)
		assert.True(t, ok, "message %d is within quota", i)
	}

	ok, err := store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, ok, "message over quota is rejected")

	// A different recipient has an independent budget.
	ok, err = store.AllowMessage(ctx, 1, 99)
	require.NoError(t, err)
	assert.True(t, ok)

	// A different sender has an independent budget.
	ok, err = store.AllowMessage(ctx, 42, 2)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestAllowMessageDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStoreWithRate(t, testRateWindow, 0)

	for range 100 {
		ok, err := store.AllowMessage(ctx, 1, 2)
		require.NoError(t, err)
		require.True(t, ok)
	}
}

func TestBlockAndIsBlocked(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	blocked, err := store.IsBlocked(ctx, 10, 555)
	require.NoError(t, err)
	assert.False(t, blocked)

	require.NoError(t, store.Block(ctx, 10, 555))

	blocked, err = store.IsBlocked(ctx, 10, 555)
	require.NoError(t, err)
	assert.True(t, blocked)

	// Block is idempotent.
	require.NoError(t, store.Block(ctx, 10, 555))
	blocked, err = store.IsBlocked(ctx, 10, 555)
	require.NoError(t, err)
	assert.True(t, blocked)

	// Scoped to the (owner, sender) pair.
	blocked, err = store.IsBlocked(ctx, 10, 777)
	require.NoError(t, err)
	assert.False(t, blocked)

	blocked, err = store.IsBlocked(ctx, 20, 555)
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestRotateTokenUnknownUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	_, err := store.RotateToken(ctx, 12345)
	assert.Error(t, err)
}
