package main

import "time"

type config struct {
	TelegramBotToken string        `env:"TELEGRAM_BOT_TOKEN"`
	DatabasePath     string        `env:"DATABASE_PATH" envDefault:"/data/anon3anon.db"`
	AllowedUserIDs   []int64       `env:"ALLOWED_USER_IDS" envSeparator:","`
	RateLimitWindow  time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1h"`
	RateLimitMax     int           `env:"RATE_LIMIT_MAX" envDefault:"100"`
	HealthAddr       string        `env:"HEALTH_ADDR" envDefault:":8080"`
	LogLevel         string        `env:"LOG_LEVEL" envDefault:"info"`

	PseudonymKey string `env:"PSEUDONYM_KEY"`

	RetentionAge           time.Duration `env:"RETENTION_AGE" envDefault:"720h"`
	RetentionSweepInterval time.Duration `env:"RETENTION_SWEEP_INTERVAL" envDefault:"1h"`
}
