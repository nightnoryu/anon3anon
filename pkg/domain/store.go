package domain

import "context"

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

	// SetSession points a sender's chat at the owner they are currently
	// messaging (the last link they opened wins).
	SetSession(ctx context.Context, senderChatID, ownerUserID int64) error
	// GetSession returns the owner a sender's chat is currently messaging.
	GetSession(ctx context.Context, senderChatID int64) (ownerUserID int64, ok bool, err error)

	// PutRelay stores a delivered-message mapping.
	PutRelay(ctx context.Context, r Relay) error
	// LookupRelay finds the relay for a message that was replied to.
	LookupRelay(ctx context.Context, destChatID int64, destMsgID int) (Relay, bool, error)

	// AllowMessage records one message from senderID to recipientID in the
	// current time bucket and reports whether the sender is still within the
	// per-bucket quota for that recipient.
	AllowMessage(ctx context.Context, senderID, recipientID int64) (bool, error)
}
