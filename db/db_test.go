package db

import "testing"

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
