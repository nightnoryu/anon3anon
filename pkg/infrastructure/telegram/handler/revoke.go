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
		d.revokeLink(ctx, b, update.Message)
	}
}

func (d DependencyContainer) revokeLink(ctx context.Context, c telegramClient, msg *models.Message) {
	if _, ok, err := d.Store.UserByID(ctx, msg.From.ID); err != nil {
		d.Logger.Error(err)
		return
	} else if !ok {
		d.registerRecipient(ctx, c, msg)
		return
	}

	newToken, err := d.Store.RotateToken(ctx, msg.From.ID)
	if err != nil {
		d.Logger.Error(err)
		return
	}

	if _, err := d.Store.ClearSessionsForOwner(ctx, msg.From.ID); err != nil {
		d.Logger.Error(err)
		return
	}

	d.reply(ctx, c, msg.Chat.ID, fmt.Sprintf(newLinkMessageTemplate, buildMyLink(d.BotUsername, newToken)))
}
