package main

import (
	"strings"

	"github.com/go-faster/errors"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func initLogger(level jsonlog.Level) log.MainLogger {
	return jsonlog.NewLogger(&jsonlog.Config{
		AppName: appID,
		Level:   level,
	})
}

func parseLogLevel(name string) (jsonlog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return jsonlog.DebugLevel, nil
	case "", "info":
		return jsonlog.InfoLevel, nil
	case "warn", "warning":
		return jsonlog.WarnLevel, nil
	case "error":
		return jsonlog.ErrorLevel, nil
	default:
		return defaultLogLevel, errors.Errorf("unknown log level %q: use debug, info, warn or error", name)
	}
}
