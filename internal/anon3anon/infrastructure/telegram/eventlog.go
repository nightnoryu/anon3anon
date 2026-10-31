package telegram

import (
	"context"
	"time"

	"github.com/nightnoryu/go-kita/log"
)

type eventLogKey struct{}

type eventLog struct {
	fields    log.Fields
	startedAt time.Time
}

func WithEventLog(ctx context.Context, fields log.Fields) context.Context {
	return context.WithValue(ctx, eventLogKey{}, eventLog{
		fields:    fields,
		startedAt: time.Now(),
	})
}

func EventLogger(ctx context.Context, logger log.Logger) log.Logger {
	event, ok := ctx.Value(eventLogKey{}).(eventLog)
	if !ok {
		return logger
	}

	fields := make(log.Fields, len(event.fields)+2)
	for key, value := range event.fields {
		fields[key] = value
	}
	fields["duration_ms"] = float64(time.Since(event.startedAt)) / float64(time.Millisecond)
	fields["outcome"] = string(OutcomeFromContext(ctx))
	return logger.WithFields(fields)
}
