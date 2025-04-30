package jsonlog

import (
	"time"

	"github.com/sirupsen/logrus"
)

const appNameKey = "app_name"

var fieldMap = logrus.FieldMap{
	logrus.FieldKeyTime: "@timestamp",
	logrus.FieldKeyMsg:  "message",
}

type Logger interface {
	WithField(key string, value interface{}) Logger

	Info(...interface{})
	Error(error, ...interface{})
	FatalError(error, ...interface{})
}

type Config struct {
	AppName string
	Level   Level
}

func NewLogger(config *Config) Logger {
	impl := logrus.New()
	impl.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap:        fieldMap,
	})
	impl.SetLevel(logrus.Level(config.Level))
	return &logger{
		FieldLogger: impl.WithField(appNameKey, config.AppName),
	}
}

type logger struct {
	logrus.FieldLogger
}

func (l *logger) WithField(key string, value interface{}) Logger {
	return &logger{l.FieldLogger.WithField(key, value)}
}

func (l *logger) Error(err error, args ...interface{}) {
	l.FieldLogger.WithError(err).Error(args)
}

func (l *logger) FatalError(err error, args ...interface{}) {
	l.FieldLogger.WithError(err).Fatal(args...)
}
