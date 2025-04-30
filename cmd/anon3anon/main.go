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

	opts := []bot.Option{
		bot.WithMiddlewares(middleware.NewLoggingMiddleware(logger)),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, handler.NewStartCommandHandler(logger)),
		bot.WithDefaultHandler(handler.NewAnonymousMessagesHandler(logger, conf.OwnerChatID)),
	}

	b, err := bot.New(conf.TelegramBotToken, opts...)
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
