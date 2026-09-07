package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"anon3anon/pkg/domain"
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

func (d DependencyContainer) tryRouteReply(ctx context.Context, c telegramClient, msg *models.Message) bool {
	if msg.ReplyToMessage == nil {
		return false
	}

	relay, ok, err := d.Store.LookupRelay(ctx, msg.Chat.ID, msg.ReplyToMessage.ID)
	if err != nil {
		d.Logger.Error(err)
		return true
	}
	if !ok {
		return false
	}

	if err := d.relay(ctx, c, msg, relay.OriginChatID, relay.OwnerUserID); err != nil {
		d.Logger.Error(err)
		d.reply(ctx, c, msg.Chat.ID, deliveryFailedMessage)
	}
	return true
}

func (d DependencyContainer) routeToOwner(ctx context.Context, c telegramClient, msg *models.Message) {
	ownerUserID, ok, err := d.Store.GetSession(ctx, msg.Chat.ID)
	if err != nil {
		d.Logger.Error(err)
		return
	}
	if !ok {
		d.reply(ctx, c, msg.Chat.ID, noSessionMessage)
		return
	}

	owner, ok, err := d.Store.UserByID(ctx, ownerUserID)
	if err != nil {
		d.Logger.Error(err)
		return
	}
	if !ok {
		d.reply(ctx, c, msg.Chat.ID, noSessionMessage)
		return
	}

	if err := d.relay(ctx, c, msg, owner.ChatID, owner.TgUserID); err != nil {
		d.Logger.Error(err)
		d.reply(ctx, c, msg.Chat.ID, deliveryFailedMessage)
		return
	}
	d.reply(ctx, c, msg.Chat.ID, messageSentMessage)
}

func (d DependencyContainer) relay(ctx context.Context, c telegramClient, src *models.Message, destChatID, ownerUserID int64) error {
	copied, err := c.CopyMessage(ctx, &bot.CopyMessageParams{
		ChatID:     destChatID,
		FromChatID: src.Chat.ID,
		MessageID:  src.ID,
	})
	if err != nil {
		return err
	}
	return d.Store.PutRelay(ctx, domain.Relay{
		DestChatID:   destChatID,
		DestMsgID:    copied.ID,
		OriginChatID: src.Chat.ID,
		OwnerUserID:  ownerUserID,
	})
}
