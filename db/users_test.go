package db

import (
	"testing"

	"events-rest-api/models"
	"events-rest-api/utils"
)

func TestInsertUser_Success(t *testing.T) {
	setupTestDB(t)

	user, err := InsertUser(models.User{
		Email:    "alice@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("InsertUser: %v", err)
	}
	if user.ID <= 0 {
		t.Fatalf("expected positive user ID, got %d", user.ID)
	}
	// Current behavior: returned struct still holds plaintext password.
	if user.Password != "secret123" {
		t.Fatalf("expected returned password to remain plaintext, got %q", user.Password)
	}

	fromDB, err := GetUserByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if fromDB.ID != user.ID {
		t.Fatalf("ID mismatch: insert=%d get=%d", user.ID, fromDB.ID)
	}
	if fromDB.Email != "alice@example.com" {
		t.Fatalf("email mismatch: %q", fromDB.Email)
	}
	if fromDB.Password == "secret123" {
		t.Fatal("stored password should be hashed, not plaintext")
	}
	ok, err := utils.CompareHash("secret123", fromDB.Password)
	if err != nil || !ok {
		t.Fatalf("CompareHash failed: ok=%v err=%v", ok, err)
	}
}

func TestInsertUser_DuplicateEmail(t *testing.T) {
	setupTestDB(t)

	if _, err := InsertUser(models.User{Email: "dup@example.com", Password: "a"}); err != nil {
		t.Fatalf("first InsertUser: %v", err)
	}
	_, err := InsertUser(models.User{Email: "dup@example.com", Password: "b"})
	if err == nil {
		t.Fatal("expected error on duplicate email")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	setupTestDB(t)

	_, err := GetUserByEmail("missing@example.com")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if err != ErrUserNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetUserByEmail_Success(t *testing.T) {
	setupTestDB(t)

	inserted, err := InsertUser(models.User{
		Email:    "bob@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("InsertUser: %v", err)
	}

	fromDB, err := GetUserByEmail("bob@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if fromDB.ID != inserted.ID {
		t.Fatalf("ID mismatch: insert=%d get=%d", inserted.ID, fromDB.ID)
	}
	if fromDB.Email != "bob@example.com" {
		t.Fatalf("email mismatch: %q", fromDB.Email)
	}
	if fromDB.Password == "password123" {
		t.Fatal("stored password should be hashed, not plaintext")
	}
}
