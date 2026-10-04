package migration

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var migrationFilename = regexp.MustCompile(`^(\d{6})_[a-z0-9_]+\.(up|down)\.sql$`)

func TestMigrationFilesAreConsistentlyNamedAndPaired(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate migration test file")
	}

	entries, err := os.ReadDir(filepath.Join(filepath.Dir(filename), "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("read migrations directory: %v", err)
	}

	pairs := make(map[string]map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		matches := migrationFilename.FindStringSubmatch(entry.Name())
		if matches == nil {
			t.Errorf("migration file %q does not follow <version>_<description>.<direction>.sql", entry.Name())
			continue
		}

		migrationName := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".sql"), "."+matches[2])
		if pairs[migrationName] == nil {
			pairs[migrationName] = make(map[string]bool)
		}
		pairs[migrationName][matches[2]] = true
	}

	if len(pairs) == 0 {
		t.Fatal("no migration files found")
	}

	for name, directions := range pairs {
		if !directions["up"] || !directions["down"] {
			t.Errorf("migration %q must have both up and down files", name)
		}
	}
}
