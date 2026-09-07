package domain

import "time"

// Relay records a single message the bot delivered from one chat to another, so
// that a reply to that delivered message can be routed back to its origin.
//
// It is written for both directions of a conversation: when a sender's message
// is copied to an owner, and when an owner's reply is copied back to a sender.
// A reply is matched by (chat where the reply happened, replied-to message ID).
type Relay struct {
	DestChatID   int64 // chat the message was delivered to
	DestMsgID    int   // message ID of the delivered copy within DestChatID
	OriginChatID int64 // chat a reply to that copy must be forwarded to
	OwnerUserID  int64 // owner whose conversation this mapping belongs to
	CreatedAt    time.Time
}
