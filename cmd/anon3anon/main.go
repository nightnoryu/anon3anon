package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/jsonlog"
	"github.com/nightnoryu/anon3anon/pkg/infrastructure/telegram/handler"
	"github.com/nightnoryu/anon3anon/pkg/infrastructure/telegram/middleware"

	"github.com/go-telegram/bot"
)

const appID = "anon3anon"

func main() {
	logger := initLogger()

	conf, err := parseEnv()
	if err != nil {
		logger.FatalError(err)
	}

	options := initBotOptions(conf, logger)
	b, err := bot.New(conf.TelegramBotToken, options...)
	if err != nil {
		logger.FatalError(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b.Start(ctx)
}

func initLogger() jsonlog.Logger {
	logger := jsonlog.NewLogger(&jsonlog.Config{
		AppName: appID,
		Level:   jsonlog.InfoLevel,
	})
	return logger
}

func initBotOptions(conf *config, logger jsonlog.Logger) []bot.Option {
	startCommandHandler := handler.NewStartCommandHandler(logger)
	anonymousMessagesHandler := handler.NewAnonymousMessagesHandler(logger, conf.OwnerChatID)

	return []bot.Option{
		bot.WithMiddlewares(middleware.NewLoggingMiddleware(logger)),
		bot.WithMessageTextHandler("start", bot.MatchTypeCommand, startCommandHandler),
		bot.WithDefaultHandler(anonymousMessagesHandler),
	}
}
