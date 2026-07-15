package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTables_Idempotent(t *testing.T) {
	setupTestDB(t)

	// Second call should not panic (CREATE TABLE IF NOT EXISTS).
	createTables()

	// Schema is usable after double create.
	user := seedUser(t, "schema@example.com", "pass")
	if user.ID <= 0 {
		t.Fatalf("expected usable schema, got user ID %d", user.ID)
	}
}

func TestResolveDBPath_Defaults(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("VERCEL", "")
	t.Setenv("VERCEL_ENV", "")

	if got := resolveDBPath(); got != "./events.db" {
		t.Fatalf("resolveDBPath() = %q, want ./events.db", got)
	}
}

func TestResolveDBPath_EnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_PATH", "/custom/data.db")
	t.Setenv("SQLITE_PATH", "/ignored.db")
	t.Setenv("VERCEL", "1")

	if got := resolveDBPath(); got != "/custom/data.db" {
		t.Fatalf("DATABASE_PATH override: got %q", got)
	}

	t.Setenv("DATABASE_PATH", "")
	t.Setenv("SQLITE_PATH", "/from-sqlite-path.db")
	if got := resolveDBPath(); got != "/from-sqlite-path.db" {
		t.Fatalf("SQLITE_PATH override: got %q", got)
	}
}

func TestResolveDBPath_VercelUsesTemp(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("VERCEL", "1")
	t.Setenv("VERCEL_ENV", "")

	want := filepath.Join(os.TempDir(), "events.db")
	if got := resolveDBPath(); got != want {
		t.Fatalf("VERCEL path: got %q, want %q", got, want)
	}

	t.Setenv("VERCEL", "")
	t.Setenv("VERCEL_ENV", "production")
	if got := resolveDBPath(); got != want {
		t.Fatalf("VERCEL_ENV path: got %q, want %q", got, want)
	}
}

func TestSqliteDSN(t *testing.T) {
	if got := sqliteDSN(":memory:"); got != ":memory:" {
		t.Fatalf("memory dsn: got %q", got)
	}

	got := sqliteDSN("/tmp/events.db")
	if !strings.HasPrefix(got, "file:/tmp/events.db?") {
		t.Fatalf("dsn prefix: got %q", got)
	}
	if !strings.Contains(got, "mode=rwc") {
		t.Fatalf("expected mode=rwc in %q", got)
	}
	if !strings.Contains(got, "_pragma=foreign_keys(1)") {
		t.Fatalf("expected foreign_keys pragma in %q", got)
	}
}

func TestEnsureDBDir(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir", "events.db")
	if err := ensureDBDir(path); err != nil {
		t.Fatalf("ensureDBDir: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("parent dir missing: %v", err)
	}
}

func TestInitDB_WritableTempPath(t *testing.T) {
	// Avoid clobbering a process-wide package handle if other tests run in parallel packages;
	// within this package tests are sequential by default for package state.
	dir := t.TempDir()
	path := filepath.Join(dir, "init-test.db")
	t.Setenv("DATABASE_PATH", path)
	t.Setenv("VERCEL", "")
	t.Setenv("VERCEL_ENV", "")

	// Do not leave global db pointing at this file for later tests that use setupTestDB —
	// setupTestDB overwrites db, so this is safe as long as we finish first.
	InitDB()
	t.Cleanup(func() {
		if db != nil {
			_ = db.Close()
			db = nil
		}
	})

	if err := Ping(); err != nil {
		t.Fatalf("Ping after InitDB: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected db file at %s: %v", path, err)
	}
}
