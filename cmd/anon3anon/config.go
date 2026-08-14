package main

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

func parseEnv() (*config, error) {
	c := new(config)
	if err := env.ParseWithOptions(c, env.Options{Prefix: strings.ToUpper(appID) + "_"}); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}
	return c, nil
}

type config struct {
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	OwnerChatID      int    `env:"OWNER_CHAT_ID"`
}
