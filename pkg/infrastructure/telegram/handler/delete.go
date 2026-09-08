package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewDeleteHandler(d DependencyContainer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}
		d.deleteAccount(ctx, b, update.Message)
	}
}

func (d DependencyContainer) deleteAccount(ctx context.Context, c telegramClient, msg *models.Message) {
	deleted, err := d.Store.DeleteUser(ctx, msg.From.ID)
	if err != nil {
		d.Logger.Error(err)
		d.reply(ctx, c, msg.Chat.ID, accountDeleteFailedMessage)
		return
	}
	if !deleted {
		d.reply(ctx, c, msg.Chat.ID, noAccountToDeleteMessage)
		return
	}
	d.reply(ctx, c, msg.Chat.ID, accountDeletedMessage)
}
