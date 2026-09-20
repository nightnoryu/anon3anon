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

	"anon3anon/pkg/infrastructure/storage/sqlite"
	"anon3anon/pkg/infrastructure/telegram/handler"
	"anon3anon/pkg/infrastructure/telegram/middleware"
	"anon3anon/pkg/pseudonym"
)

const (
	appID                     = "anon3anon"
	telegramStartupMaxRetries = 3
	telegramStartupRetryDelay = time.Second
)

func main() {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	conf, err := env.ParseEnv[config](appID)
	if err != nil {
		stdlog.Fatal(err)
	}
	messages, err := handler.MessagesForLanguage(conf.Language)
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

	startHealthServer(ctx, conf.HealthAddr, store, logger)
	startRetentionSweeper(ctx, conf, store, logger)

	options, err := initBotOptions(ctx, conf, store, keys, logger, messages)
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

func registerCommands(ctx context.Context, b *bot.Bot, messages handler.Messages) error {
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
	messages handler.Messages,
) ([]bot.Option, error) {
	username, err := resolveBotUsername(ctx, conf.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	deps := handler.DependencyContainer{
		Store:        store,
		Logger:       logger,
		BotUsername:  username,
		AllowedUsers: handler.NewAllowList(conf.AllowedUserIDs),
		Messages:     messages,
		OwnerLink:    conf.OwnerLink,
	}

	return []bot.Option{
		bot.WithSkipGetMe(),
		bot.WithMiddlewares(
			middleware.NewPrivateChatMiddleware(),
			middleware.NewLoggingMiddleware(logger, keys),
		),
		bot.WithMessageTextHandler(handler.CommandStart, bot.MatchTypeCommandStartOnly, handler.NewStartCommandHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandHelp, bot.MatchTypeCommandStartOnly, handler.NewHelpHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandMyLink, bot.MatchTypeCommandStartOnly, handler.NewMyLinkHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandRevoke, bot.MatchTypeCommandStartOnly, handler.NewRevokeHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandBlock, bot.MatchTypeCommandStartOnly, handler.NewBlockHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandStop, bot.MatchTypeCommandStartOnly, handler.NewStopHandler(deps)),
		bot.WithMessageTextHandler(handler.CommandDelete, bot.MatchTypeCommandStartOnly, handler.NewDeleteHandler(deps)),
		bot.WithDefaultHandler(handler.NewMessageRouter(deps)),
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
	return retryTelegramStartupWithPolicy(
		ctx,
		operation,
		telegramStartupMaxRetries,
		telegramStartupRetryDelay,
	)
}

func retryTelegramStartupWithPolicy(
	ctx context.Context,
	operation func(context.Context) error,
	maxRetries int,
	initialDelay time.Duration,
) error {
	delay := initialDelay
	for retries := 0; ; retries++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if retries == maxRetries || !isRetryableTelegramError(err) {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
	}
}

func isRetryableTelegramError(err error) bool {
	return !errors.Is(err, context.Canceled) &&
		!errors.Is(err, bot.ErrorBadRequest) &&
		!errors.Is(err, bot.ErrorUnauthorized) &&
		!errors.Is(err, bot.ErrorForbidden) &&
		!errors.Is(err, bot.ErrorNotFound) &&
		!errors.Is(err, bot.ErrorConflict)
}
