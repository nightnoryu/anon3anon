package handler

import (
	"context"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/jsonlog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewStartCommandHandler(logger jsonlog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		params := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   greetingMessage,
		}
		_, err := b.SendMessage(ctx, params)
		if err != nil {
			logger.Error(err)
		}
	}
}
