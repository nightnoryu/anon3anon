package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewHelpHandler(d DependencyContainer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}
		d.sendHelp(ctx, b, update.Message)
	}
}

// sendHelp replies with usage instructions covering both the recipient and the
// sender flows.
func (d DependencyContainer) sendHelp(ctx context.Context, c telegramClient, msg *models.Message) {
	d.reply(ctx, c, msg.Chat.ID, helpMessage)
}
