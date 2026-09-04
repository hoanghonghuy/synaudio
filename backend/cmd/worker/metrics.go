package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	platformmetrics "github.com/synaudio/synaudio/backend/internal/platform/metrics"
)

func startWorkerMetrics(ctx context.Context, registry *platformmetrics.Registry, log *slog.Logger) {
	addr := os.Getenv("WORKER_METRICS_ADDR")
	if addr == "" {
		addr = ":9091"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           registry.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("worker metrics listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("worker metrics server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}
