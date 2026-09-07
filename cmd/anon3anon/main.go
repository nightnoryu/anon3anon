package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/runtime"

	"anon3anon/pkg/infrastructure/storage/sqlite"
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

	store, err := sqlite.Open(conf.DatabasePath, conf.RateLimitWindow, conf.RateLimitMax)
	if err != nil {
		logger.FatalError(err)
	}
	defer func() {
		if cerr := store.Close(); cerr != nil {
			logger.Error(cerr)
		}
	}()

	options, err := initBotOptions(ctx, conf, store, logger)
	if err != nil {
		logger.FatalError(err)
	}

	b, err := bot.New(conf.TelegramBotToken, options...)
	if err != nil {
		logger.FatalError(err)
	}

	if err := registerCommands(ctx, b); err != nil {
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

func registerCommands(ctx context.Context, b *bot.Bot) error {
	_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Получить свою персональную ссылку"},
			{Command: "mylink", Description: "Показать текущую персональную ссылку"},
			{Command: "revoke", Description: "Отозвать ссылку и выпустить новую"},
			{Command: "block", Description: "Ответом на сообщение - заблокировать отправителя"},
		},
	})
	return err
}

func initBotOptions(ctx context.Context, conf *config, store *sqlite.Store, logger log.Logger) ([]bot.Option, error) {
	username, err := resolveBotUsername(ctx, conf.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	deps := handler.DependencyContainer{
		Store:        store,
		Logger:       logger,
		BotUsername:  username,
		AllowedUsers: handler.NewAllowList(conf.AllowedUserIDs),
	}

	return []bot.Option{
		bot.WithMiddlewares(middleware.NewLoggingMiddleware(logger)),
		bot.WithMessageTextHandler("start", bot.MatchTypeCommand, handler.NewStartCommandHandler(deps)),
		bot.WithMessageTextHandler("mylink", bot.MatchTypeCommand, handler.NewMyLinkHandler(deps)),
		bot.WithMessageTextHandler("revoke", bot.MatchTypeCommand, handler.NewRevokeHandler(deps)),
		bot.WithMessageTextHandler("block", bot.MatchTypeCommand, handler.NewBlockHandler(deps)),
		bot.WithDefaultHandler(handler.NewMessageRouter(deps)),
	}, nil
}

func resolveBotUsername(ctx context.Context, botToken string) (string, error) {
	probe, err := bot.New(botToken, bot.WithSkipGetMe())
	if err != nil {
		return "", err
	}
	me, err := probe.GetMe(ctx)
	if err != nil {
		return "", err
	}
	return me.Username, nil
}
