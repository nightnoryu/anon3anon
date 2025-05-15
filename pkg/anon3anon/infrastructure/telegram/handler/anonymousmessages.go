package handler

import (
	"context"

	"github.com/nightnoryu/anon3anon/pkg/anon3anon/infrastructure/jsonlog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewAnonymousMessagesHandler(logger jsonlog.Logger, ownerChatId int) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		params := &bot.CopyMessageParams{
			ChatID:     ownerChatId,
			FromChatID: update.Message.Chat.ID,
			MessageID:  update.Message.ID,
		}
		_, err := b.CopyMessage(ctx, params)
		if err != nil {
			logger.Error(err)
			return
		}

		sendParams := &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   messageSentMessage,
		}
		_, err = b.SendMessage(ctx, sendParams)
		if err != nil {
			logger.Error(err)
		}
	}
}
