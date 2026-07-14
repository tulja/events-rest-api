package db

import (
	"database/sql"
	"testing"
	"time"

	"events-rest-api/models"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = testDB.Close()
	})

	// :memory: is per-connection; keep a single connection for schema visibility.
	testDB.SetMaxOpenConns(1)
	testDB.SetMaxIdleConns(1)

	db = testDB
	enableForeignKeys()
	createTables()
}

func seedUser(t *testing.T, email, password string) models.User {
	t.Helper()
	user, err := InsertUser(models.User{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("seedUser(%q): %v", email, err)
	}
	return user
}

func seedEvent(t *testing.T, userID int64, name string) models.Event {
	t.Helper()
	event := models.Event{
		Name:        name,
		Description: name + " description",
		Location:    "Test Location",
		DateTime:    fixedEventTime(),
		UserID:      userID,
	}
	if err := InsertEvent(&event); err != nil {
		t.Fatalf("seedEvent(%q): %v", name, err)
	}
	return event
}

func fixedEventTime() time.Time {
	return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
}
