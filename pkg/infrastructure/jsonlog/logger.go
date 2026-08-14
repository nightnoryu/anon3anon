package jsonlog

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nightnoryu/anon3anon/pkg/infrastructure/log"
)

const appNameKey = "app_name"

type logger struct {
	*zap.Logger
}

func NewLogger(config *Config) log.MainLogger {
	implConfig := zap.NewProductionConfig()
	implConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(config.Level))
	impl := zap.Must(implConfig.Build())
	return &logger{
		Logger: impl.With(zap.String(appNameKey, config.AppName)),
	}
}

func (l *logger) WithFields(fields log.Fields) log.Logger {
	implFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		implFields = append(implFields, zap.Any(key, value))
	}
	return &logger{l.With(implFields...)}
}

func (l *logger) Debug(args ...any) {
	l.Logger.Debug(fmt.Sprint(args...))
}

func (l *logger) Info(args ...any) {
	l.Logger.Info(fmt.Sprint(args...))
}

func (l *logger) Error(err error, args ...any) {
	l.With(zap.Error(err)).Error(fmt.Sprint(args...))
}

func (l *logger) FatalError(err error, args ...any) {
	l.With(zap.Error(err)).Fatal(fmt.Sprint(args...))
}
