package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/runtime"

	"anon3anon/pkg/infrastructure/telegram/handler"
	"anon3anon/pkg/infrastructure/telegram/middleware"
)

const appID = "anon3anon"

func main() {
	ctx := runtime.ListenOSKillSignals(context.Background())

	logger := initLogger()

	conf, err := env.ParseEnv[config](appID)
	if err != nil {
		logger.FatalError(err)
	}

	options := initBotOptions(conf, logger)
	b, err := bot.New(conf.TelegramBotToken, options...)
	if err != nil {
		logger.FatalError(err)
	}

	b.Start(ctx)
}

func initLogger() log.MainLogger {
	logger := jsonlog.NewLogger(&jsonlog.Config{
		AppName: appID,
		Level:   jsonlog.InfoLevel,
	})
	return logger
}

func initBotOptions(conf *config, logger log.Logger) []bot.Option {
	startCommandHandler := handler.NewStartCommandHandler(logger)
	anonymousMessagesHandler := handler.NewAnonymousMessagesHandler(logger, conf.OwnerChatID)

	return []bot.Option{
		bot.WithMiddlewares(middleware.NewLoggingMiddleware(logger)),
		bot.WithMessageTextHandler("start", bot.MatchTypeCommand, startCommandHandler),
		bot.WithDefaultHandler(anonymousMessagesHandler),
	}
}
