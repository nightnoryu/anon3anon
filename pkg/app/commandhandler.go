package app

func NewCommandHandler(api BotAPI) CommandHandler {
	return &commandHandler{
		api: api,
	}
}

type CommandHandler interface {
	HandleCommand(update MessageUpdate) error
}

type commandHandler struct {
	api BotAPI
}

func (h *commandHandler) HandleCommand(update MessageUpdate) error {
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

	return h.api.SendMessage(update.FromChatID, Message{Text: msgText})
}
