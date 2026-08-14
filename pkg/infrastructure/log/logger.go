package log

type Fields map[string]any

// Logger - simplified logger abstraction
type Logger interface {
	WithFields(Fields) Logger
	Debug(...any)
	Info(...any)
	Error(error, ...any)
}

// MainLogger - Logger which can also report fatal errors
type MainLogger interface {
	Logger
	FatalError(error, ...any)
}
