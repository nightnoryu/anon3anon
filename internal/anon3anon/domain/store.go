package domain

import (
	"context"
	"time"
)

type PurgeStats struct {
	Sessions     int64
	Relays       int64
	Blocks       int64
	MessageRates int64
}

type Store interface {
	// UpsertUser returns the existing user for tgUserID, creating one with a
	// freshly generated link token if none exists. chatID is refreshed on every
	// call so the bot can always reach the user.
	UpsertUser(ctx context.Context, tgUserID, chatID int64) (User, error)

	UserByToken(ctx context.Context, tokenValue string) (User, bool, error)
	UserByID(ctx context.Context, tgUserID int64) (User, bool, error)

	RotateToken(ctx context.Context, tgUserID int64) (string, error)

	// DeleteUser erases the recipient tgUserID and every row tied to them or to
	// their chat. It is a no-op for a tgUserID that is not a registered recipient.
	DeleteUser(ctx context.Context, tgUserID int64) (deleted bool, err error)

	SetSession(ctx context.Context, senderChatID, ownerUserID int64) error
	GetSession(ctx context.Context, senderChatID int64) (ownerUserID int64, ok bool, err error)
	// TouchSession bumps the sender's session freshness so an active
	// conversation is not pruned by PurgeExpired while it is still in use. It is
	// a no-op if the sender has no session.
	TouchSession(ctx context.Context, senderChatID int64) error
	ClearSession(ctx context.Context, senderChatID int64) error
	ClearSessionsForOwner(ctx context.Context, ownerUserID int64) (sessionsRemoved int64, err error)

	PurgeExpired(ctx context.Context, cutoff time.Time) (PurgeStats, error)

	ClearRelaysForOwner(ctx context.Context, ownerUserID int64) (relaysRemoved int64, err error)
	ClearRelaysForSender(ctx context.Context, senderChatID, ownerUserID int64) (relaysRemoved int64, err error)

	PutRelay(ctx context.Context, r Relay) error
	LookupRelay(ctx context.Context, destChatID int64, destMsgID int) (Relay, bool, error)

	// AllowMessage records one message from senderID to recipientID in the
	// current time bucket and reports whether the sender is still within the
	// per-bucket quota for that recipient.
	AllowMessage(ctx context.Context, senderID, recipientID int64) (bool, error)

	// RefundMessage returns one unit of quota to the senderID->recipientID
	// bucket after a message counted by AllowMessage failed to be delivered, so
	// a failed relay does not burn the sender's budget. No-op when rate limiting is disabled.
	RefundMessage(ctx context.Context, senderID, recipientID int64) error

	Block(ctx context.Context, ownerUserID, senderChatID int64) error
	IsBlocked(ctx context.Context, ownerUserID, senderChatID int64) (bool, error)
}
