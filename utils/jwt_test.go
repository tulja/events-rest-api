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
