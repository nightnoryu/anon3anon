package jsonlog

import "go.uber.org/zap/zapcore"

type Level zapcore.Level

const (
	FatalLevel = Level(zapcore.FatalLevel)
	ErrorLevel = Level(zapcore.ErrorLevel)
	WarnLevel  = Level(zapcore.WarnLevel)
	InfoLevel  = Level(zapcore.InfoLevel)
	DebugLevel = Level(zapcore.DebugLevel)
)
