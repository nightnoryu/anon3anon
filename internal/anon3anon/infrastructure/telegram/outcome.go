package telegram

import "context"

type Outcome string

const (
	OutcomeSuccess     Outcome = "success"
	OutcomeRateLimited Outcome = "rate_limited"
	OutcomeBlocked     Outcome = "blocked"
	OutcomeError       Outcome = "error"
)

type outcomeKey struct{}

type outcomeTracker struct {
	outcome Outcome
}

func WithOutcome(ctx context.Context) context.Context {
	return context.WithValue(ctx, outcomeKey{}, &outcomeTracker{outcome: OutcomeSuccess})
}

func SetOutcome(ctx context.Context, outcome Outcome) {
	tracker, ok := ctx.Value(outcomeKey{}).(*outcomeTracker)
	if !ok {
		return
	}

	if !validOutcome(outcome) {
		outcome = OutcomeError
	}
	if tracker.outcome == OutcomeError {
		return
	}
	tracker.outcome = outcome
}

func OutcomeFromContext(ctx context.Context) Outcome {
	tracker, ok := ctx.Value(outcomeKey{}).(*outcomeTracker)
	if !ok || !validOutcome(tracker.outcome) {
		return OutcomeSuccess
	}
	return tracker.outcome
}

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomeSuccess, OutcomeRateLimited, OutcomeBlocked, OutcomeError:
		return true
	default:
		return false
	}
}
