package main

type config struct {
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	// DatabasePath is where the embedded SQLite database file lives.
	DatabasePath string `env:"DATABASE_PATH" envDefault:"/data/anon3anon.db"`
	// AllowedUserIDs restricts who may register as a recipient. Empty means
	// anyone can. Comma-separated Telegram user IDs.
	AllowedUserIDs []int64 `env:"ALLOWED_USER_IDS" envSeparator:","`
}
