package middleware

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/log"
)

const (
	chatIDField   = "chat_id"
	usernameField = "username"
)

func NewLoggingMiddleware(logger log.Logger) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			chatLogger := logger.WithFields(log.Fields{
				chatIDField:   update.Message.Chat.ID,
				usernameField: update.Message.Chat.Username,
			})

			text := update.Message.Text
			if update.Message.Caption != "" {
				text = update.Message.Caption
			}

			chatLogger.Info(fmt.Sprintf("new message: %s", text))

			next(ctx, bot, update)
		}
	}
}
