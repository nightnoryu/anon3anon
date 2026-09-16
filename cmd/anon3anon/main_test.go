package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/stretchr/testify/require"
)

func TestRetryTelegramStartupEventuallySucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := retryTelegramStartupWithPolicy(t.Context(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary proxy failure")
		}
		return nil
	}, 4, 0)

	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func TestRetryTelegramStartupStopsAfterMaximumRetries(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("temporary proxy failure")
	attempts := 0
	err := retryTelegramStartupWithPolicy(t.Context(), func(context.Context) error {
		attempts++
		return wantErr
	}, 2, 0)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 3, attempts)
}

func TestRetryTelegramStartupDoesNotRetryPermanentErrors(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := retryTelegramStartupWithPolicy(t.Context(), func(context.Context) error {
		attempts++
		return bot.ErrorUnauthorized
	}, 4, 0)

	require.ErrorIs(t, err, bot.ErrorUnauthorized)
	require.Equal(t, 1, attempts)
}

func TestRetryTelegramStartupRetriesRequestTimeout(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := retryTelegramStartupWithPolicy(t.Context(), func(context.Context) error {
		attempts++
		if attempts == 1 {
			return context.DeadlineExceeded
		}
		return nil
	}, 4, 0)

	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}

func TestRetryTelegramStartupHonorsCancellationDuringBackoff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	attempts := 0
	err := retryTelegramStartupWithPolicy(ctx, func(context.Context) error {
		attempts++
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		return errors.New("temporary proxy failure")
	}, 4, time.Hour)

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, attempts)
}
