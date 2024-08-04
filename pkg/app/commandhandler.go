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
		msgText = "Жду твоих вопросов!!\nОтветы будут в канале @meme_me_a_meme (>ᴗ•)"
	case InfoCommand:
		msgText = "Бот привязан к каналу @meme_me_a_meme, так что ответы ищи там!!\nНа данный момент поддерживаются текст, фото и видео („• ᴗ •„)"
	case UnknownCommand:
		msgText = "Неизвестная команда!"
	}

	return h.api.SendMessage(update.FromChatID, Message{Text: msgText})
}
