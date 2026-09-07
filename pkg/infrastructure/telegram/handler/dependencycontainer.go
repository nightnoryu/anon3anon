package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/log"

	"anon3anon/pkg/domain"
)

type telegramClient interface {
	CopyMessage(ctx context.Context, params *bot.CopyMessageParams) (*models.MessageID, error)
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

type DependencyContainer struct {
	Store        domain.Store
	Logger       log.Logger
	BotUsername  string
	AllowedUsers AllowList
}

type AllowList map[int64]struct{}

func NewAllowList(ids []int64) AllowList {
	set := make(AllowList, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func (a AllowList) Allowed(id int64) bool {
	if len(a) == 0 {
		return true
	}
	_, ok := a[id]
	return ok
}

func (d DependencyContainer) reply(ctx context.Context, c telegramClient, chatID int64, text string) {
	if _, err := c.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		d.Logger.Error(err)
	}
}
