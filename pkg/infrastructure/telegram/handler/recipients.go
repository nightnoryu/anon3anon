package handler

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot/models"
)

// registerRecipient registers the sender as a message recipient (subject to the
// allow list) and replies with their personal link.
func (d DependencyContainer) registerRecipient(ctx context.Context, c telegramClient, msg *models.Message) {
	if !d.AllowedUsers.Allowed(msg.From.ID) {
		d.reply(ctx, c, msg.Chat.ID, notAllowedMessage)
		return
	}

	user, err := d.Store.UpsertUser(ctx, msg.From.ID, msg.Chat.ID)
	if err != nil {
		d.Logger.Error(err)
		return
	}

	d.reply(ctx, c, msg.Chat.ID, fmt.Sprintf(linkMessageTemplate, buildMyLink(d.BotUsername, user.LinkToken)))
}

// joinByToken points the sender's chat at the owner of the given link token so
// their subsequent messages are relayed there.
func (d DependencyContainer) joinByToken(ctx context.Context, c telegramClient, msg *models.Message, tokenValue string) {
	owner, ok, err := d.Store.UserByToken(ctx, tokenValue)
	if err != nil {
		d.Logger.Error(err)
		return
	}
	if !ok {
		d.reply(ctx, c, msg.Chat.ID, invalidLinkMessage)
		return
	}
	if owner.TgUserID == msg.From.ID {
		d.reply(ctx, c, msg.Chat.ID, ownLinkMessage)
		return
	}

	if err := d.Store.SetSession(ctx, msg.Chat.ID, owner.TgUserID); err != nil {
		d.Logger.Error(err)
		return
	}
	d.reply(ctx, c, msg.Chat.ID, joinedMessage)
}
