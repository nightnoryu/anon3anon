package main

import (
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nightnoryu/anon3anon/pkg/app"
)

const (
	telegramBotTokenEnvKey = "TELEGRAM_BOT_TOKEN"
	ownerChatIDEnvKey      = "OWNER_CHAT_ID"
)

type config struct {
	Token       string
	OwnerChatID int64
}

func getConfig() (config, error) {
	token := os.Getenv(telegramBotTokenEnvKey)
	ownerChatID, err := strconv.ParseInt(os.Getenv(ownerChatIDEnvKey), 10, 64)
	if err != nil {
		return config{}, err
	}

	return config{
		Token:       token,
		OwnerChatID: ownerChatID,
	}, nil
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Panic(err)
	}

	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	service := app.NewAnonymousQuestionsService(bot, config.OwnerChatID)

	if err := service.ListenForMessages(); err != nil {
		log.Panic(err)
	}
}
