package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/planning"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
	"github.com/synaudio/synaudio/backend/internal/platform/pgstore"
	"github.com/synaudio/synaudio/backend/internal/platform/storage"
	"github.com/synaudio/synaudio/backend/internal/story"
)

func main() {
	log := logging.New("api")

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

	queries := db.New(pool)

	identityStore := pgstore.NewIdentityStore(queries)
	authService := identity.NewAuthService(identityStore)
	authHandler := identity.NewAuthHandler(authService)

	storyStore := pgstore.NewStoryStore(queries)
	objStorage, err := storage.NewMinIO(cfg)
	if err != nil {
		log.Error("storage init failed", "error", err)
		os.Exit(1)
	}
	storyService := story.NewService(storyStore, story.WithObjectStorage(objStorage))
	storyHandler := story.NewHandler(storyService)

	planningStore := pgstore.NewPlanningStore(queries)
	planningService := planning.NewService(planningStore, planning.WithArchitect(planning.NewMockArchitect()))
	planningHandler := planning.NewHandler(planningService)

	router := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(pingCtx); err != nil {
				return httpapi.ErrDependencyUnavailable
			}
			return nil
		},
		AuthHandler:     authHandler,
		StoryHandler:    storyHandler,
		PlanningHandler: planningHandler,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Info("api stopped")
}
