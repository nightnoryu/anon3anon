package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewStartCommandHandler(d DependencyContainer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}

		if payload := commandPayload(update.Message.Text); payload != "" {
			d.joinByToken(ctx, b, update.Message, payload)
			return
		}
		d.registerRecipient(ctx, b, update.Message)
	}
}
