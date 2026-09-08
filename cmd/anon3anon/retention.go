package main

import (
	"context"
	"time"

	"github.com/nightnoryu/go-kita/log"

	"anon3anon/pkg/infrastructure/storage/sqlite"
)

func startRetentionSweeper(ctx context.Context, conf *config, store *sqlite.Store, logger log.Logger) {
	if conf.RetentionAge <= 0 || conf.RetentionSweepInterval <= 0 {
		return
	}

	sweep := func() {
		if ctx.Err() != nil {
			return
		}

		cutoff := time.Now().UTC().Add(-conf.RetentionAge)
		stats, err := store.PurgeExpired(ctx, cutoff)
		if stats.Sessions > 0 || stats.Relays > 0 || stats.Blocks > 0 || stats.MessageRates > 0 {
			logger.WithFields(log.Fields{
				"sessions_removed":      stats.Sessions,
				"relays_removed":        stats.Relays,
				"blocks_removed":        stats.Blocks,
				"message_rates_removed": stats.MessageRates,
			}).Info("retention sweep")
		}
		if err != nil {
			logger.Error(err)
		}
	}

	go func() {
		sweep()

		ticker := time.NewTicker(conf.RetentionSweepInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep()
			}
		}
	}()
}
