package app

func NewAnonymousQuestionsService(api BotAPI, errorsChan chan error) AnonymousMessagesService {
	return &anonymousMessagesService{
		api:        api,
		errorsChan: errorsChan,
	}
}

const (
	messageSentReply       = "*Сообщение отправлено!*"
	newMessageNotification = "*Новое анонимное сообщение!*"
)

type AnonymousMessagesService interface {
	ServeMessages() error
}

type anonymousMessagesService struct {
	api        BotAPI
	errorsChan chan error
}

func (s *anonymousMessagesService) ServeMessages() error {
	return s.api.HandleUpdates(func(update MessageUpdate) {
		if update.Command != nil {
			err := s.handleCommand(update)
			if err != nil {
				s.errorsChan <- err
			}
			return
		}

		err := s.handleMessage(update.Message)
		if err != nil {
			s.errorsChan <- err
		}

		err = s.pingClient(update.FromChatID)
		if err != nil {
			s.errorsChan <- err
		}
	})
}

func (s *anonymousMessagesService) handleCommand(update MessageUpdate) error {
	if update.Command == nil {
		return nil
	}

	var msgText string
	switch *update.Command {
	case StartCommand:
		msgText = "Жду твоих вопросов!"
	case UnknownCommand:
		msgText = "Неизвестная команда!"
	}

	return s.api.SendMessage(update.FromChatID, Message{Text: msgText})
}

func (s *anonymousMessagesService) pingClient(chatID int64) error {
	return s.api.SendMessage(chatID, Message{Text: messageSentReply})
}

func (s *anonymousMessagesService) handleMessage(message Message) error {
	msgText := newMessageNotification + "\n\n" + message.Text
	return s.api.SendMessageToOwner(Message{
		Text:  msgText,
		Image: message.Image,
	})
}
