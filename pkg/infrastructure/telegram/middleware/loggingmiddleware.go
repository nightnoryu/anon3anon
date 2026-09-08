package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/log"

	"anon3anon/pkg/pseudonym"
)

const (
	messageKindText    = "text"
	messageKindCommand = "command"

	updateIDField    = "update_id"
	chatRefField     = "chat_ref"
	chatTypeField    = "chat_type"
	chatIDField      = "chat_id"
	userIDField      = "user_id"
	usernameField    = "username"
	messageIDField   = "message_id"
	messageKindField = "message_kind"
	isReplyField     = "is_reply"
	hasMediaField    = "has_media"
	textField        = "text"
)

func NewLoggingMiddleware(logger log.Logger, keys *pseudonym.Keyring) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			msg := update.Message
			kind := messageKind(msg)

			logger.WithFields(log.Fields{
				updateIDField:    update.ID,
				chatRefField:     keys.Ref(msg.Chat.ID),
				chatTypeField:    string(msg.Chat.Type),
				messageIDField:   msg.ID,
				messageKindField: kind,
				isReplyField:     msg.ReplyToMessage != nil,
				hasMediaField:    kind != messageKindText && kind != messageKindCommand,
			}).Info("new message")

			logIdentifiableMessage(logger, update, msg)

			next(ctx, bot, update)
		}
	}
}

func logIdentifiableMessage(logger log.Logger, update *models.Update, msg *models.Message) {
	text := msg.Text
	if msg.Caption != "" {
		text = msg.Caption
	}

	var (
		userID   int64
		username = msg.Chat.Username
	)
	if msg.From != nil {
		userID = msg.From.ID
		if msg.From.Username != "" {
			username = msg.From.Username
		}
	}

	logger.WithFields(log.Fields{
		updateIDField:  update.ID,
		chatIDField:    msg.Chat.ID,
		userIDField:    userID,
		usernameField:  username,
		messageIDField: msg.ID,
		textField:      text,
	}).Debug("new message content")
}

// messageKind classifies a message for logging: a bot command, a specific media
// type, or plain text.
func messageKind(m *models.Message) string {
	if len(m.Entities) > 0 && m.Entities[0].Type == models.MessageEntityTypeBotCommand {
		return messageKindCommand
	}

	switch {
	case len(m.Photo) > 0:
		return "photo"
	case m.Document != nil:
		return "document"
	case m.Voice != nil:
		return "voice"
	case m.Video != nil:
		return "video"
	case m.VideoNote != nil:
		return "video_note"
	case m.Sticker != nil:
		return "sticker"
	case m.Animation != nil:
		return "animation"
	default:
		return messageKindText
	}
}
