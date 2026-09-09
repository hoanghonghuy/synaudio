package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestAdminPermissionVocabularyRollbackPreservesPreexistingAuthorization(t *testing.T) {
	up, err := os.ReadFile("000017_admin_permission_vocabulary.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("000017_admin_permission_vocabulary.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	upSQL := string(up)
	downSQL := string(down)

	for _, required := range []string{
		"migration_000017_admin_permission_ownership",
		"permission_created",
		"admin_binding_created",
		"ON CONFLICT (code) DO NOTHING",
		"ON CONFLICT DO NOTHING",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("up migration must record ownership for conflict-safe seed rows; missing %q", required)
		}
	}

	for _, required := range []string{
		"o.admin_binding_created = TRUE",
		"o.permission_created = TRUE",
		"DROP TABLE migration_000017_admin_permission_ownership",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("down migration must delete only migration-owned rows; missing %q", required)
		}
	}

	if strings.Contains(downSQL, "p.code IN") {
		t.Fatal("down migration must not delete permissions by vocabulary code because those rows may pre-exist")
	}
}
