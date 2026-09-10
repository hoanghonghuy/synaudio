package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DatabasePoolRoleAPI    = "api"
	DatabasePoolRoleWorker = "worker"

	defaultDatabasePoolMaxConnsDev         = 0 // resolved to pgx default at apply time
	defaultDatabasePoolOperationsReserve   = 5
	defaultDatabasePoolRolloutSurgeFactor  = 2
	defaultDatabasePoolMaxConnLifetime     = time.Hour
	defaultDatabasePoolMaxConnIdleTime     = 30 * time.Minute
	defaultDatabasePoolHealthCheckPeriod   = time.Minute
	defaultDatabasePoolAcquireTimeout      = 30 * time.Second
	maxDatabasePoolMaxConns                = 256
)

// DatabasePoolSettings is the explicit per-process PostgreSQL pool contract shared
// by API and worker binaries.
type DatabasePoolSettings struct {
	Role               string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	MaxConnIdleTime    time.Duration
	HealthCheckPeriod  time.Duration
	AcquireTimeout     time.Duration
	ConnectionBudget   int32
	OperationsReserve  int32
	APIReplicas        int32
	WorkerReplicas     int32
	APIMaxConns        int32
	WorkerMaxConns     int32
	RolloutSurgeFactor int32
}

// LoadDatabasePoolSettings loads the shared pool contract for the current process.
// Production requires explicit capacity and budget inputs so total database demand
// cannot grow implicitly with replica count or CPU defaults.
func LoadDatabasePoolSettings(appEnv string) (DatabasePoolSettings, error) {
	maxConns, err := getenvInt32("DATABASE_POOL_MAX_CONNS", defaultDatabasePoolMaxConnsDev)
	if err != nil {
		return DatabasePoolSettings{}, fmt.Errorf("DATABASE_POOL_MAX_CONNS: %w", err)
	}
	minConns, err := getenvInt32("DATABASE_POOL_MIN_CONNS", 0)
	if err != nil {
		return DatabasePoolSettings{}, fmt.Errorf("DATABASE_POOL_MIN_CONNS: %w", err)
	}
	maxConnLifetime, err := getenvDuration("DATABASE_POOL_MAX_CONN_LIFETIME", defaultDatabasePoolMaxConnLifetime)
	if err != nil {
		return DatabasePoolSettings{}, err
	}
	maxConnIdleTime, err := getenvDuration("DATABASE_POOL_MAX_CONN_IDLE_TIME", defaultDatabasePoolMaxConnIdleTime)
	if err != nil {
		return DatabasePoolSettings{}, err
	}
	healthCheckPeriod, err := getenvDuration("DATABASE_POOL_HEALTH_CHECK_PERIOD", defaultDatabasePoolHealthCheckPeriod)
	if err != nil {
		return DatabasePoolSettings{}, err
	}
	acquireTimeout, err := getenvDuration("DATABASE_POOL_ACQUIRE_TIMEOUT", defaultDatabasePoolAcquireTimeout)
	if err != nil {
		return DatabasePoolSettings{}, err
	}

	settings := DatabasePoolSettings{
		Role:              strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_POOL_PROCESS_ROLE"))),
		MaxConns:          maxConns,
		MinConns:          minConns,
		MaxConnLifetime:   maxConnLifetime,
		MaxConnIdleTime:   maxConnIdleTime,
		HealthCheckPeriod: healthCheckPeriod,
		AcquireTimeout:    acquireTimeout,
	}

	if appEnv == EnvProduction {
		connectionBudget, err := getenvRequiredInt32("DATABASE_CONNECTION_BUDGET")
		if err != nil {
			return DatabasePoolSettings{}, err
		}
		apiReplicas, err := getenvRequiredInt32("DATABASE_POOL_API_REPLICAS")
		if err != nil {
			return DatabasePoolSettings{}, err
		}
		workerReplicas, err := getenvRequiredInt32("DATABASE_POOL_WORKER_REPLICAS")
		if err != nil {
			return DatabasePoolSettings{}, err
		}
		apiMaxConns, err := getenvRequiredInt32("DATABASE_POOL_API_MAX_CONNS")
		if err != nil {
			return DatabasePoolSettings{}, err
		}
		workerMaxConns, err := getenvRequiredInt32("DATABASE_POOL_WORKER_MAX_CONNS")
		if err != nil {
			return DatabasePoolSettings{}, err
		}
		operationsReserve, err := getenvInt32("DATABASE_POOL_OPERATIONS_RESERVE", defaultDatabasePoolOperationsReserve)
		if err != nil {
			return DatabasePoolSettings{}, fmt.Errorf("DATABASE_POOL_OPERATIONS_RESERVE: %w", err)
		}
		rolloutSurgeFactor, err := getenvInt32("DATABASE_POOL_ROLLOUT_SURGE_FACTOR", defaultDatabasePoolRolloutSurgeFactor)
		if err != nil {
			return DatabasePoolSettings{}, fmt.Errorf("DATABASE_POOL_ROLLOUT_SURGE_FACTOR: %w", err)
		}

		settings.ConnectionBudget = connectionBudget
		settings.OperationsReserve = operationsReserve
		settings.APIReplicas = apiReplicas
		settings.WorkerReplicas = workerReplicas
		settings.APIMaxConns = apiMaxConns
		settings.WorkerMaxConns = workerMaxConns
		settings.RolloutSurgeFactor = rolloutSurgeFactor
	}

	if err := settings.validate(appEnv); err != nil {
		return DatabasePoolSettings{}, err
	}
	return settings, nil
}

