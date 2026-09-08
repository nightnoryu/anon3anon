package sqlite_test

import (
	"bytes"
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/domain"
	"anon3anon/pkg/infrastructure/storage/sqlite"
	"anon3anon/pkg/pseudonym"
)

const (
	testRateWindow = time.Hour
	testRateMax    = 3
)

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	return newStoreWithRate(t, testRateWindow, testRateMax)
}

func newTestKeyring(t *testing.T) *pseudonym.Keyring {
	t.Helper()
	keys, err := pseudonym.NewKeyring(bytes.Repeat([]byte("k"), pseudonym.MinKeyLen))
	require.NoError(t, err)
	return keys
}

func newStoreWithRate(t *testing.T, window time.Duration, maxRate int) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"), window, maxRate, newTestKeyring(t))
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

func TestClearSessionsForOwner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	other, err := store.UpsertUser(ctx, 20, 20)
	require.NoError(t, err)

	require.NoError(t, store.SetSession(ctx, 501, owner.TgUserID))
	require.NoError(t, store.SetSession(ctx, 502, owner.TgUserID))
	require.NoError(t, store.SetSession(ctx, 503, other.TgUserID))

	removed, err := store.ClearSessionsForOwner(ctx, owner.TgUserID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed)

	for _, sender := range []int64{501, 502} {
		_, ok, err2 := store.GetSession(ctx, sender)
		require.NoError(t, err2)
		assert.Falsef(t, ok, "sender %d must be cut off", sender)
	}

	// Other owners' sessions are untouched.
	got, ok, err := store.GetSession(ctx, 503)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, other.TgUserID, got)

	// Idempotent: no sessions left to clear is not an error.
	removed, err = store.ClearSessionsForOwner(ctx, owner.TgUserID)
	require.NoError(t, err)
	assert.Zero(t, removed)
}

func TestDeleteUserCascades(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	keep, err := store.UpsertUser(ctx, 99, 99)
	require.NoError(t, err)

	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID)) // sender -> owner
	require.NoError(t, store.SetSession(ctx, 10, keep.TgUserID))  // owner acting as a sender
	require.NoError(t, store.SetSession(ctx, 51, keep.TgUserID))  // unrelated
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 10, DestMsgID: 1, OriginChatID: 50, OwnerUserID: owner.TgUserID,
	}))
	require.NoError(t, store.Block(ctx, owner.TgUserID, 50))
	require.NoError(t, store.Block(ctx, keep.TgUserID, 10)) // someone blocked the owner-as-sender
	_, err = store.AllowMessage(ctx, 50, owner.TgUserID)
	require.NoError(t, err)

	deleted, err := store.DeleteUser(ctx, owner.TgUserID)
	require.NoError(t, err)
	assert.True(t, deleted)

	_, ok, err := store.UserByID(ctx, owner.TgUserID)
	require.NoError(t, err)
	assert.False(t, ok)

	for _, s := range []int64{50, 10} {
		_, sok, serr := store.GetSession(ctx, s)
		require.NoError(t, serr)
		assert.Falsef(t, sok, "session %d must be gone", s)
	}
	_, ok, err = store.LookupRelay(ctx, 10, 1)
	require.NoError(t, err)
	assert.False(t, ok, "owner's relay must be gone")

	blocked, err := store.IsBlocked(ctx, owner.TgUserID, 50)
	require.NoError(t, err)
	assert.False(t, blocked)
	blocked, err = store.IsBlocked(ctx, keep.TgUserID, 10)
	require.NoError(t, err)
	assert.False(t, blocked, "blocks against the deleted user's chat must be gone")

	// Unrelated rows survive.
	got, ok, err := store.GetSession(ctx, 51)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, keep.TgUserID, got)

	// Idempotent.
	deleted, err = store.DeleteUser(ctx, owner.TgUserID)
	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestDeleteUserIgnoresNonRecipients(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 10, DestMsgID: 1, OriginChatID: 50, OwnerUserID: owner.TgUserID,
	}))

	// Sender 50 has no user record: /delete from them must touch nothing.
	deleted, err := store.DeleteUser(ctx, 50)
	require.NoError(t, err)
	assert.False(t, deleted)

	got, ok, err := store.GetSession(ctx, 50)
	require.NoError(t, err)
	require.True(t, ok, "the sender's live session must survive their no-op /delete")
	assert.Equal(t, owner.TgUserID, got)

	_, ok, err = store.LookupRelay(ctx, 10, 1)
	require.NoError(t, err)
	assert.True(t, ok, "the recipient's relay row must survive")
}

