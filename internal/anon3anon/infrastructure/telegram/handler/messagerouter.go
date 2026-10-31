package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewMessageRouter(d DependencyContainer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		msg := update.Message
		if msg == nil || msg.From == nil {
			return
		}
		if d.tryRouteReply(ctx, b, msg) {
			return
		}
		d.routeToOwner(ctx, b, msg)
	}
}
