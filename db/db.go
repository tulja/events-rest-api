package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func InitDB() {
	path := resolveDBPath()
	if err := ensureDBDir(path); err != nil {
		slog.Error("failed to prepare database directory", "path", path, "err", err)
		panic(err)
	}

	var err error
	// modernc driver name is "sqlite". Use URI so create/write flags are explicit.
	dsn := sqliteDSN(path)
	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		slog.Error("failed to open database", "path", path, "err", err)
		panic(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// sql.Open is lazy; Ping forces a real open so we fail fast with a clear path.
	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "path", path, "dsn", dsn, "err", err)
		panic(err)
	}

	slog.Info("opened sqlite database", "path", path)
	enableForeignKeys()
	createTables()
}

// resolveDBPath picks a writable SQLite file path.
// - DATABASE_PATH / SQLITE_PATH if set
// - $TMPDIR/events.db on Vercel / other read-only app filesystems (only /tmp is writable)
// - ./events.db otherwise (local dev)
//
// Note: On Vercel, /tmp is ephemeral per instance. Data is not durable across cold starts
// or multiple instances; use a managed DB for production persistence.
func resolveDBPath() string {
	if p := os.Getenv("DATABASE_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("SQLITE_PATH"); p != "" {
		return p
	}
	// Prefer temp when the platform or cwd cannot host a writable SQLite file.
	// Long-running Vercel servers do not always set VERCEL=1 the same way as serverless.
	if reason, ok := mustUseTempDB(); ok {
		path := filepath.Join(os.TempDir(), "events.db")
		slog.Info("using temp sqlite path", "path", path, "reason", reason)
		return path
	}
	return "./events.db"
}

// mustUseTempDB reports whether SQLite should live under os.TempDir().
// reason is a short diagnostic string when ok is true.
func mustUseTempDB() (reason string, ok bool) {
	if isVercel() {
		return "vercel", true
	}
	// Defense in depth: deployment FS is often read-only even when platform env is missing.
	if !dirIsWritable(".") {
		return "cwd-not-writable", true
	}
	return "", false
}

func isVercel() bool {
	// System env vars Vercel may inject (serverless, builds, and long-running servers).
	// Do not rely on a single key — product surfaces differ.
	for _, key := range []string{
		"VERCEL",
		"VERCEL_ENV",
		"VERCEL_URL",
		"VERCEL_REGION",
		"VERCEL_DEPLOYMENT_ID",
		"VERCEL_PROJECT_ID",
		"VERCEL_GIT_COMMIT_SHA",
	} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	// Build/run paths on Vercel look like /vercel/path0/...
	if cwd, err := os.Getwd(); err == nil {
		if strings.HasPrefix(cwd, "/vercel/") || strings.Contains(cwd, "/vercel/path") {
			return true
		}
	}
	return false
}

// dirIsWritable returns true if a new file can be created in dir.
func dirIsWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".sqlite-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// ensureDBDir creates the parent directory of path when needed (no-op for :memory: / bare files).
func ensureDBDir(path string) error {
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func sqliteDSN(path string) string {
	if path == ":memory:" {
		return path
	}
	// Ensure absolute path for file URI when possible.
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, "file:") {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	// mode=rwc creates the file if missing (needed for first boot).
	// _pragma=foreign_keys(1) applies on every new connection (pool-safe).
	return fmt.Sprintf("file:%s?mode=rwc&_pragma=foreign_keys(1)", filepath.ToSlash(path))
}

// InitInMemory opens an in-memory SQLite DB and creates schema.
// Intended for tests in other packages (e.g. routes) that cannot access the unexported handle.
func InitInMemory() error {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	// :memory: is per-connection; keep a single connection for schema visibility.
	testDB.SetMaxOpenConns(1)
	testDB.SetMaxIdleConns(1)
	db = testDB
	if err := db.Ping(); err != nil {
		return err
	}
	enableForeignKeys()
	createTables()
	return nil
}

// Ping checks that the database connection is usable.
func Ping() error {
	if db == nil {
		return sql.ErrConnDone
	}
	return db.Ping()
}

func enableForeignKeys() {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		slog.Error("failed to enable foreign keys", "err", err)
		panic(err)
	}
	slog.Debug("sqlite foreign keys enabled")
}

func createTables() {
	createEventsTableQuery := `CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        description TEXT NOT NULL,
        location TEXT NOT NULL,
        date_time DATETIME NOT NULL,
        userid INTEGER 
    )`
	_, err := db.Exec(createEventsTableQuery)
	if err != nil {
		slog.Error("failed to create table", "table", "events", "err", err)
		panic(err)
	}

	createUsersTableQuery := `CREATE TABLE IF NOT EXISTS users (
		 id INTEGER PRIMARY KEY AUTOINCREMENT, 
		 email TEXT NOT NULL UNIQUE, 
		 password TEXT NOT NULL
	)`
	_, err = db.Exec(createUsersTableQuery)
	if err != nil {
		slog.Error("failed to create table", "table", "users", "err", err)
		panic(err)
	}

	createRegistrationsTableQuery := `CREATE TABLE IF NOT EXISTS registrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		event_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE (event_id, user_id)
	)`
	_, err = db.Exec(createRegistrationsTableQuery)
	if err != nil {
		slog.Error("failed to create table", "table", "registrations", "err", err)
		panic(err)
	}

	slog.Info("database schema ready")
}
