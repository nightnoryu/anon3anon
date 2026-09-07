package domain

import "time"

// User is a registered recipient: someone who ran /start and owns a personal link.
type User struct {
	TgUserID  int64
	ChatID    int64
	LinkToken string
	CreatedAt time.Time
}
