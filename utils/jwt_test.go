package utils

import (
	"testing"
	"time"

	"events-rest-api/models"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndVerifyToken_RoundTrip(t *testing.T) {
	SetJWTSigningKeyForTest([]byte("test-signing-key"))
	t.Cleanup(ResetJWTSigningKeyForTest)

	user := models.User{ID: 42, Email: "alice@example.com"}
	token, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	userID, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if userID != 42 {
		t.Fatalf("userId: got %d want 42", userID)
	}
}

func TestVerifyToken_InvalidSignature(t *testing.T) {
	SetJWTSigningKeyForTest([]byte("key-a"))
	token, err := GenerateToken(models.User{ID: 1, Email: "a@example.com"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	SetJWTSigningKeyForTest([]byte("key-b"))
	t.Cleanup(ResetJWTSigningKeyForTest)

	if _, err := VerifyToken(token); err == nil {
		t.Fatal("expected signature verification error")
	}
}

func TestVerifyToken_Malformed(t *testing.T) {
	SetJWTSigningKeyForTest([]byte("test-signing-key"))
	t.Cleanup(ResetJWTSigningKeyForTest)

	if _, err := VerifyToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestVerifyToken_Empty(t *testing.T) {
	SetJWTSigningKeyForTest([]byte("test-signing-key"))
	t.Cleanup(ResetJWTSigningKeyForTest)

	if _, err := VerifyToken(""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestGenerateToken_IncludesClaims(t *testing.T) {
	SetJWTSigningKeyForTest([]byte("test-signing-key"))
	t.Cleanup(ResetJWTSigningKeyForTest)

	tokenStr, err := GenerateToken(models.User{ID: 7, Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return []byte("test-signing-key"), nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	if claims["email"] != "bob@example.com" {
		t.Fatalf("email claim: %v", claims["email"])
	}
	exp, ok := claims["exp"].(float64)
	if !ok || exp <= float64(time.Now().Unix()) {
		t.Fatalf("expected future exp claim, got %v", claims["exp"])
	}
}

func TestLoadJWTSigningKey_FromEnv(t *testing.T) {
	ResetJWTSigningKeyForTest()
	t.Cleanup(ResetJWTSigningKeyForTest)

	// Ensure Vault is not required when env is set.
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SIGNING_KEY", "env-signing-key-value")

	if err := EnsureJWTSigningKey(); err != nil {
		t.Fatalf("EnsureJWTSigningKey from env: %v", err)
	}

	user := models.User{ID: 9, Email: "env@example.com"}
	token, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	userID, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if userID != 9 {
		t.Fatalf("userId: got %d want 9", userID)
	}
}

func TestLoadJWTSigningKey_JWTSecretFallback(t *testing.T) {
	ResetJWTSigningKeyForTest()
	t.Cleanup(ResetJWTSigningKeyForTest)

	t.Setenv("JWT_SIGNING_KEY", "")
	t.Setenv("JWT_SECRET", "legacy-secret-name")

	if err := EnsureJWTSigningKey(); err != nil {
		t.Fatalf("EnsureJWTSigningKey from JWT_SECRET: %v", err)
	}

	token, err := GenerateToken(models.User{ID: 1, Email: "a@b.c"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := VerifyToken(token); err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
}

func TestEnsureJWTSigningKey_MissingConfig(t *testing.T) {
	ResetJWTSigningKeyForTest()
	t.Cleanup(ResetJWTSigningKeyForTest)

	t.Setenv("JWT_SIGNING_KEY", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("VAULT_ADDR", "http://127.0.0.1:1") // nothing listening
	t.Setenv("VAULT_TOKEN", "not-used-if-unreachable")

	err := EnsureJWTSigningKey()
	if err == nil {
		t.Fatal("expected error when env and Vault are unavailable")
	}
}
