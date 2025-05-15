package jsonlog

import "github.com/sirupsen/logrus"

type Level logrus.Level

const (
	FatalLevel = Level(logrus.FatalLevel)
	ErrorLevel = Level(logrus.ErrorLevel)
	WarnLevel  = Level(logrus.WarnLevel)
	InfoLevel  = Level(logrus.InfoLevel)
	DebugLevel = Level(logrus.DebugLevel)
)
