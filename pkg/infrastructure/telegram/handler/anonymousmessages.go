package handler

import (
	"context"
	"fmt"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/jsonlog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NewAnonymousMessagesHandler(logger jsonlog.Logger, ownerChatID int) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		if ownerChatID == 0 {
			logger.Info(fmt.Sprintf("owner chat ID not set. set to %d to use the last chat", update.Message.Chat.ID))
			return
		}

		params := &bot.CopyMessageParams{
			ChatID:     ownerChatID,
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