func (s DatabasePoolSettings) validate(appEnv string) error {
	if appEnv == EnvProduction {
		if s.MaxConns <= 0 {
			return fmt.Errorf("DATABASE_POOL_MAX_CONNS must be positive in production")
		}
		if s.Role != DatabasePoolRoleAPI && s.Role != DatabasePoolRoleWorker {
			return fmt.Errorf("DATABASE_POOL_PROCESS_ROLE must be %q or %q in production", DatabasePoolRoleAPI, DatabasePoolRoleWorker)
		}
		switch s.Role {
		case DatabasePoolRoleAPI:
			if s.MaxConns != s.APIMaxConns {
				return fmt.Errorf("DATABASE_POOL_MAX_CONNS (%d) must match DATABASE_POOL_API_MAX_CONNS (%d) for api role", s.MaxConns, s.APIMaxConns)
			}
		case DatabasePoolRoleWorker:
			if s.MaxConns != s.WorkerMaxConns {
				return fmt.Errorf("DATABASE_POOL_MAX_CONNS (%d) must match DATABASE_POOL_WORKER_MAX_CONNS (%d) for worker role", s.MaxConns, s.WorkerMaxConns)
			}
		}
	} else if s.MaxConns < 0 {
		return fmt.Errorf("DATABASE_POOL_MAX_CONNS must be zero or positive")
	}

	if s.MaxConns > maxDatabasePoolMaxConns {
		return fmt.Errorf("DATABASE_POOL_MAX_CONNS must be <= %d", maxDatabasePoolMaxConns)
	}
	if s.MinConns < 0 {
		return fmt.Errorf("DATABASE_POOL_MIN_CONNS must be zero or positive")
	}
	if s.MaxConns > 0 && s.MinConns > s.MaxConns {
		return fmt.Errorf("DATABASE_POOL_MIN_CONNS must be <= DATABASE_POOL_MAX_CONNS")
	}
	if s.MaxConnLifetime <= 0 {
		return fmt.Errorf("DATABASE_POOL_MAX_CONN_LIFETIME must be positive")
	}
	if s.MaxConnIdleTime <= 0 {
		return fmt.Errorf("DATABASE_POOL_MAX_CONN_IDLE_TIME must be positive")
	}
	if s.HealthCheckPeriod <= 0 {
		return fmt.Errorf("DATABASE_POOL_HEALTH_CHECK_PERIOD must be positive")
	}
	if s.AcquireTimeout <= 0 {
		return fmt.Errorf("DATABASE_POOL_ACQUIRE_TIMEOUT must be positive")
	}

	if appEnv != EnvProduction {
		return nil
	}

	if s.ConnectionBudget <= 0 {
		return fmt.Errorf("DATABASE_CONNECTION_BUDGET must be positive")
	}
	if s.OperationsReserve < 0 {
		return fmt.Errorf("DATABASE_POOL_OPERATIONS_RESERVE must be zero or positive")
	}
	if s.APIReplicas <= 0 {
		return fmt.Errorf("DATABASE_POOL_API_REPLICAS must be positive")
	}
	if s.WorkerReplicas <= 0 {
		return fmt.Errorf("DATABASE_POOL_WORKER_REPLICAS must be positive")
	}
	if s.APIMaxConns <= 0 || s.WorkerMaxConns <= 0 {
		return fmt.Errorf("DATABASE_POOL_API_MAX_CONNS and DATABASE_POOL_WORKER_MAX_CONNS must be positive")
	}
	if s.RolloutSurgeFactor < 1 {
		return fmt.Errorf("DATABASE_POOL_ROLLOUT_SURGE_FACTOR must be >= 1")
	}

	peak := int64(s.RolloutSurgeFactor) * (int64(s.APIReplicas)*int64(s.APIMaxConns) + int64(s.WorkerReplicas)*int64(s.WorkerMaxConns))
	peak += int64(s.OperationsReserve)
	if peak > int64(s.ConnectionBudget) {
		return fmt.Errorf(
			"database pool budget exceeded: rollout peak %d connections (%d surge * (%d api * %d + %d worker * %d) + %d reserve) exceeds DATABASE_CONNECTION_BUDGET %d",
			peak,
			s.RolloutSurgeFactor,
			s.APIReplicas,
			s.APIMaxConns,
			s.WorkerReplicas,
			s.WorkerMaxConns,
			s.OperationsReserve,
			s.ConnectionBudget,
		)
	}
	return nil
}

// EffectiveMaxConns returns the configured max connections, or the explicit pgx
// default when development leaves DATABASE_POOL_MAX_CONNS unset.
func (s DatabasePoolSettings) EffectiveMaxConns() int32 {
	if s.MaxConns > 0 {
		return s.MaxConns
	}
	defaultMax := int32(4)
	if cpus := runtime.NumCPU(); cpus > int(defaultMax) {
		return int32(cpus)
	}
	return defaultMax
}

func getenvInt32(key string, fallback int32) (int32, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("must be a valid integer: %w", err)
	}
	return int32(value), nil
}

func getenvRequiredInt32(key string) (int32, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required in production", key)
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}
	return int32(value), nil
}
