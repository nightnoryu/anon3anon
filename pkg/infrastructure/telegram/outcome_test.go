package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutcomeDefaultsToSuccess(t *testing.T) {
	t.Parallel()

	assert.Equal(t, OutcomeSuccess, OutcomeFromContext(WithOutcome(context.Background())))
}

func TestSetOutcomeKeepsValuesBoundedAndPreservesErrors(t *testing.T) {
	t.Parallel()

	ctx := WithOutcome(context.Background())
	SetOutcome(ctx, OutcomeRateLimited)
	assert.Equal(t, OutcomeRateLimited, OutcomeFromContext(ctx))

	SetOutcome(ctx, Outcome("unbounded"))
	assert.Equal(t, OutcomeError, OutcomeFromContext(ctx))

	SetOutcome(ctx, OutcomeBlocked)
	assert.Equal(t, OutcomeError, OutcomeFromContext(ctx))
}
