package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/nightnoryu/go-kita/log"

	"anon3anon/pkg/infrastructure/health"
	"anon3anon/pkg/infrastructure/storage/sqlite"
)

func startHealthServer(ctx context.Context, addr string, store *sqlite.Store, logger log.Logger) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           health.Handler(store, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error(err)
		}
	}()
}
