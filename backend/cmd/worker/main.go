package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/audit"
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/notification"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
	platformmetrics "github.com/synaudio/synaudio/backend/internal/platform/metrics"
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
	emailCfg, err := config.LoadEmail(cfg.AppEnv, cfg.AppPublicURL)
	if err != nil {
		log.Error("email config load failed", "error", err)
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
	auditService := audit.NewService(pgstore.NewAuditStore(queries))
	identityService := identity.NewAuthService(pgstore.NewIdentityStore(queries))

	var emailService *notification.Service
	if emailCfg.Mode != config.EmailModeDisabled {
		emailService, err = providers.BuildEmail(emailCfg, pgstore.NewEmailOutboxStore(pool))
		if err != nil {
			log.Error("email provider init failed", "error", err)
			os.Exit(1)
		}
	}

	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker-1"
	}

	metricRegistry := platformmetrics.NewRegistry()
	metricRegistry.WorkerHeartbeat(time.Now())
	startWorkerMetrics(ctx, metricRegistry, log)

	jobAudit := func(ctx context.Context, event generation.JobAuditEvent) error {
		actorType := audit.ActorSystem
		if event.Job.JobType == "WRITER" {
			actorType = audit.ActorAI
		}
		result := audit.ResultSucceeded
		metricOutcome := "success"
		if event.Outcome == "FAILED" {
			result = audit.ResultFailed
			metricOutcome = "failure"
		}
		metricRegistry.ObserveGenerationJob(event.Job.JobType, metricOutcome, event.ErrorClass)
		metricRegistry.ObserveGenerationDuration(event.Job.JobType, metricOutcome, event.ErrorClass, event.Duration)
		_, err := auditService.RecordReliable(ctx, audit.Event{
			ActorType:       actorType,
			Action:          "GENERATION_JOB_" + event.Outcome,
			ResourceType:    "GENERATION_JOB",
			ResourceID:      event.Job.ID,
			Result:          result,
			GenerationRunID: event.Job.RunID,
			Provenance: map[string]any{
				"attempt_id": event.AttemptID,
				"job_type":   event.Job.JobType,
			},
			Metadata: map[string]any{
				"worker_id":   workerID,
				"error_class": event.ErrorClass,
				"error_code":  event.ErrorCode,
			},
		})
		return err
	}

	deletionAudit := func(ctx context.Context, event identity.AccountDeletionPurgeEvent) error {
		result := audit.ResultSucceeded
		if event.Outcome == "FAILED" {
			result = audit.ResultFailed
		}
		_, err := auditService.RecordReliable(ctx, audit.Event{
			ActorType:    audit.ActorSystem,
			Action:       "ACCOUNT_DELETION_PURGE_" + event.Outcome,
			ResourceType: "USER",
			ResourceID:   event.UserID,
			Result:       result,
			Metadata: map[string]any{
				"worker_id": workerID,
			},
		})
		return err
	}

	worker := generation.NewWorker(
		generationService,
		workerID,
		processJob(generationService, log),
		generation.WithJobAudit(jobAudit),
	)

	log.Info("worker started", "env", cfg.AppEnv, "worker_id", workerID)

	reclaimTicker := time.NewTicker(30 * time.Second)
	defer reclaimTicker.Stop()

	auditTicker := time.NewTicker(15 * time.Second)
	defer auditTicker.Stop()

	emailTicker := time.NewTicker(5 * time.Second)
	defer emailTicker.Stop()

	deletionTicker := time.NewTicker(time.Hour)
	defer deletionTicker.Stop()

	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		case <-reclaimTicker.C:
			metricRegistry.WorkerHeartbeat(time.Now())
			reclaimed, err := generationService.ReclaimStaleJobs(ctx, "5 minutes")
			metricRegistry.ObserveWorkerLoop("stale_reclaim", err)
			if err != nil {
				log.Error("reclaim stale jobs failed", "error", err)
				continue
			}
			metricRegistry.AddWorkerItems("stale_reclaim", "reclaimed", len(reclaimed))
			if len(reclaimed) > 0 {
				log.Info("reclaimed stale jobs", "count", len(reclaimed))
			}
		case <-auditTicker.C:
			metricRegistry.WorkerHeartbeat(time.Now())
			report, err := auditService.DeliverPending(ctx, 50)
			metricRegistry.ObserveWorkerLoop("audit_delivery", err)
			if err != nil {
				log.Error("audit outbox reconciliation failed", "error", err)
				continue
			}
			metricRegistry.AddWorkerItems("audit_delivery", "claimed", report.Claimed)
			metricRegistry.AddWorkerItems("audit_delivery", "delivered", report.Delivered)
			metricRegistry.AddWorkerItems("audit_delivery", "retrying", report.Retrying)
			metricRegistry.AddWorkerItems("audit_delivery", "dead_letter", report.DeadLetter)
			if report.DeadLetter > 0 {
				log.Error("audit delivery dead-lettered", "count", report.DeadLetter, "claimed", report.Claimed)
			} else if report.Claimed > 0 {
				log.Info("audit outbox reconciled", "claimed", report.Claimed, "delivered", report.Delivered, "retrying", report.Retrying)
			}
		case <-emailTicker.C:
			metricRegistry.WorkerHeartbeat(time.Now())
			if emailService == nil {
				continue
			}
			processed := 0
			var deliveryErr error
			for i := 0; i < 20; i++ {
				didWork, err := emailService.DeliverNext(ctx)
				if err != nil {
					deliveryErr = err
					log.Error("transactional email delivery failed", "error", err)
					break
				}
				if !didWork {
					break
				}
				processed++
			}
			metricRegistry.ObserveWorkerLoop("email_delivery", deliveryErr)
			metricRegistry.AddWorkerItems("email_delivery", "processed", processed)
		case <-deletionTicker.C:
			metricRegistry.WorkerHeartbeat(time.Now())
			purged, err := identityService.PurgeEligibleAccountsObserved(ctx, 50, deletionAudit)
			metricRegistry.ObserveWorkerLoop("account_deletion", err)
			if err != nil {
				log.Error("account deletion reconciliation failed", "error", err)
				continue
			}
			metricRegistry.AddWorkerItems("account_deletion", "purged", purged)
			if purged > 0 {
				log.Info("eligible accounts purged", "count", purged)
			}
		case <-pollTicker.C:
			metricRegistry.WorkerHeartbeat(time.Now())
			err := worker.ProcessOne(ctx)
			if err != nil {
				if err == generation.ErrNoRunnableJob {
					metricRegistry.ObserveWorkerLoop("generation_poll", nil)
					continue
				}
				metricRegistry.ObserveWorkerLoop("generation_poll", err)
				log.Error("process job failed", "error", err)
				continue
			}
			metricRegistry.ObserveWorkerLoop("generation_poll", nil)
			metricRegistry.AddWorkerItems("generation_poll", "processed", 1)
		}
	}
}

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
