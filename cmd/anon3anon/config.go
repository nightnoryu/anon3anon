package main

import "time"

type config struct {
	TelegramBotToken string        `env:"TELEGRAM_BOT_TOKEN"`
	DatabasePath     string        `env:"DATABASE_PATH" envDefault:"/data/anon3anon.db"`
	AllowedUserIDs   []int64       `env:"ALLOWED_USER_IDS" envSeparator:","`
	RateLimitWindow  time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1h"`
	RateLimitMax     int           `env:"RATE_LIMIT_MAX" envDefault:"100"`
}
