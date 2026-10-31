package domain

import "time"

type User struct {
	TgUserID  int64
	ChatID    int64
	LinkToken string
	CreatedAt time.Time
}