func TestPurgeExpired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 501, owner.TgUserID))
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 7, DestMsgID: 42, OriginChatID: 9, OwnerUserID: owner.TgUserID,
	}))
	require.NoError(t, store.Block(ctx, owner.TgUserID, 501))
	ok, err := store.AllowMessage(ctx, 501, owner.TgUserID)
	require.NoError(t, err)
	require.True(t, ok)

	// Nothing is old enough yet.
	stats, err := store.PurgeExpired(ctx, time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, domain.PurgeStats{}, stats)

	// Everything now predates the cutoff. The cutoff is pushed a full rate
	// window ahead so the message_rates bucket index also falls before it.
	stats, err = store.PurgeExpired(ctx, time.Now().UTC().Add(2*testRateWindow))
	require.NoError(t, err)
	assert.Equal(t, domain.PurgeStats{Sessions: 1, Relays: 1, Blocks: 1, MessageRates: 1}, stats)

	_, ok, err = store.GetSession(ctx, 501)
	require.NoError(t, err)
	assert.False(t, ok, "expired session must be gone")

	_, ok, err = store.LookupRelay(ctx, 7, 42)
	require.NoError(t, err)
	assert.False(t, ok, "expired relay must be gone")

	blocked, err := store.IsBlocked(ctx, owner.TgUserID, 501)
	require.NoError(t, err)
	assert.False(t, blocked, "expired block must be gone")

	// Idempotent: nothing left to purge is not an error.
	stats, err = store.PurgeExpired(ctx, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, domain.PurgeStats{}, stats)
}

func TestPurgeExpiredKeepsCurrentMessageRateBucket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	ok, err := store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, ok)

	// A cutoff in the past must not touch the live bucket.
	stats, err := store.PurgeExpired(ctx, time.Now().UTC().Add(-2*testRateWindow))
	require.NoError(t, err)
	assert.Zero(t, stats.MessageRates)

	// The quota is still counted: max messages total, so the next is refused.
	for i := 1; i < testRateMax; i++ {
		ok, err = store.AllowMessage(ctx, 1, 2)
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err = store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestPurgeExpiredSkipsMessageRatesWhenRateLimitDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// A sub-second window counts as disabled; the sweep must not divide by zero.
	store := newStoreWithRate(t, 500*time.Millisecond, 3)

	stats, err := store.PurgeExpired(ctx, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	assert.Zero(t, stats.MessageRates)
}

func TestRefundMessageReturnsQuota(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t) // window 1h, max 3

	// Fill the quota exactly, then confirm the next message is refused.
	for i := 1; i <= testRateMax; i++ {
		ok, err := store.AllowMessage(ctx, 1, 2)
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err := store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	require.False(t, ok)

	// Two refunds undo the rejected attempt and one accepted one.
	require.NoError(t, store.RefundMessage(ctx, 1, 2))
	require.NoError(t, store.RefundMessage(ctx, 1, 2))

	ok, err = store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	assert.True(t, ok, "refunded quota is reusable")

	ok, err = store.AllowMessage(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, ok, "only the refunded slot came back")

	// Refund never drives the count below zero and does not error for a
	// sender->recipient pair that has no bucket at all.
	require.NoError(t, store.RefundMessage(ctx, 77, 88))
}

func TestRefundMessageDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStoreWithRate(t, testRateWindow, 0)

	require.NoError(t, store.RefundMessage(ctx, 1, 2))
}

func TestTouchSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	owner, err := store.UpsertUser(ctx, 10, 10)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 501, owner.TgUserID))

	// Touching an existing session keeps it pointed at the same owner and
	// protects it from a subsequent purge with a cutoff just before now.
	require.NoError(t, store.TouchSession(ctx, 501))

	stats, err := store.PurgeExpired(ctx, time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	assert.Zero(t, stats.Sessions)

	got, ok, err := store.GetSession(ctx, 501)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, owner.TgUserID, got)

	// No-op for a sender without a session.
	require.NoError(t, store.TouchSession(ctx, 999))
	_, ok, err = store.GetSession(ctx, 999)
	require.NoError(t, err)
	assert.False(t, ok)
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

func TestUpsertUserIsSafeUnderConcurrency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	// The bot dispatches every update in its own goroutine, so a user who taps
	// /start twice reaches UpsertUser concurrently.
	const callers = 8
	var (
		wg    sync.WaitGroup
		users = make([]domain.User, callers)
		errs  = make([]error, callers)
	)
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			users[i], errs[i] = store.UpsertUser(ctx, 300, 300)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "caller %d", i)
	}
	// One row, one token: nobody may be handed a second registration.
	for i, u := range users {
		assert.Equalf(t, users[0].LinkToken, u.LinkToken, "caller %d saw a different token", i)
	}
}

func TestClearRelaysForOwner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 1000, DestMsgID: 1, OriginChatID: 50, OwnerUserID: 10,
	}))
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 50, DestMsgID: 2, OriginChatID: 1000, OwnerUserID: 10,
	}))
	// A different owner's conversation must survive.
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 2000, DestMsgID: 3, OriginChatID: 50, OwnerUserID: 20,
	}))

	removed, err := store.ClearRelaysForOwner(ctx, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, removed)

	for _, c := range []struct {
		chatID int64
		msgID  int
	}{{1000, 1}, {50, 2}} {
		_, found, lookupErr := store.LookupRelay(ctx, c.chatID, c.msgID)
		require.NoError(t, lookupErr)
		assert.Falsef(t, found, "relay %d/%d must be gone", c.chatID, c.msgID)
	}

	_, ok, err := store.LookupRelay(ctx, 2000, 3)
	require.NoError(t, err)
	assert.True(t, ok, "another owner's relay must be untouched")

	// Idempotent.
	removed, err = store.ClearRelaysForOwner(ctx, 10)
	require.NoError(t, err)
	assert.Zero(t, removed)
}

func TestClearRelaysForSenderOnlyTouchesThatPair(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newStore(t)

	// Owner 10 talks to senders 50 and 51.
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 1000, DestMsgID: 1, OriginChatID: 50, OwnerUserID: 10,
	}))
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 50, DestMsgID: 2, OriginChatID: 1000, OwnerUserID: 10,
	}))
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 1000, DestMsgID: 3, OriginChatID: 51, OwnerUserID: 10,
	}))
	// Sender 50 also talks to owner 20.
	require.NoError(t, store.PutRelay(ctx, domain.Relay{
		DestChatID: 50, DestMsgID: 4, OriginChatID: 2000, OwnerUserID: 20,
	}))

	removed, err := store.ClearRelaysForSender(ctx, 50, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, removed, "both directions of the 50<->10 pair")

	for _, c := range []struct {
		chatID int64
		msgID  int
		want   bool
		what   string
	}{
		{1000, 1, false, "inbound leg of the cleared pair"},
		{50, 2, false, "outbound leg of the cleared pair"},
		{1000, 3, true, "the same owner's other sender"},
		{50, 4, true, "the same sender's other owner"},
	} {
		_, ok, err := store.LookupRelay(ctx, c.chatID, c.msgID)
		require.NoError(t, err)
		assert.Equalf(t, c.want, ok, "%s", c.what)
	}
}
