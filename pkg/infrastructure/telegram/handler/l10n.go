package handler

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

const (
	LanguageRussian = "ru"
	LanguageEnglish = "en"
)

type Messages struct {
	messageSent, replySent, newAnonymousMessagePrefix      string
	joined, invalidLink, ownLink                           string
	notAllowed, notAllowedWithOwnerLink                    string
	noSession, deliveryFailed, rateLimited                 string
	blocked, blockNeedsReply, blockUnknown, blockedSender  string
	stopped, noSessionToStop, unsupportedContent           string
	accountDeleted, noAccountToDelete, accountDeleteFailed string
	help, linkTemplate, newLinkTemplate                    string
	startCommandDescription, helpCommandDescription        string
	myLinkCommandDescription, revokeCommandDescription     string
	blockCommandDescription, stopCommandDescription        string
	deleteCommandDescription                               string
}

func MessagesForLanguage(language string) (Messages, error) {
	switch language {
	case LanguageRussian:
		return russianMessages, nil
	case LanguageEnglish:
		return englishMessages, nil
	default:
		return Messages{}, fmt.Errorf("unsupported language %q: must be %q or %q", language, LanguageRussian, LanguageEnglish)
	}
}

func (m Messages) CommandDescriptions() []models.BotCommand {
	return []models.BotCommand{
		{Command: CommandStart, Description: m.startCommandDescription},
		{Command: CommandHelp, Description: m.helpCommandDescription},
		{Command: CommandMyLink, Description: m.myLinkCommandDescription},
		{Command: CommandRevoke, Description: m.revokeCommandDescription},
		{Command: CommandBlock, Description: m.blockCommandDescription},
		{Command: CommandStop, Description: m.stopCommandDescription},
		{Command: CommandDelete, Description: m.deleteCommandDescription},
	}
}

func (m Messages) registrationNotAllowed(ownerLink string) string {
	if ownerLink == "" {
		return m.notAllowed
	}
	return fmt.Sprintf(m.notAllowedWithOwnerLink, ownerLink)
}

var russianMessages = Messages{
	messageSent: "Сообщение отправлено!!", replySent: "Ответ отправлен!!", newAnonymousMessagePrefix: "Новое анонимное сообщение!!",
	joined:      "Теперь можешь писать сюда анонимные сообщения - я передам их адресату. Ответы придут в этот же чат.",
	invalidLink: "Ссылка недействительна. Проверь, что скопировал её целиком.", ownLink: "Это твоя собственная ссылка. Отправь её кому-нибудь другому!",
	notAllowed: "Регистрация новых получателей закрыта.", notAllowedWithOwnerLink: "Регистрация новых получателей закрыта. Свяжитесь с владельцем бота: %s",
	noSession:      "Сначала открой чью-нибудь персональную ссылку, чтобы начать переписку. Хочешь свою - нажми /start.",
	deliveryFailed: "Не удалось доставить сообщение - возможно, пользователь заблокировал бота.", rateLimited: "Притормози, ковбой! Слишком много сообщений этому адресату, попробуй позже.",
	blocked:         "Банхаммер прописан! Больше не будешь получать сообщения от этого отправителя.",
	blockNeedsReply: "Чтобы заблокировать отправителя, ответь командой /block на его анонимное сообщение.",
	blockUnknown:    "Не могу понять, кого блокировать. Ответь /block прямо на анонимное сообщение.", blockedSender: "Этот адресат вас заблокировал, ему больше нельзя написать :(",
	stopped:             "Ты вышел из переписки. Новые сообщения никуда не уйдут, пока не откроешь чью-нибудь персональную ссылку снова.",
	noSessionToStop:     "Ты сейчас не в переписке - выходить не из чего. Открой чью-нибудь персональную ссылку, чтобы начать.",
	unsupportedContent:  "Контакт и геопозицию я не передаю. Отправь текст, фото, файл или голосовое.",
	accountDeleted:      "Аккаунт удалён. Твоя ссылка больше не работает, все сессии, ответы и блокировки стёрты. Нажми /start, если захочешь начать заново.",
	noAccountToDelete:   "У тебя нет зарегистрированного аккаунта - удалять нечего. Нажми /start, чтобы получить персональную ссылку.",
	accountDeleteFailed: "Не удалось удалить аккаунт, попробуй ещё раз. Если не проходит - напиши позже.",
	help: "Этот бот передаёт анонимные сообщения между людьми в обе стороны.\n\n" +
		"Если ты получатель:\n1. Нажми /start - бот выдаст твою персональную ссылку.\n2. Поделись ссылкой с теми, от кого хочешь получать анонимные сообщения.\n3. Их сообщения придут в этот чат. Чтобы ответить, используй «ответить» на сообщение - ответ уйдёт отправителю анонимно.\n4. /mylink - показать ссылку снова, /revoke - выпустить новую взамен старой, /block ответом на сообщение - заблокировать его отправителя.\n5. /delete - удалить аккаунт и все связанные данные без возможности восстановления.\n\n" +
		"Если ты отправитель:\n1. Открой чью-нибудь персональную ссылку - бот подтвердит, что можно писать.\n2. Пиши сообщения как обычно - они придут адресату анонимно.\n3. Ответы адресата придут в этот чат. Отвечай на них через «ответить», чтобы продолжить переписку.\n4. /stop - выйти из текущей переписки.",
	linkTemplate:            "Твоя персональная ссылка:\n%s\n\nОтправь её тем, от кого хочешь получать анонимные сообщения. Их сообщения придут сюда, а твои ответы (через «ответить» на сообщение) я передам обратно анонимно.",
	newLinkTemplate:         "Готово, старая ссылка деактивирована, а те, кто уже писал по ней, отключены. Ответить на их прошлые сообщения теперь тоже нельзя - переписка обрывается в обе стороны. Новая ссылка:\n%s",
	startCommandDescription: "Получить свою персональную ссылку", helpCommandDescription: "Как пользоваться ботом", myLinkCommandDescription: "Показать текущую персональную ссылку", revokeCommandDescription: "Отозвать ссылку и выпустить новую", blockCommandDescription: "Ответом на сообщение - заблокировать отправителя", stopCommandDescription: "Выйти из текущей переписки", deleteCommandDescription: "Удалить аккаунт и все связанные данные",
}

