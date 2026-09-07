package handler

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewRevokeHandler(d DependencyContainer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}
		msg := update.Message

		if _, ok, err := d.Store.UserByID(ctx, msg.From.ID); err != nil {
			d.Logger.Error(err)
			return
		} else if !ok {
			d.registerRecipient(ctx, b, msg)
			return
		}

		newToken, err := d.Store.RotateToken(ctx, msg.From.ID)
		if err != nil {
			d.Logger.Error(err)
			return
		}
		d.reply(ctx, b, msg.Chat.ID, fmt.Sprintf(newLinkMessageTemplate, buildMyLink(d.BotUsername, newToken)))
	}
}
