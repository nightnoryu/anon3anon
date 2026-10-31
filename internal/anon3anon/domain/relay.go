package domain

import "time"

type Relay struct {
	DestChatID   int64
	DestMsgID    int
	OriginChatID int64
	OwnerUserID  int64
	CreatedAt    time.Time
}
