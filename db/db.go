package db

import (
	"database/sql"
	"log/slog"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func InitDB() {
	const path = "./events.db"
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		slog.Error("failed to open database", "path", path, "err", err)
		panic(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	slog.Info("opened sqlite database", "path", path)
	enableForeignKeys()
	createTables()
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
