package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"anon3anon/pkg/domain"
)

// tryRouteReply handles the message when it is a reply to a message the bot
// previously delivered, routing it back to that message's origin. It reports
// whether it consumed the message.
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
	d.reply(ctx, c, msg.Chat.ID, replySentMessage)
	return true
}

// routeToOwner relays a first-contact message to the owner the sender's chat is
// currently pointed at, or explains that there is no active conversation.
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

// relay copies src into destChatID and records the mapping needed to route a
// reply back to src's chat.
func (d DependencyContainer) relay(
	ctx context.Context, c telegramClient, src *models.Message, destChatID, ownerUserID int64,
) error {
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
