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
	messageSentReply       = "*Сообщение отправлено!*"
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

func (s *anonymousMessagesService) pingClient(chatID int64) error {
	return s.api.SendMessage(chatID, Message{
		Text:        messageSentReply,
		UseMarkdown: true,
	})
}

func (s *anonymousMessagesService) handleMessage(message Message) error {
	msgText := newMessageNotification + "\n\n" + message.Text
	return s.api.SendMessageToOwner(Message{
		Text:  msgText,
		Image: message.Image,
		Video: message.Video,
	})
}
