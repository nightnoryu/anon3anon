package app

type BotAPI interface {
	HandleUpdates(handler MessageUpdateHandler) error
	SendMessage(chatID int64, message Message) error
	SendMessageToOwner(message Message) error
}

type MessageUpdateHandler func(MessageUpdate)

type MessageUpdate struct {
	Message
	UpdateID   int
	FromChatID int64
	Command    *Command
}

type Message struct {
	Text  string
	Image *Image
	Video *Video
}

type Image struct {
	FileID string
}

type Video struct {
	FileID string
}

type Command int

const (
	UnknownCommand Command = iota
	StartCommand
	InfoCommand
)
