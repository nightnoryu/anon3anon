package app

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
)

func GetStartCommandHandler(logger *logrus.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		params := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Жду твоих сообщений!!",
		}
		_, err := b.SendMessage(ctx, params)
		if err != nil {
			logger.Error(err)
		}
	}
}

func GetAnonymousMessagesHandler(logger *logrus.Logger, ownerChatId int) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		params := &bot.CopyMessageParams{
			ChatID:     ownerChatId,
			FromChatID: update.Message.Chat.ID,
			MessageID:  update.Message.ID,
		}
		_, err := b.CopyMessage(ctx, params)
		if err != nil {
			logger.Error(err)
			return
		}

		sendParams := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Сообщение отправлено!!",
		}
		_, err = b.SendMessage(ctx, sendParams)
		if err != nil {
			logger.Error(err)
		}
	}
}
