package middleware

import (
	"context"
	"fmt"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/jsonlog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewLoggingMiddleware(logger jsonlog.Logger) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message != nil {
				text := update.Message.Text
				if len(update.Message.Caption) > 0 {
					text = update.Message.Caption
				}
				logger.Info(fmt.Sprintf("message from %s: %s", update.Message.From.Username, text))
			}
			next(ctx, bot, update)
		}
	}
}
