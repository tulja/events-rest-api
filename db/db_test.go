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

func clearVercelEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"VERCEL",
		"VERCEL_ENV",
		"VERCEL_URL",
		"VERCEL_REGION",
		"VERCEL_DEPLOYMENT_ID",
		"VERCEL_PROJECT_ID",
		"VERCEL_GIT_COMMIT_SHA",
	} {
		t.Setenv(key, "")
	}
}

func TestResolveDBPath_Defaults(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	clearVercelEnv(t)

	// Local cwd is writable in tests → default path.
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
	clearVercelEnv(t)

	want := filepath.Join(os.TempDir(), "events.db")

	cases := []struct {
		key, val string
	}{
		{"VERCEL", "1"},
		{"VERCEL_ENV", "production"},
		{"VERCEL_URL", "example.vercel.app"},
		{"VERCEL_REGION", "iad1"},
		{"VERCEL_DEPLOYMENT_ID", "dpl_abc"},
		{"VERCEL_PROJECT_ID", "prj_abc"},
		{"VERCEL_GIT_COMMIT_SHA", "deadbeef"},
	}
	for _, tc := range cases {
		clearVercelEnv(t)
		t.Setenv(tc.key, tc.val)
		if got := resolveDBPath(); got != want {
			t.Fatalf("%s=%q path: got %q, want %q", tc.key, tc.val, got, want)
		}
	}
}

func TestIsVercel_EnvSignals(t *testing.T) {
	clearVercelEnv(t)
	if isVercel() {
		// May still be true if cwd is under /vercel/ (unlikely in local tests).
		t.Log("isVercel true with cleared env; cwd may look like Vercel")
	}

	t.Setenv("VERCEL_URL", "my-app.vercel.app")
	if !isVercel() {
		t.Fatal("expected isVercel true when VERCEL_URL is set")
	}
}

func TestDirIsWritable(t *testing.T) {
	if !dirIsWritable(t.TempDir()) {
		t.Fatal("expected temp dir to be writable")
	}
	if dirIsWritable(filepath.Join(t.TempDir(), "does-not-exist")) {
		t.Fatal("expected missing dir to be not writable")
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
