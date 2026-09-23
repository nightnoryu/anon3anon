package middleware

import (
	"context"
	"strings"
	"time"

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
	eventTypeField   = "event_type"
	commandField     = "command"
	durationMSField  = "duration_ms"

	eventTypeAnonymousMessage = "anonymous_message"
	eventTypeReply            = "reply"
	eventTypeCommandCall      = "command_call"
	unknownCommand            = "unknown"
)

func NewLoggingMiddleware(logger log.Logger, keys *pseudonym.Keyring, supportedCommands []string) bot.Middleware {
	commands := make(map[string]struct{}, len(supportedCommands))
	for _, command := range supportedCommands {
		commands[command] = struct{}{}
	}

	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			msg := update.Message
			kind := messageKind(msg)
			eventType, command := eventDetails(msg, commands)
			startedAt := time.Now()

			next(ctx, bot, update)

			fields := log.Fields{
				updateIDField:    update.ID,
				chatRefField:     keys.Ref(msg.Chat.ID),
				chatTypeField:    string(msg.Chat.Type),
				messageIDField:   msg.ID,
				messageKindField: kind,
				isReplyField:     msg.ReplyToMessage != nil,
				hasMediaField:    kind != messageKindText && kind != messageKindCommand,
				eventTypeField:   eventType,
				durationMSField:  float64(time.Since(startedAt)) / float64(time.Millisecond),
			}
			if command != "" {
				fields[commandField] = command
			}
			logger.WithFields(fields).Info("telegram event processed")

			logIdentifiableMessage(logger, update, msg)
		}
	}
}

func eventDetails(m *models.Message, supportedCommands map[string]struct{}) (eventType, command string) {
	if command, ok := commandName(m); ok {
		if _, supported := supportedCommands[command]; !supported {
			command = unknownCommand
		}
		return eventTypeCommandCall, command
	}
	if m.ReplyToMessage != nil {
		return eventTypeReply, ""
	}
	return eventTypeAnonymousMessage, ""
}

func commandName(m *models.Message) (string, bool) {
	for _, entity := range m.Entities {
		if entity.Type != models.MessageEntityTypeBotCommand || entity.Offset != 0 || entity.Length < 2 {
			continue
		}
		if entity.Length > len(m.Text) {
			return "", false
		}

		command := strings.TrimPrefix(m.Text[:entity.Length], "/")
		command, _, _ = strings.Cut(command, "@")
		return command, command != ""
	}
	return "", false
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
