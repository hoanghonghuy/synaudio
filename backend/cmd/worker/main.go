package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
	"github.com/synaudio/synaudio/backend/internal/platform/pgstore"
	"github.com/synaudio/synaudio/backend/internal/platform/providers"
)

func main() {
	log := logging.New("worker")

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "error", err)
		os.Exit(1)
	}

	aiProviders, err := providers.BuildAI(cfg)
	if err != nil {
		log.Error("AI provider init failed", "error", err)
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

	queries := db.New(pool)
	generationStore := pgstore.NewGenerationStore(queries)
	generationService := generation.NewService(generationStore, generation.WithTextAI(aiProviders.TextAI))

	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker-1"
	}

	worker := generation.NewWorker(generationService, workerID, processJob(generationService, log))

	log.Info("worker started", "env", cfg.AppEnv, "worker_id", workerID)

	// Reclaim stale jobs periodically.
	reclaimTicker := time.NewTicker(30 * time.Second)
	defer reclaimTicker.Stop()

	// Poll for new jobs.
	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		case <-reclaimTicker.C:
			reclaimed, err := generationService.ReclaimStaleJobs(ctx, "5 minutes")
			if err != nil {
				log.Error("reclaim stale jobs failed", "error", err)
				continue
			}
			if len(reclaimed) > 0 {
				log.Info("reclaimed stale jobs", "count", len(reclaimed))
			}
		case <-pollTicker.C:
			if err := worker.ProcessOne(ctx); err != nil {
				if err == generation.ErrNoRunnableJob {
					continue
				}
				log.Error("process job failed", "error", err)
			}
		}
	}
}

// processJob dispatches a claimed job to the appropriate durable handler.
func processJob(svc *generation.Service, log *slog.Logger) generation.JobProcessor {
	return func(ctx context.Context, job generation.GenerationJob) error {
		switch job.JobType {
		case "WRITER":
			log.Info("processing writer job", "job_id", job.ID, "run_id", job.RunID)
			revision, err := svc.ExecuteWriterJob(ctx, job)
			if err != nil {
				log.Error("writer job failed", "job_id", job.ID, "run_id", job.RunID, "error", err)
				return err
			}
			log.Info("writer job durable output ready",
				"job_id", job.ID,
				"run_id", job.RunID,
				"content_revision_id", revision.ID,
				"plan_revision_id", revision.PlanRevisionID,
			)
			return nil
		default:
			log.Info("rejecting unknown job type", "job_id", job.ID, "job_type", job.JobType)
			return &generation.ClassifiedError{Class: "PERMANENT", Code: "UNKNOWN_JOB_TYPE"}
		}
	}
}
