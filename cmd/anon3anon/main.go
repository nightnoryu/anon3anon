package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/runtime"

	"anon3anon/pkg/infrastructure/health"
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

	startHealthServer(ctx, conf.HealthAddr, store, logger)

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

func startHealthServer(ctx context.Context, addr string, store *sqlite.Store, logger log.Logger) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           health.Handler(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error(err)
		}
	}()
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
			{Command: handler.CommandStart, Description: "Получить свою персональную ссылку"},
			{Command: handler.CommandHelp, Description: "Как пользоваться ботом"},
			{Command: handler.CommandMyLink, Description: "Показать текущую персональную ссылку"},
			{Command: handler.CommandRevoke, Description: "Отозвать ссылку и выпустить новую"},
			{Command: handler.CommandBlock, Description: "Ответом на сообщение - заблокировать отправителя"},
			{Command: handler.CommandStop, Description: "Выйти из текущей переписки"},
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
		bot.WithMiddlewares(
			middleware.NewPrivateChatMiddleware(),
			middleware.NewLoggingMiddleware(logger),
		),
		bot.WithMessageTextHandler(handler.CommandStart, bot.MatchTypeCommand, handler.NewStartCommandHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandHelp, bot.MatchTypeCommand, handler.NewHelpHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandMyLink, bot.MatchTypeCommand, handler.NewMyLinkHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandRevoke, bot.MatchTypeCommand, handler.NewRevokeHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandBlock, bot.MatchTypeCommand, handler.NewBlockHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandStop, bot.MatchTypeCommand, handler.NewStopHandler(deps)),
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
