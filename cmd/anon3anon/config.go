package main

type config struct {
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	OwnerChatID      int    `env:"OWNER_CHAT_ID"`
}
