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

// Store persists bot state: registered users, per-sender routing sessions,
// and the relay map that makes threaded replies possible.
type Store interface {
	// UpsertUser returns the existing user for tgUserID, creating one with a
	// freshly generated link token if none exists. chatID is refreshed on every
	// call so the bot can always reach the user.
	UpsertUser(ctx context.Context, tgUserID, chatID int64) (User, error)

	UserByToken(ctx context.Context, tokenValue string) (User, bool, error)
	UserByID(ctx context.Context, tgUserID int64) (User, bool, error)

	RotateToken(ctx context.Context, tgUserID int64) (string, error)

	// DeleteUser erases the recipient tgUserID and every row tied to them or to
	// their chat: their user record, routing sessions in both directions, relay
	// mappings, blocks, and rate-limit buckets. It is a no-op for a tgUserID
	// that is not a registered recipient. It is idempotent and reports whether a
	// user record was deleted.
	DeleteUser(ctx context.Context, tgUserID int64) (deleted bool, err error)

	// SetSession points a sender's chat at the owner they are currently
	// messaging (the last link they opened wins).
	SetSession(ctx context.Context, senderChatID, ownerUserID int64) error
	// GetSession returns the owner a sender's chat is currently messaging.
	GetSession(ctx context.Context, senderChatID int64) (ownerUserID int64, ok bool, err error)
	// TouchSession bumps the sender's session freshness so an active
	// conversation is not pruned by PurgeExpired while it is still in use. It is
	// a no-op if the sender has no session.
	TouchSession(ctx context.Context, senderChatID int64) error
	// ClearSession drops the sender's routing session so their messages are no
	// longer relayed anywhere until they open a link again. It is idempotent.
	ClearSession(ctx context.Context, senderChatID int64) error
	// ClearSessionsForOwner drops every routing session pointed at ownerUserID,
	// cutting off senders who already opened a now-revoked link. It is
	// idempotent and returns the number of sessions removed.
	ClearSessionsForOwner(ctx context.Context, ownerUserID int64) (int64, error)

	// PurgeExpired removes rows whose retention window has closed: sessions last
	// updated before cutoff, and relays, blocks, and message-rate buckets from
	// before cutoff. It bounds how long the sender<->recipient linkage is
	// retained and keeps every one of those tables from growing without limit.
	// It is idempotent and reports how many rows it removed from each table.
	PurgeExpired(ctx context.Context, cutoff time.Time) (PurgeStats, error)

	// ClearRelaysForOwner drops every delivered-message mapping in
	// ownerUserID's conversations. Clearing a sender's session alone does not
	// cut them off: a reply to a message the bot already delivered is routed
	// from the relay map, which no session is consulted for. It is idempotent
	// and returns the number of mappings removed.
	ClearRelaysForOwner(ctx context.Context, ownerUserID int64) (int64, error)
	// ClearRelaysForSender drops the mappings tying senderChatID to
	// ownerUserID in either direction, leaving that owner's conversations with
	// everyone else intact. It is idempotent and returns the number removed.
	ClearRelaysForSender(ctx context.Context, senderChatID, ownerUserID int64) (int64, error)

	// PutRelay stores a delivered-message mapping.
	PutRelay(ctx context.Context, r Relay) error
	// LookupRelay finds the relay for a message that was replied to.
	LookupRelay(ctx context.Context, destChatID int64, destMsgID int) (Relay, bool, error)

	// AllowMessage records one message from senderID to recipientID in the
	// current time bucket and reports whether the sender is still within the
	// per-bucket quota for that recipient.
	AllowMessage(ctx context.Context, senderID, recipientID int64) (bool, error)

	// RefundMessage returns one unit of quota to the senderID->recipientID
	// bucket after a message counted by AllowMessage failed to be delivered, so
	// a failed relay does not burn the sender's budget. It never drives the
	// count below zero and is a no-op when rate limiting is disabled.
	RefundMessage(ctx context.Context, senderID, recipientID int64) error

	// Block records that owner ownerUserID no longer wants to receive messages
	// from the anonymous sender in senderChatID. It is idempotent.
	Block(ctx context.Context, ownerUserID, senderChatID int64) error
	// IsBlocked reports whether owner ownerUserID has blocked senderChatID.
	IsBlocked(ctx context.Context, ownerUserID, senderChatID int64) (bool, error)
}
