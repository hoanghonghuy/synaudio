package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
)

func main() {
	log := logging.New("worker")

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database pool create failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	log.Info("worker started", "env", cfg.AppEnv)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := pool.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Error("worker database ping failed", "error", err)
				continue
			}
			log.Info("worker heartbeat ok")
		}
	}
}
