package middleware

import (
	"context"
	"fmt"

	"github.com/nightnoryu/anon3anon/pkg/anon3anon/infrastructure/jsonlog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	chatIDField   = "chat_id"
	usernameField = "username"
)

func NewLoggingMiddleware(logger jsonlog.Logger) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			chatLogger := logger.
				WithField(chatIDField, update.Message.Chat.ID).
				WithField(usernameField, update.Message.From.Username)

			text := update.Message.Text
			if len(update.Message.Caption) > 0 {
				text = update.Message.Caption
			}

			chatLogger.Info(fmt.Sprintf("new message: %s", text))

			next(ctx, bot, update)
		}
	}
}