var englishMessages = Messages{
	messageSent: "Message sent!", replySent: "Reply sent!", newAnonymousMessagePrefix: "New anonymous message!",
	joined:      "You can now send anonymous messages here and I will deliver them to the recipient. Replies will arrive in this chat.",
	invalidLink: "This link is invalid. Make sure you copied it in full.", ownLink: "This is your own link. Send it to someone else!",
	notAllowed: "Registration of new recipients is restricted.", notAllowedWithOwnerLink: "Registration of new recipients is restricted. Contact the bot owner: %s",
	noSession:      "Open someone's personal link first to start a conversation. Want your own? Use /start.",
	deliveryFailed: "The message could not be delivered. The recipient may have blocked the bot.", rateLimited: "Slow down, cowboy! You have sent too many messages to this recipient. Try again later.",
	blocked:         "Banhammer applied! You will no longer receive messages from this sender.",
	blockNeedsReply: "Reply to an anonymous message with /block to block its sender.", blockUnknown: "I can't determine whom to block. Reply to an anonymous message with /block.", blockedSender: "This recipient has blocked you, so you can no longer message them :(",
	stopped: "You left the conversation. New messages will not be delivered until you open someone's personal link again.", noSessionToStop: "You are not in a conversation, so there is nothing to leave. Open someone's personal link to start one.",
	unsupportedContent: "I cannot relay contacts or locations. Send text, a photo, a file, or a voice message instead.",
	accountDeleted:     "Your account was deleted. Your link no longer works, and all sessions, replies, and blocks were erased. Use /start to begin again.",
	noAccountToDelete:  "You do not have a registered account to delete. Use /start to get your personal link.", accountDeleteFailed: "Your account could not be deleted. Try again later.",
	help: "This bot relays anonymous messages between people in both directions.\n\n" +
		"If you are a recipient:\n1. Use /start to get your personal link.\n2. Share the link with people from whom you want to receive anonymous messages.\n3. Their messages will arrive in this chat. Reply to a message to send an anonymous reply.\n4. Use /mylink to show the link again, /revoke to replace it, or reply with /block to block a sender.\n5. Use /delete to permanently remove your account and related data.\n\n" +
		"If you are a sender:\n1. Open someone's personal link and the bot will confirm that you can write to them.\n2. Send messages normally; they will reach the recipient anonymously.\n3. The recipient's replies will arrive here. Reply to them to continue the conversation.\n4. Use /stop to leave the current conversation.",
	linkTemplate:            "Your personal link:\n%s\n\nShare it with people from whom you want to receive anonymous messages. Their messages will arrive here, and your replies to them will be delivered anonymously.",
	newLinkTemplate:         "Done. Your old link has been deactivated and people who used it have been disconnected. You can no longer reply to their earlier messages either. Your new link:\n%s",
	startCommandDescription: "Get your personal link", helpCommandDescription: "How to use the bot", myLinkCommandDescription: "Show your personal link", revokeCommandDescription: "Replace your personal link", blockCommandDescription: "Reply to block a sender", stopCommandDescription: "Leave the current conversation", deleteCommandDescription: "Delete your account and related data",
}

// Russian compatibility values keep the existing handler tests focused on
// behavior. Production handlers always use DependencyContainer.Messages.
var (
	messageSentMessage        = russianMessages.messageSent
	replySentMessage          = russianMessages.replySent
	newAnonymousMessagePrefix = russianMessages.newAnonymousMessagePrefix
	joinedMessage             = russianMessages.joined
	invalidLinkMessage        = russianMessages.invalidLink
	ownLinkMessage            = russianMessages.ownLink
	notAllowedMessage         = russianMessages.notAllowed
	noSessionMessage          = russianMessages.noSession
	deliveryFailedMessage     = russianMessages.deliveryFailed
	rateLimitedMessage        = russianMessages.rateLimited
	blockedMessage            = russianMessages.blocked
	blockNeedsReplyMessage    = russianMessages.blockNeedsReply
	blockUnknownMessage       = russianMessages.blockUnknown
	blockedSenderMessage      = russianMessages.blockedSender
	stoppedMessage            = russianMessages.stopped
	noSessionToStopMessage    = russianMessages.noSessionToStop
	unsupportedContentMessage = russianMessages.unsupportedContent
	accountDeletedMessage     = russianMessages.accountDeleted
	noAccountToDeleteMessage  = russianMessages.noAccountToDelete
)
