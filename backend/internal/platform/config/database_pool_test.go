package config_test

import (
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func setProductionPoolBudgetEnv(t *testing.T) {
	t.Setenv("DATABASE_CONNECTION_BUDGET", "100")
	t.Setenv("DATABASE_POOL_API_REPLICAS", "2")
	t.Setenv("DATABASE_POOL_WORKER_REPLICAS", "1")
	t.Setenv("DATABASE_POOL_API_MAX_CONNS", "10")
	t.Setenv("DATABASE_POOL_WORKER_MAX_CONNS", "5")
	t.Setenv("DATABASE_POOL_OPERATIONS_RESERVE", "5")
	t.Setenv("DATABASE_POOL_ROLLOUT_SURGE_FACTOR", "2")
}

func TestLoadDatabasePoolSettingsRequiresExplicitProductionCapacity(t *testing.T) {
	setProductionPoolBudgetEnv(t)
	t.Setenv("DATABASE_POOL_PROCESS_ROLE", "api")

	_, err := config.LoadDatabasePoolSettings(config.EnvProduction)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_POOL_MAX_CONNS") {
		t.Fatalf("expected production max conns requirement, got %v", err)
	}
}

func TestLoadDatabasePoolSettingsRejectsBudgetOverflow(t *testing.T) {
	setProductionPoolBudgetEnv(t)
	t.Setenv("DATABASE_POOL_PROCESS_ROLE", "api")
	t.Setenv("DATABASE_POOL_MAX_CONNS", "10")
	t.Setenv("DATABASE_CONNECTION_BUDGET", "20")

	_, err := config.LoadDatabasePoolSettings(config.EnvProduction)
	if err == nil || !strings.Contains(err.Error(), "database pool budget exceeded") {
		t.Fatalf("expected budget overflow rejection, got %v", err)
	}
}

func TestLoadDatabasePoolSettingsRequiresRoleMatchForAPI(t *testing.T) {
	setProductionPoolBudgetEnv(t)
	t.Setenv("DATABASE_POOL_PROCESS_ROLE", "api")
	t.Setenv("DATABASE_POOL_MAX_CONNS", "8")

	_, err := config.LoadDatabasePoolSettings(config.EnvProduction)
	if err == nil || !strings.Contains(err.Error(), "must match DATABASE_POOL_API_MAX_CONNS") {
		t.Fatalf("expected api max conns mismatch rejection, got %v", err)
	}
}

func TestLoadDatabasePoolSettingsAcceptsValidProductionBudget(t *testing.T) {
	setProductionPoolBudgetEnv(t)
	t.Setenv("DATABASE_POOL_PROCESS_ROLE", "worker")
	t.Setenv("DATABASE_POOL_MAX_CONNS", "5")

	settings, err := config.LoadDatabasePoolSettings(config.EnvProduction)
	if err != nil {
		t.Fatalf("expected valid production pool settings, got %v", err)
	}
	if settings.Role != config.DatabasePoolRoleWorker {
		t.Fatalf("expected worker role, got %q", settings.Role)
	}
	if settings.EffectiveMaxConns() != 5 {
		t.Fatalf("expected max conns 5, got %d", settings.EffectiveMaxConns())
	}
}

func TestLoadDatabasePoolSettingsUsesExplicitDevelopmentDefault(t *testing.T) {
	settings, err := config.LoadDatabasePoolSettings(config.EnvDevelopment)
	if err != nil {
		t.Fatalf("expected development pool settings, got %v", err)
	}
	if settings.EffectiveMaxConns() < 4 {
		t.Fatalf("expected development default max conns >= 4, got %d", settings.EffectiveMaxConns())
	}
}

func TestLoadDatabasePoolSettingsRejectsInvalidMinConns(t *testing.T) {
	t.Setenv("DATABASE_POOL_MAX_CONNS", "4")
	t.Setenv("DATABASE_POOL_MIN_CONNS", "8")

	_, err := config.LoadDatabasePoolSettings(config.EnvDevelopment)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_POOL_MIN_CONNS") {
		t.Fatalf("expected min conns validation failure, got %v", err)
	}
}
