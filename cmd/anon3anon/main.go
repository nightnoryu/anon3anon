package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nightnoryu/anon3anon/pkg/app"
	"github.com/nightnoryu/anon3anon/pkg/infrastructure"
)

func main() {
	conf, err := parseEnv()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(conf.TelegramBotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	errorsChan := make(chan error)
	botAPI := infrastructure.NewBotAPI(bot, conf.OwnerChatID)

	commandHandler := app.NewCommandHandler(botAPI)

	service := app.NewAnonymousQuestionsService(errorsChan, commandHandler, botAPI)
	go func() {
		for err := range errorsChan {
			log.Println(err)
		}
	}()

	if err = service.ServeMessages(); err != nil {
		log.Fatal(err)
	}
}
