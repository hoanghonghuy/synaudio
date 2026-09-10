package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	platformmetrics "github.com/synaudio/synaudio/backend/internal/platform/metrics"
	"github.com/synaudio/synaudio/backend/internal/platform/workerprobe"
)

type databasePinger interface {
	Ping(context.Context) error
}

func startWorkerProbe(
	ctx context.Context,
	pool databasePinger,
	registry *platformmetrics.Registry,
	acceptingWork func() bool,
	log *slog.Logger,
) {
	addr := os.Getenv("WORKER_PROBE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}

	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase: func(pingCtx context.Context) error {
			return pool.Ping(pingCtx)
		},
		HeartbeatAge: func() time.Duration {
			return registry.HeartbeatAge(time.Now())
		},
		AcceptingWork: acceptingWork,
	})

	server, err := platformmetrics.NewPrivateServer(addr, handler)
	if err != nil {
		log.Error("worker probe config invalid", "error", err)
		return
	}
	if server == nil {
		return
	}

	go func() {
		log.Info("worker probe listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("worker probe server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}
