package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/audit"
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/listener"
	"github.com/synaudio/synaudio/backend/internal/planning"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
	"github.com/synaudio/synaudio/backend/internal/platform/httpserver"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
	"github.com/synaudio/synaudio/backend/internal/platform/metrics"
	"github.com/synaudio/synaudio/backend/internal/platform/pgstore"
	"github.com/synaudio/synaudio/backend/internal/platform/providers"
	"github.com/synaudio/synaudio/backend/internal/platform/storage"
	"github.com/synaudio/synaudio/backend/internal/retcon"
	"github.com/synaudio/synaudio/backend/internal/story"
)

func main() {
	log := logging.New("api")

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	emailCfg, err := config.LoadEmail(cfg.AppEnv, cfg.AppPublicURL)
	if err != nil {
		log.Error("email config load failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	accessTokenKeyring, err := config.LoadAccessTokenKeyring(cfg.AppEnv, cfg.AccessTokenSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Error("access-token keyring config failed", logging.ErrAttr(err))
		os.Exit(1)
	}

	aiProviders, err := providers.BuildAI(cfg)
	if err != nil {
		log.Error("AI provider init failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	ttsProvider, err := providers.BuildTTS(cfg)
	if err != nil {
		log.Error("TTS provider init failed", logging.ErrAttr(err))
		os.Exit(1)
	}

	audioProcessorSettings, err := config.LoadAudioProcessorSettings(cfg.AppEnv)
	if err != nil {
		log.Error("audio processor config failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	var audioProcessor audio.AudioProcessor
	var ffmpegProcessor *audio.FFmpegProcessor
	switch audioProcessorSettings.Mode {
	case config.AudioProcessorMock:
		audioProcessor = audio.NewMockAudioProcessor()
	case config.AudioProcessorFFmpeg:
		ffmpegProcessor = audio.NewFFmpegProcessor(audioProcessorSettings.Binary)
		if err := ffmpegProcessor.Validate(); err != nil {
			log.Error("FFmpeg processor unavailable", logging.ErrAttr(err))
			os.Exit(1)
		}
		audioProcessor = ffmpegProcessor
	default:
		log.Error("unsupported audio processor mode", "mode", audioProcessorSettings.Mode)
		os.Exit(1)
	}

	poolSettings, err := config.LoadDatabasePoolSettings(cfg.AppEnv)
	if err != nil {
		log.Error("database pool config failed", logging.ErrAttr(err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, poolSettings)
	if err != nil {
		log.Error("database pool create failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	defer pool.Close()

	database := db.Contextual(pool)
	queries := db.New(database)
	auditBoundary := audit.TransactionBoundary(func(parent context.Context, run func(context.Context) error) error {
		tx, err := pool.Begin(parent)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(parent) }()
		if err := run(db.WithTx(parent, tx)); err != nil {
			return err
		}
		return tx.Commit(parent)
	})

	identityStore := pgstore.NewIdentityStore(queries)
	authService, err := identity.NewRotatingAuthService(identityStore, identity.AuthSettings{
		AccessTokenTTL:        cfg.AccessTokenTTL,
		RefreshSessionTTL:     cfg.RefreshSessionTTL,
		RefreshSessionIdleTTL: cfg.RefreshSessionIdleTTL,
		RecentAuthWindow:      cfg.RecentAuthWindow,
	}, accessTokenKeyring.ActiveKeyID, accessTokenKeyring.Keys, accessTokenKeyring.MaxTTL)
	if err != nil {
		log.Error("access-token manager init failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	var authHandler http.Handler = identity.NewAuthHandler(authService)
	authHandler = identity.WrapSecurityAssurance(authHandler, authService)
	if emailCfg.Mode != config.EmailModeDisabled {
		emailStore := pgstore.NewEmailOutboxStore(database)
		emailService, err := providers.BuildEmail(emailCfg, emailStore)
		if err != nil {
			log.Error("email provider init failed", logging.ErrAttr(err))
			os.Exit(1)
		}
		identityBoundary := identity.TransactionBoundary(func(parent context.Context, run func(context.Context) error) error {
			return db.InTransaction(parent, database, run)
		})
		authHandler = identity.WrapTransactionalEmail(authHandler, authService, emailService, identityBoundary)
	}

	auditStore := pgstore.NewAuditStore(queries)
	auditService := audit.NewService(auditStore)
	auditHandler := audit.NewHandler(auditService)

	storyStore := pgstore.NewStoryStore(queries)
	objStorage, err := storage.NewMinIO(cfg)
	if err != nil {
		log.Error("storage init failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	planningStore := pgstore.NewPlanningStore(queries, database)

	generationStore := pgstore.NewGenerationStore(queries)
	generationService := generation.NewService(generationStore, generation.WithTextAI(aiProviders.TextAI))
	generationHandler := generation.NewLatestRunAwareHandler(generation.NewHandler(generationService, authService.ResolveUserID), generationService)

	audioStore := pgstore.NewAudioStore(queries, database)
	audioService := audio.NewService(audioStore,
		audio.WithTTS(ttsProvider),
		audio.WithObjectStorage(objStorage),
		audio.WithPresigner(objStorage),
		audio.WithAudioProcessor(audioProcessor),
		audio.WithApprovedContentAuthority(generationService),
	)

	storyService := story.NewService(storyStore, story.WithObjectStorage(objStorage))

	planningService := planning.NewService(planningStore,
		planning.WithArchitect(aiProviders.Architect),
		planning.WithMemoryExtractor(aiProviders.MemoryExtractor),
		planning.WithPublishChecker(planning.NewCompositePublishChecker(
			planningStore,
			generationService,
			audioService,
			storyService,
		)),
	)
	planningHandler := planning.NewHandler(planningService)
	planningWorkspaceHandler := planning.NewWorkspaceHandler(planningService)

	storyService = story.NewService(storyStore,
		story.WithObjectStorage(objStorage),
		story.WithActivationChecker(planningService),
	)
	storyHandler := story.NewHandler(storyService)
	storyReadinessHandler := story.NewReadinessHandler(storyService)

	audioService.SetListenerAudioGate(planning.NewListenerEligibility(planningStore, storyService))
	audioHandler := audio.NewHandler(audioService)

	listenerStore := pgstore.NewListenerStore(queries)
	listenerService := listener.NewService(listenerStore)
	listenerHandler := listener.NewHandler(listenerService, authService.ResolveUserID)

	retconStore := pgstore.NewRetconStore(queries)
	retconService := retcon.NewService(retconStore)
	retconHandler := retcon.NewHandler(retconService)

	dependencyChecks := map[string]func() error{
		"database": func() error {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return pool.Ping(pingCtx)
		},
		"storage": func() error {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return objStorage.Ping(pingCtx)
		},
	}
	if ffmpegProcessor != nil {
		dependencyChecks["ffmpeg"] = ffmpegProcessor.Validate
	}

	authAbuseCfg, err := config.LoadAuthAbuse(cfg.AppEnv)
	if err != nil {
		log.Error("auth abuse config failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	if err := config.AuthAbuseProductionRequiresSharedState(cfg.AppEnv, authAbuseCfg.Backend); err != nil {
		log.Error("auth abuse config failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	trustedProxyCIDRs, err := config.LoadTrustedProxyCIDRs()
	if err != nil {
		log.Error("trusted proxy config failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	if err := config.ValidateTrustedProxyCIDRs(trustedProxyCIDRs); err != nil {
		log.Error("trusted proxy config failed", logging.ErrAttr(err))
		os.Exit(1)
	}
	trustedProxy, err := httpapi.ParseTrustedProxyConfig(trustedProxyCIDRs)
	if err != nil {
		log.Error("trusted proxy config failed", logging.ErrAttr(err))
		os.Exit(1)
	}

	var authAbuseLimiter httpapi.AbuseLimiter
	switch authAbuseCfg.Backend {
	case config.AuthAbuseBackendMemory:
		authAbuseLimiter = httpapi.NewMemoryAbuseLimiter()
	case config.AuthAbuseBackendPostgres:
		authAbuseLimiter = httpapi.NewPostgresAbuseLimiter(queries)
	default:
		log.Error("unsupported auth abuse backend", "backend", authAbuseCfg.Backend)
		os.Exit(1)
	}

	metricRegistry := metrics.NewRegistry()
	providers.WireMetrics(metricRegistry)
	adminSecurityHandler := identity.NewAdminSecurityHandler(authService)

	router := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(pingCtx); err != nil {
				return httpapi.ErrDependencyUnavailable
			}
			if ffmpegProcessor != nil {
				if err := ffmpegProcessor.Validate(); err != nil {
					return httpapi.ErrDependencyUnavailable
				}
			}
			return nil
		},
		DependencyChecks:         dependencyChecks,
		Logger:                   log,
		TrustedProxy:             trustedProxy,
		AuthAbuse: &httpapi.AuthAbuse{
			Policies: httpapi.DefaultAuthAbusePolicies(),
			Limiter:  authAbuseLimiter,
			Metrics:  metricRegistry,
			Logger:   log,
		},
		AdminCheck:               authService.ResolveAdmin,
		AdminPermissionCheck:     authService.ResolveAdminPermission,
		AdminRecentAuthCheck:     authService.RequireRecentAuth,
		AuthRecentAuthCheck:      authService.RequireSessionRecentAuth,
		AdminActor:               authService.ResolveUserID,
		AdminSecurityHandler:     adminSecurityHandler,
		AuditRecord:              auditService.RecordReliable,
		AuditBoundary:            auditBoundary,
		AuthHandler:              authHandler,
		AuditHandler:             auditHandler,
		StoryHandler:             storyHandler,
		StoryReadinessHandler:    storyReadinessHandler,
		PlanningHandler:          planningHandler,
		PlanningWorkspaceHandler: planningWorkspaceHandler,
		GenerationHandler:        generationHandler,
		AudioHandler:             audioHandler,
		ListenerHandler:          listenerHandler,
		RetconHandler:            retconHandler,
	})

	metrics.StartDatabasePoolSampler(ctx, pool, metricRegistry, config.DatabasePoolRoleAPI, 15*time.Second)
	metricsServer, err := metrics.NewPrivateServer(os.Getenv("API_METRICS_ADDR"), metricRegistry.Handler())
	if err != nil {
		log.Error("metrics config invalid", logging.ErrAttr(err))
		os.Exit(1)
	}
	if metricsServer != nil {
		go func() {
			log.Info("api metrics listening", "addr", metricsServer.Addr)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("api metrics server failed", logging.ErrAttr(err))
				os.Exit(1)
			}
		}()
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: metricRegistry.HTTPMiddleware(router),
	}
	httpserver.ApplyPublicAPIPolicy(server)

	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.AppEnv, "audio_processor", audioProcessorSettings.Mode)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api server failed", logging.ErrAttr(err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	if metricsServer != nil {
		_ = metricsServer.Shutdown(shutdownCtx)
	}
	log.Info("api stopped")
}
