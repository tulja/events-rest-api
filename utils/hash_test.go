package utils

import (
	"strings"
	"testing"
)

func TestGenerateHash_Success(t *testing.T) {
	hash, err := GenerateHash("secret123")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "secret123" {
		t.Fatal("hash must not equal plaintext password")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Fatalf("expected bcrypt prefix, got %q", hash)
	}
}

func TestGenerateHash_DifferentSalts(t *testing.T) {
	h1, err := GenerateHash("same-password")
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}
	h2, err := GenerateHash("same-password")
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}
	if h1 == h2 {
		t.Fatal("expected different salts to produce different hashes")
	}
}

func TestCompareHash_Match(t *testing.T) {
	hash, err := GenerateHash("correct-horse")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}
	ok, err := CompareHash("correct-horse", hash)
	if err != nil {
		t.Fatalf("CompareHash: %v", err)
	}
	if !ok {
		t.Fatal("expected match")
	}
}

func TestCompareHash_Mismatch(t *testing.T) {
	hash, err := GenerateHash("correct-horse")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}
	ok, err := CompareHash("wrong-password", hash)
	if err != nil {
		t.Fatalf("CompareHash: %v", err)
	}
	if ok {
		t.Fatal("expected mismatch")
	}
}

func TestCompareHash_InvalidHash(t *testing.T) {
	ok, err := CompareHash("password", "not-a-valid-bcrypt-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash")
	}
	if ok {
		t.Fatal("expected ok=false")
	}
}
