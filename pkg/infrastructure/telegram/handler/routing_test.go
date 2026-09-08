package handler

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/infrastructure/storage/sqlite"
)

type noopLogger struct{}

func (noopLogger) WithFields(log.Fields) log.Logger { return noopLogger{} }
func (noopLogger) Debug(...any)                     {}
func (noopLogger) Info(...any)                      {}
func (noopLogger) Error(error, ...any)              {}

type fakeClient struct {
	copies    []bot.CopyMessageParams
	sends     []bot.SendMessageParams
	copyErr   error
	nextMsgID int
}

func (f *fakeClient) CopyMessage(_ context.Context, p *bot.CopyMessageParams) (*models.MessageID, error) {
	if f.copyErr != nil {
		return nil, f.copyErr
	}
	f.copies = append(f.copies, *p)
	f.nextMsgID++
	return &models.MessageID{ID: f.nextMsgID}, nil
}

func (f *fakeClient) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	f.sends = append(f.sends, *p)
	return &models.Message{ID: len(f.sends)}, nil
}

func (f *fakeClient) lastSend() string {
	if len(f.sends) == 0 {
		return ""
	}
	return f.sends[len(f.sends)-1].Text
}

func newTestDeps(t *testing.T, allowed ...int64) (DependencyContainer, *sqlite.Store) {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"), time.Hour, 3)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close()) })

	return DependencyContainer{
		Store:        store,
		Logger:       noopLogger{},
		BotUsername:  "testbot",
		AllowedUsers: NewAllowList(allowed),
	}, store
}

func testMsg(fromID, chatID int64, text string) *models.Message {
	return &models.Message{
		ID:   1,
		From: &models.User{ID: fromID},
		Chat: models.Chat{ID: chatID},
		Text: text,
	}
}

func TestRegisterRecipientRepliesWithPersonalLink(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)
	c := &fakeClient{}

	d.registerRecipient(ctx, c, testMsg(100, 100, "/start"))

	require.Len(t, c.sends, 1)
	user, ok, err := store.UserByID(ctx, 100)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Contains(t, c.sends[0].Text, "https://t.me/testbot?start="+user.LinkToken)
}

func TestRegisterRecipientRejectedWhenNotOnAllowList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t, 999) // only user 999 may register
	c := &fakeClient{}

	d.registerRecipient(ctx, c, testMsg(100, 100, "/start"))

	assert.Equal(t, notAllowedMessage, c.lastSend())
	_, ok, err := store.UserByID(ctx, 100)
	require.NoError(t, err)
	assert.False(t, ok, "rejected user must not be persisted")
}

func TestJoinByToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 1, 1)
	require.NoError(t, err)

	t.Run("valid token opens a session", func(t *testing.T) {
		c := &fakeClient{}
		d.joinByToken(ctx, c, testMsg(2, 2, "/start "+owner.LinkToken), owner.LinkToken)

		assert.Equal(t, joinedMessage, c.lastSend())
		got, ok, err := store.GetSession(ctx, 2)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, owner.TgUserID, got)
	})

	t.Run("unknown token is rejected", func(t *testing.T) {
		c := &fakeClient{}
		d.joinByToken(ctx, c, testMsg(3, 3, "/start nope"), "nope")
		assert.Equal(t, invalidLinkMessage, c.lastSend())
	})

	t.Run("opening your own link is rejected", func(t *testing.T) {
		c := &fakeClient{}
		d.joinByToken(ctx, c, testMsg(1, 1, "/start "+owner.LinkToken), owner.LinkToken)
		assert.Equal(t, ownLinkMessage, c.lastSend())
	})
}

func TestRouteToOwnerWithoutSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)
	c := &fakeClient{}

	d.routeToOwner(ctx, c, testMsg(5, 5, "hi"))

	assert.Equal(t, noSessionMessage, c.lastSend())
	assert.Empty(t, c.copies)
}

