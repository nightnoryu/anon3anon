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

func startHealthServer(ctx context.Context, addr string, store *sqlite.Store, logger log.Logger, metrics *appMetrics) {
	healthHandler, err := health.Handler(store, logger)
	if err != nil {
		logger.Error(err, "create health handlers")
		return
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           newServerHandler(healthHandler, metrics),
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

func newServerHandler(healthHandler http.Handler, metrics *appMetrics) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.handler())
	mux.Handle("/", healthHandler)
	return mux
}
