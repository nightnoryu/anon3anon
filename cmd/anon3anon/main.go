package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conf, err := parseEnv()
	if err != nil {
		log.Fatal(err)
	}

	opts := []bot.Option{
		bot.WithDebug(),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, getStartHandler()),
		bot.WithDefaultHandler(getDefaultHandler(conf)),
	}

	b, err := bot.New(conf.TelegramBotToken, opts...)
	if err != nil {
		log.Fatal(err)
	}

	b.Start(ctx)
}

func getStartHandler() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		params := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Жду твоих сообщений!!\nОтветы будут в канале @meme_me_a_meme",
		}
		_, err := b.SendMessage(ctx, params)
		if err != nil {
			log.Print(err)
		}
	}
}

func getDefaultHandler(conf *config) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		params := &bot.CopyMessageParams{
			ChatID:     conf.OwnerChatID,
			FromChatID: update.Message.Chat.ID,
			MessageID:  update.Message.ID,
		}
		_, err := b.CopyMessage(ctx, params)
		if err != nil {
			log.Print(err)
		}

		sendParams := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Сообщение отправлено!",
		}
		_, err = b.SendMessage(ctx, sendParams)
		if err != nil {
			log.Print(err)
		}
	}
}
