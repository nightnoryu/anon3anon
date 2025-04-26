package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/nightnoryu/anon3anon/pkg/app"

	"github.com/go-telegram/bot"
	"github.com/sirupsen/logrus"
)

const appID = "anon3anon"

func main() {
	logger := initLogger()

	conf, err := parseEnv()
	if err != nil {
		logger.Fatal(err)
	}

	opts := []bot.Option{
		bot.WithDebug(),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, app.GetStartCommandHandler(logger)),
		bot.WithDefaultHandler(app.GetAnonymousMessagesHandler(logger, conf.OwnerChatID)),
	}

	b, err := bot.New(conf.TelegramBotToken, opts...)
	if err != nil {
		logger.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b.Start(ctx)
}

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.WarnLevel)
	return logger
}
