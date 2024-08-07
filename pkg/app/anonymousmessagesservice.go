package app

func NewAnonymousQuestionsService(
	errorsChan chan error,
	commandHandler CommandHandler,
	api BotAPI,
) AnonymousMessagesService {
	return &anonymousMessagesService{
		errorsChan:     errorsChan,
		commandHandler: commandHandler,
		api:            api,
	}
}

const (
	messageSentReply        = "*Сообщение отправлено!*"
	unsupportedMessageReply = "*Такое сообщение не поддерживается :(*"

	newMessageNotification = "Новое анонимное сообщение!"
)

type AnonymousMessagesService interface {
	ServeMessages() error
}

type anonymousMessagesService struct {
	errorsChan     chan error
	commandHandler CommandHandler
	api            BotAPI
}

func (s *anonymousMessagesService) ServeMessages() error {
	return s.api.HandleUpdates(func(update MessageUpdate) {
		if update.Command != nil {
			err := s.commandHandler.HandleCommand(update)
			if err != nil {
				s.errorsChan <- err
			}
			return
		}

		if update.Message.Sticker != nil {
			err := s.api.SendMessage(update.FromChatID, Message{
				Text:        unsupportedMessageReply,
				UseMarkdown: true,
			})
			if err != nil {
				s.errorsChan <- err
			}
			return
		}

		notificationText := newMessageNotification + "\n\n" + update.Message.Text
		err := s.api.SendMessageToOwner(Message{
			Text:  notificationText,
			Image: update.Message.Image,
			Video: update.Message.Video,
		})
		if err != nil {
			s.errorsChan <- err
		}

		err = s.api.SendMessage(update.FromChatID, Message{
			Text:        messageSentReply,
			UseMarkdown: true,
		})
		if err != nil {
			s.errorsChan <- err
		}
	})
}
