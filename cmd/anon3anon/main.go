package main

import (
	"context"
	stdlog "log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/runtime"

	"anon3anon/pkg/infrastructure/storage/sqlite"
	"anon3anon/pkg/infrastructure/telegram/handler"
	"anon3anon/pkg/infrastructure/telegram/middleware"
	"anon3anon/pkg/pseudonym"
)

const (
	appID           = "anon3anon"
	defaultLogLevel = jsonlog.InfoLevel
)

func main() {
	ctx := runtime.ListenOSKillSignals(context.Background())

	conf, err := env.ParseEnv[config](appID)
	if err != nil {
		stdlog.Fatal(err)
	}

	level, err := parseLogLevel(conf.LogLevel)
	if err != nil {
		stdlog.Fatal(err)
	}
	logger := initLogger(level)

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

	options, err := initBotOptions(ctx, conf, store, keys, logger)
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

func registerCommands(ctx context.Context, b *bot.Bot) error {
	_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: handler.CommandStart, Description: "Получить свою персональную ссылку"},
			{Command: handler.CommandHelp, Description: "Как пользоваться ботом"},
			{Command: handler.CommandMyLink, Description: "Показать текущую персональную ссылку"},
			{Command: handler.CommandRevoke, Description: "Отозвать ссылку и выпустить новую"},
			{Command: handler.CommandBlock, Description: "Ответом на сообщение - заблокировать отправителя"},
			{Command: handler.CommandStop, Description: "Выйти из текущей переписки"},
			{Command: handler.CommandDelete, Description: "Удалить аккаунт и все связанные данные"},
		},
	})
	return err
}

func initBotOptions(
	ctx context.Context,
	conf *config,
	store *sqlite.Store,
	keys *pseudonym.Keyring,
	logger log.Logger,
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
	}

	return []bot.Option{
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
	me, err := probe.GetMe(ctx)
	if err != nil {
		return "", err
	}
	return me.Username, nil
}
