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

// WithOutcome adds an outcome tracker to ctx. A message is successful unless a
// handler records a more specific terminal result.
func WithOutcome(ctx context.Context) context.Context {
	return context.WithValue(ctx, outcomeKey{}, &outcomeTracker{outcome: OutcomeSuccess})
}

// SetOutcome records outcome for the current message. Invalid values are
// normalized to error so log cardinality remains bounded. Error is terminal:
// it must not be hidden by a later outcome assignment.
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

// OutcomeFromContext returns the current message outcome, or success when the
// context was not initialized by the logging middleware.
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
