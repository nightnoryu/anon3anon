package app

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NewAnonymousQuestionsService(bot *tgbotapi.BotAPI, ownerChatID int64) AnonymousMessagesService {
	return &anonymousMessagesService{
		bot:         bot,
		ownerChatID: ownerChatID,
	}
}

type AnonymousMessagesService interface {
	ListenForMessages() error
}

type anonymousMessagesService struct {
	bot         *tgbotapi.BotAPI
	ownerChatID int64
}

func (s *anonymousMessagesService) ListenForMessages() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.Text == "/start" {
			reply := tgbotapi.NewMessage(update.Message.Chat.ID, "Жду твоих вопросов!")
			s.bot.Send(reply)
			continue
		}

		if len(update.Message.Photo) > 0 {
			var photos []interface{}
			photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(update.Message.Photo[0].FileID))
			photo.Caption = "*Новое анонимное сообщение!*"
			photo.ParseMode = tgbotapi.ModeMarkdown
			photos = append(photos, photo)

			mediaMsg := tgbotapi.NewMediaGroup(s.ownerChatID, photos)

			s.bot.Send(mediaMsg)
		} else {
			msg := tgbotapi.NewMessage(s.ownerChatID, "*Новое анонимное сообщение!*\n\n"+update.Message.Text)
			msg.ParseMode = tgbotapi.ModeMarkdown

			s.bot.Send(msg)
		}

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, "Сообщение отправлено!")

		s.bot.Send(reply)
	}

	return nil
}