func TestRouteToOwnerDeliversMessageAndRecordsRelay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "anonymous hello"))

	require.Len(t, c.copies, 1)
	assert.Equal(t, owner.ChatID, c.copies[0].ChatID)
	assert.Equal(t, int64(50), c.copies[0].FromChatID)
	assert.Equal(t, messageSentMessage, c.lastSend())

	relay, ok, err := store.LookupRelay(ctx, owner.ChatID, c.nextMsgID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(50), relay.OriginChatID)
	assert.Equal(t, owner.TgUserID, relay.OwnerUserID)
}

func TestRouteToOwnerBlockedSender(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))
	require.NoError(t, store.Block(ctx, owner.TgUserID, 50))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "let me in"))

	assert.Equal(t, blockedSenderMessage, c.lastSend())
	assert.Empty(t, c.copies, "blocked sender's message must not be delivered")
}

func TestRouteToOwnerRateLimited(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t) // rate max 3 per window

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	for range 3 {
		d.routeToOwner(ctx, c, testMsg(5, 50, "spam"))
	}
	assert.Equal(t, messageSentMessage, c.lastSend())

	d.routeToOwner(ctx, c, testMsg(5, 50, "spam over quota"))
	assert.Equal(t, rateLimitedMessage, c.lastSend())
	assert.Len(t, c.copies, 3, "over-quota message must not be delivered")
}

func TestTryRouteReplyOwnerRepliesAreNotRateLimited(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t) // rate max 3 per window

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello owner"))
	deliveredID := c.nextMsgID

	// Owner answers the same sender well past RATE_LIMIT_MAX times.
	for range 10 {
		reply := testMsg(owner.TgUserID, owner.ChatID, "answer")
		reply.ReplyToMessage = &models.Message{ID: deliveredID}
		require.True(t, d.tryRouteReply(ctx, c, reply))
	}

	assert.Equal(t, replySentMessage, c.lastSend())
	assert.Len(t, c.copies, 11, "every owner reply must be delivered")
}

func TestTryRouteReplyInboundRepliesStayRateLimited(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t) // rate max 3 per window

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	// First inbound message (quota 1/3), then an owner reply so the sender has
	// something to reply to.
	d.routeToOwner(ctx, c, testMsg(5, 50, "hi"))
	deliveredID := c.nextMsgID
	ownerReply := testMsg(owner.TgUserID, owner.ChatID, "hi back")
	ownerReply.ReplyToMessage = &models.Message{ID: deliveredID}
	require.True(t, d.tryRouteReply(ctx, c, ownerReply))
	relayedID := c.nextMsgID

	// Sender replies via the reply path: quota reaches 3, then the 4th is refused.
	for range 2 {
		r := testMsg(5, 50, "more")
		r.ReplyToMessage = &models.Message{ID: relayedID}
		require.True(t, d.tryRouteReply(ctx, c, r))
	}
	assert.Equal(t, replySentMessage, c.lastSend())

	over := testMsg(5, 50, "over quota")
	over.ReplyToMessage = &models.Message{ID: relayedID}
	require.True(t, d.tryRouteReply(ctx, c, over))
	assert.Equal(t, rateLimitedMessage, c.lastSend())
}

func TestRouteToOwnerDeliveryFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{copyErr: errors.New("telegram down")}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hi"))

	assert.Equal(t, deliveryFailedMessage, c.lastSend())
}

func TestTryRouteReplyIgnoresNonReplies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)
	c := &fakeClient{}

	consumed := d.tryRouteReply(ctx, c, testMsg(5, 50, "not a reply"))

	assert.False(t, consumed)
	assert.Empty(t, c.sends)
	assert.Empty(t, c.copies)
}

func TestTryRouteReplyIgnoresReplyToUnknownMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)
	c := &fakeClient{}

	msg := testMsg(5, 50, "reply text")
	msg.ReplyToMessage = &models.Message{ID: 777}

	assert.False(t, d.tryRouteReply(ctx, c, msg))
}

func TestTryRouteReplyRoutesOwnerReplyBackToSender(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	// Sender's first message reaches the owner and creates a relay.
	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello owner"))
	deliveredID := c.nextMsgID

	// Owner replies to that delivered copy.
	reply := testMsg(owner.TgUserID, owner.ChatID, "hello back")
	reply.ReplyToMessage = &models.Message{ID: deliveredID}

	consumed := d.tryRouteReply(ctx, c, reply)

	assert.True(t, consumed)
	assert.Equal(t, replySentMessage, c.lastSend())
	require.Len(t, c.copies, 2)
	assert.Equal(t, int64(50), c.copies[1].ChatID, "reply routed back to the sender's chat")
}

