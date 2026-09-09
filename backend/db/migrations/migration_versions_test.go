package migrations

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/file"
)

func TestMigrationVersionsAreUnique(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	stemsByVersion := map[uint]map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		stem, ok := migrationStem(entry.Name())
		if !ok {
			continue
		}
		version, ok := migrationVersionPrefix(stem)
		if !ok {
			t.Fatalf("migration filename %q must start with a numeric version prefix", entry.Name())
		}
		if stemsByVersion[version] == nil {
			stemsByVersion[version] = map[string]struct{}{}
		}
		stemsByVersion[version][stem] = struct{}{}
	}

	var duplicates []string
	for version, stems := range stemsByVersion {
		if len(stems) <= 1 {
			continue
		}
		names := make([]string, 0, len(stems))
		for stem := range stems {
			names = append(names, stem)
		}
		sort.Strings(names)
		duplicates = append(duplicates, strings.Join(names, ", "))
		t.Errorf("duplicate migration version %03d: %s", version, strings.Join(names, ", "))
	}
	if len(duplicates) > 0 {
		t.Fatalf("found %d duplicate migration version(s); golang-migrate rejects duplicate version prefixes", len(duplicates))
	}
}

func TestGolangMigrateSourceAcceptsMigrationDirectory(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	driver, err := (&file.File{}).Open("file://" + filepath.ToSlash(dir))
	if err != nil {
		t.Fatalf("golang-migrate source open: %v", err)
	}
	defer driver.Close()

	version, err := driver.First()
	if err != nil {
		t.Fatalf("golang-migrate first version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected first migration version 1, got %d", version)
	}
}

func migrationStem(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, ".up.sql"):
		return strings.TrimSuffix(name, ".up.sql"), true
	case strings.HasSuffix(name, ".down.sql"):
		return strings.TrimSuffix(name, ".down.sql"), true
	default:
		return "", false
	}
}

func migrationVersionPrefix(stem string) (uint, bool) {
	underscore := strings.IndexByte(stem, '_')
	if underscore <= 0 {
		return 0, false
	}
	version, err := strconv.ParseUint(stem[:underscore], 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(version), true
}
