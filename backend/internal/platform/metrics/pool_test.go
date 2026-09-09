package metrics

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRegistryDatabasePoolMetricsUseBoundedRoleLabels(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://invalid:invalid@127.0.0.1:1/invalid?connect_timeout=1")
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	cfg.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	r := NewRegistry()
	r.SetDatabasePool("api", pool.Stat())
	r.SetDatabasePool("user-controlled-role", pool.Stat())

	res := httptest.NewRecorder()
	r.Handler().ServeHTTP(res, httptest.NewRequest("GET", "/metrics", nil))
	body := res.Body.String()

	for _, want := range []string{
		`synaudio_database_pool_max_conns{role="api"} 3`,
		`synaudio_database_pool_max_conns{role="other"} 3`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "user-controlled-role") {
		t.Fatalf("metrics output leaked unbounded role label: %s", body)
	}
}
