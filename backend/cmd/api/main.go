package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/listener"
	"github.com/synaudio/synaudio/backend/internal/planning"
	"github.com/synaudio/synaudio/backend/internal/retcon"
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
	planningStore := pgstore.NewPlanningStore(queries)
	planningService := planning.NewService(planningStore,
		planning.WithArchitect(planning.NewMockArchitect()),
		planning.WithMemoryExtractor(planning.NewMockMemoryExtractor()),
	)
	planningHandler := planning.NewHandler(planningService)

	storyService := story.NewService(storyStore,
		story.WithObjectStorage(objStorage),
		story.WithActivationChecker(planningService),
	)
	storyHandler := story.NewHandler(storyService)

	generationStore := pgstore.NewGenerationStore(queries)
	generationService := generation.NewService(generationStore, generation.WithTextAI(generation.NewMockTextAI()))
	generationHandler := generation.NewHandler(generationService)

	audioStore := pgstore.NewAudioStore(queries)
	audioService := audio.NewService(audioStore,
		audio.WithTTS(audio.NewMockTTS()),
		audio.WithPresigner(objStorage),
	)
	audioHandler := audio.NewHandler(audioService)

	listenerStore := pgstore.NewListenerStore(queries)
	listenerService := listener.NewService(listenerStore)
	listenerHandler := listener.NewHandler(listenerService)

	retconStore := pgstore.NewRetconStore(queries)
	retconService := retcon.NewService(retconStore)
	retconHandler := retcon.NewHandler(retconService)

	router := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(pingCtx); err != nil {
				return httpapi.ErrDependencyUnavailable
			}
			return nil
		},
		AuthHandler:      authHandler,
		StoryHandler:     storyHandler,
		PlanningHandler:  planningHandler,
		GenerationHandler: generationHandler,
		AudioHandler:     audioHandler,
		ListenerHandler:  listenerHandler,
		RetconHandler:    retconHandler,
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
