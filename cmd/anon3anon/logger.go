package main

import (
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func initLogger(level jsonlog.Level) (log.MainLogger, error) {
	return jsonlog.NewLogger(&jsonlog.Config{
		AppName: appID,
		Level:   level,
	})
}