func TestTryRouteReplyBlockedSenderCannotReply(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "hello"))
	deliveredID := c.nextMsgID

	// Owner replies once so a relay exists in the sender's chat too.
	ownerReply := testMsg(owner.TgUserID, owner.ChatID, "hi")
	ownerReply.ReplyToMessage = &models.Message{ID: deliveredID}
	require.True(t, d.tryRouteReply(ctx, c, ownerReply))
	relayedToSenderID := c.nextMsgID

	require.NoError(t, store.Block(ctx, owner.TgUserID, 50))

	// Sender tries to answer that relayed reply.
	senderReply := testMsg(5, 50, "please unblock me")
	senderReply.ReplyToMessage = &models.Message{ID: relayedToSenderID}
	consumed := d.tryRouteReply(ctx, c, senderReply)

	assert.True(t, consumed)
	assert.Equal(t, blockedSenderMessage, c.lastSend())
}

func TestBlockNeedsReply(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)
	c := &fakeClient{}

	d.block(ctx, c, testMsg(10, 1000, "/block"))

	assert.Equal(t, blockNeedsReplyMessage, c.lastSend())
}

func TestBlockOnUnknownMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, _ := newTestDeps(t)
	c := &fakeClient{}

	msg := testMsg(10, 1000, "/block")
	msg.ReplyToMessage = &models.Message{ID: 12345}
	d.block(ctx, c, msg)

	assert.Equal(t, blockUnknownMessage, c.lastSend())
}

func TestBlockStopsFurtherMessagesFromSender(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)

	owner, err := store.UpsertUser(ctx, 10, 1000)
	require.NoError(t, err)
	require.NoError(t, store.SetSession(ctx, 50, owner.TgUserID))

	c := &fakeClient{}
	d.routeToOwner(ctx, c, testMsg(5, 50, "first contact"))
	deliveredID := c.nextMsgID

	// Owner blocks by replying /block to the delivered message.
	blockMsg := testMsg(owner.TgUserID, owner.ChatID, "/block")
	blockMsg.ReplyToMessage = &models.Message{ID: deliveredID}
	d.block(ctx, c, blockMsg)
	assert.Equal(t, blockedMessage, c.lastSend())

	blocked, err := store.IsBlocked(ctx, owner.TgUserID, 50)
	require.NoError(t, err)
	assert.True(t, blocked)

	// The sender's next message is refused.
	c2 := &fakeClient{}
	d.routeToOwner(ctx, c2, testMsg(5, 50, "hello again"))
	assert.Equal(t, blockedSenderMessage, c2.lastSend())
	assert.Empty(t, c2.copies)
}

func TestFullConversationRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d, store := newTestDeps(t)
	c := &fakeClient{}

	// Owner registers and gets a link.
	d.registerRecipient(ctx, c, testMsg(10, 1000, "/start"))
	owner, ok, err := store.UserByID(ctx, 10)
	require.NoError(t, err)
	require.True(t, ok)

	// Sender opens the link.
	d.joinByToken(ctx, c, testMsg(20, 2000, "/start "+owner.LinkToken), owner.LinkToken)

	// Sender writes; it lands with the owner.
	d.routeToOwner(ctx, c, testMsg(20, 2000, "anonymous question"))
	toOwner := c.copies[len(c.copies)-1]
	assert.Equal(t, owner.ChatID, toOwner.ChatID)
	deliveredID := c.nextMsgID

	// Owner replies to that copy; it lands back with the sender.
	reply := testMsg(10, 1000, "anonymous answer")
	reply.ReplyToMessage = &models.Message{ID: deliveredID}
	require.True(t, d.tryRouteReply(ctx, c, reply))
	toSender := c.copies[len(c.copies)-1]
	assert.Equal(t, int64(2000), toSender.ChatID)
}
