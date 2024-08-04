package infrastructure

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/nightnoryu/anon3anon/pkg/app"
)

const (
	updateTimeoutInSeconds = 60
	messageParseMode       = tgbotapi.ModeMarkdown

	startCommand = "start"
	infoCommand  = "info"
)

func NewBotAPI(bot *tgbotapi.BotAPI, ownerChatID int64) app.BotAPI {
	return &botAPI{
		bot:         bot,
		ownerChatID: ownerChatID,
	}
}

type botAPI struct {
	bot         *tgbotapi.BotAPI
	ownerChatID int64
}

func (api *botAPI) HandleUpdates(handler app.MessageUpdateHandler) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTimeoutInSeconds

	updates := api.bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("%+v\n", update.Message)

		messageUpdate := app.MessageUpdate{
			Message:    api.hydrateMessage(update.Message),
			UpdateID:   update.UpdateID,
			FromChatID: update.FromChat().ID,
			Command:    api.hydrateCommand(update.Message),
		}

		handler(messageUpdate)
	}

	return nil
}

func (api *botAPI) SendMessage(chatID int64, message app.Message) error {
	if message.Image != nil {
		return api.sendPhotoMessage(chatID, message)
	}

	if message.Video != nil {
		return api.sendVideoMessage(chatID, message)
	}

	return api.sendTextMessage(chatID, message)
}

func (api *botAPI) SendMessageToOwner(message app.Message) error {
	return api.SendMessage(api.ownerChatID, message)
}

func (api *botAPI) hydrateMessage(msg *tgbotapi.Message) app.Message {
	text := msg.Text
	if len(text) == 0 {
		text = msg.Caption
	}

	return app.Message{
		Text:  text,
		Image: api.hydrateImage(msg.Photo),
		Video: api.hydrateVideo(msg.Video),
	}
}

func (api *botAPI) sendTextMessage(chatID int64, message app.Message) error {
	msg := tgbotapi.NewMessage(
		chatID,
		message.Text,
	)
	msg.ParseMode = messageParseMode

	_, err := api.bot.Send(msg)
	return errors.WithStack(err)
}

func (api *botAPI) sendPhotoMessage(chatID int64, message app.Message) error {
	photos := api.preparePhotos(message)
	mediaMsg := tgbotapi.NewMediaGroup(chatID, photos)

	_, err := api.bot.Send(mediaMsg)
	return errors.WithStack(err)
}

func (api *botAPI) sendVideoMessage(chatID int64, message app.Message) error {
	video := api.prepareVideo(message)
	mediaMsg := tgbotapi.NewMediaGroup(chatID, video)

	_, err := api.bot.Send(mediaMsg)
	return errors.WithStack(err)
}

func (api *botAPI) hydrateCommand(msg *tgbotapi.Message) *app.Command {
	if !msg.IsCommand() {
		return nil
	}

	var cmd app.Command
	switch msg.Command() {
	case startCommand:
		cmd = app.StartCommand
	case infoCommand:
		cmd = app.InfoCommand
	default:
		cmd = app.UnknownCommand
	}

	return &cmd
}

func (api *botAPI) hydrateImage(photos []tgbotapi.PhotoSize) *app.Image {
	if len(photos) == 0 {
		return nil
	}

	var originalFileID string
	var originalFileSize int
	for _, photo := range photos {
		if photo.FileSize > originalFileSize {
			originalFileID = photo.FileID
			originalFileSize = photo.FileSize
		}
	}

	return &app.Image{
		FileID: originalFileID,
	}
}

func (api *botAPI) hydrateVideo(video *tgbotapi.Video) *app.Video {
	if video == nil {
		return nil
	}

	return &app.Video{
		FileID: video.FileID,
	}
}

func (api *botAPI) preparePhotos(message app.Message) []interface{} {
	photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(message.Image.FileID))
	photo.Caption = message.Text
	photo.ParseMode = messageParseMode

	var photos []interface{}
	photos = append(photos, photo)

	return photos
}

func (api *botAPI) prepareVideo(message app.Message) []interface{} {
	video := tgbotapi.NewInputMediaVideo(tgbotapi.FileID(message.Video.FileID))
	video.Caption = message.Text
	video.ParseMode = messageParseMode
	return []interface{}{video}
}
