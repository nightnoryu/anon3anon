package main

import (
	"context"
	"errors"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/retry"

	"anon3anon/internal/anon3anon/infrastructure/storage/sqlite"
	handler2 "anon3anon/internal/anon3anon/infrastructure/telegram/handler"
	middleware2 "anon3anon/internal/anon3anon/infrastructure/telegram/middleware"
	"anon3anon/internal/common/pseudonym"
)

const (
	appID                      = "anon3anon"
	telegramStartupMaxAttempts = 3
	telegramStartupRetryDelay  = time.Second
)

func main() {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	conf, err := env.ParseEnv[config](appID)
	if err != nil {
		stdlog.Fatal(err)
	}
	messages, err := handler2.MessagesForLanguage(conf.Language)
	if err != nil {
		stdlog.Fatal(err)
	}

	logger, err := initLogger(conf.LogLevel)
	if err != nil {
		stdlog.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	keys, err := initKeyring(conf.PseudonymKey)
	if err != nil {
		logger.FatalError(err)
	}

	store, err := sqlite.Open(conf.DatabasePath, conf.RateLimitWindow, conf.RateLimitMax, keys)
	if err != nil {
		logger.FatalError(err)
	}
	defer func() {
		if cerr := store.Close(); cerr != nil {
			logger.Error(cerr)
		}
	}()

	metrics := newAppMetrics()
	startHealthServer(ctx, conf.HealthAddr, store, logger, metrics)
	startRetentionSweeper(ctx, conf, store, logger, metrics)

	options, err := initBotOptions(ctx, conf, store, keys, logger, messages, metrics)
	if err != nil {
		logger.FatalError(err)
	}

	b, err := bot.New(conf.TelegramBotToken, options...)
	if err != nil {
		logger.FatalError(err)
	}

	if err := registerCommands(ctx, b, messages); err != nil {
		logger.FatalError(err)
	}

	b.Start(ctx)
}

func registerCommands(ctx context.Context, b *bot.Bot, messages handler2.Messages) error {
	return retryTelegramStartup(ctx, func(ctx context.Context) error {
		_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
			Commands: messages.CommandDescriptions(),
		})
		return err
	})
}

func initBotOptions(
	ctx context.Context,
	conf *config,
	store *sqlite.Store,
	keys *pseudonym.Keyring,
	logger log.Logger,
	messages handler2.Messages,
	metrics *appMetrics,
) ([]bot.Option, error) {
	username, err := resolveBotUsername(ctx, conf.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	deps := handler2.DependencyContainer{
		Store:        store,
		Logger:       logger,
		BotUsername:  username,
		AllowedUsers: handler2.NewAllowList(conf.AllowedUserIDs),
		Messages:     messages,
		OwnerLink:    conf.OwnerLink,
	}

	return []bot.Option{
		bot.WithSkipGetMe(),
		bot.WithErrorsHandler(func(err error) {
			logger.Error(err)
		}),
		bot.WithMiddlewares(
			middleware2.NewPrivateChatMiddleware(),
			middleware2.NewLoggingMiddleware(logger, keys, handler2.CommandNames(), metrics.observeMessage),
		),
		bot.WithMessageTextHandler(handler2.CommandStart, bot.MatchTypeCommandStartOnly, handler2.NewStartCommandHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandHelp, bot.MatchTypeCommandStartOnly, handler2.NewHelpHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandMyLink, bot.MatchTypeCommandStartOnly, handler2.NewMyLinkHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandRevoke, bot.MatchTypeCommandStartOnly, handler2.NewRevokeHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandBlock, bot.MatchTypeCommandStartOnly, handler2.NewBlockHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandStop, bot.MatchTypeCommandStartOnly, handler2.NewStopHandler(deps)),
		bot.WithMessageTextHandler(handler2.CommandDelete, bot.MatchTypeCommandStartOnly, handler2.NewDeleteHandler(deps)),
		bot.WithDefaultHandler(handler2.NewMessageRouter(deps)),
	}, nil
}

func resolveBotUsername(ctx context.Context, botToken string) (string, error) {
	probe, err := bot.New(botToken, bot.WithSkipGetMe())
	if err != nil {
		return "", err
	}

	var username string
	err = retryTelegramStartup(ctx, func(ctx context.Context) error {
		me, getMeErr := probe.GetMe(ctx)
		if getMeErr != nil {
			return getMeErr
		}
		username = me.Username
		return nil
	})
	if err != nil {
		return "", err
	}
	return username, nil
}

func retryTelegramStartup(ctx context.Context, operation func(context.Context) error) error {
	retryer, err := retry.New(retry.Config{
		MaxAttempts:  telegramStartupMaxAttempts,
		InitialDelay: telegramStartupRetryDelay,
		Multiplier:   2,
	})
	if err != nil {
		return err
	}

	return retryer.Do(ctx, operation, isRetryableTelegramError)
}

func isRetryableTelegramError(err error) bool {
	return !errors.Is(err, context.Canceled) &&
		!errors.Is(err, bot.ErrorBadRequest) &&
		!errors.Is(err, bot.ErrorUnauthorized) &&
		!errors.Is(err, bot.ErrorForbidden) &&
		!errors.Is(err, bot.ErrorNotFound) &&
		!errors.Is(err, bot.ErrorConflict)
}
