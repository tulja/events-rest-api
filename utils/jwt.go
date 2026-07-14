package utils

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"events-rest-api/models"
	"events-rest-api/secrets"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtMu         sync.Mutex
	jwtSigningKey []byte
)

// SetJWTSigningKeyForTest injects a signing key for unit tests (bypasses Vault).
func SetJWTSigningKeyForTest(key []byte) {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	jwtSigningKey = append([]byte(nil), key...)
}

// ResetJWTSigningKeyForTest clears the cached JWT signing key state.
func ResetJWTSigningKeyForTest() {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	jwtSigningKey = nil
}

// EnsureJWTSigningKey loads the signing key from Vault if not already cached.
// Call at process startup so missing Vault/config fails fast.
func EnsureJWTSigningKey() error {
	_, err := loadJWTSigningKey()
	return err
}

// loadJWTSigningKey returns the cached key, or loads it from Vault on success only.
// Failed loads are not cached so a later retry can succeed after Vault recovers.
func loadJWTSigningKey() ([]byte, error) {
	jwtMu.Lock()
	defer jwtMu.Unlock()

	if len(jwtSigningKey) > 0 {
		return jwtSigningKey, nil
	}

	client, err := secrets.NewClient(nil)
	if err != nil {
		err = fmt.Errorf("failed to create vault client: %w", err)
		slog.Error("failed to load JWT signing key", "err", err)
		return nil, err
	}

	keyStr, err := client.GetSecretValue(context.Background(), "events-api/jwt", "signing-key")
	if err != nil {
		err = fmt.Errorf("failed to read secret events-api/jwt/signing-key from Vault: %w", err)
		slog.Error("failed to load JWT signing key", "err", err)
		return nil, err
	}

	jwtSigningKey = []byte(keyStr)
	slog.Info("JWT signing key loaded from Vault", "path", "events-api/jwt", "key", "signing-key")
	return jwtSigningKey, nil
}

func GenerateToken(user models.User) (string, error) {
	signingKey, err := loadJWTSigningKey()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	signed, err := token.SignedString(signingKey)
	if err != nil {
		slog.Error("failed to sign JWT", "userId", user.ID, "err", err)
		return "", err
	}
	return signed, nil
}

func VerifyToken(token string) (int64, error) {
	signingKey, err := loadJWTSigningKey()
	if err != nil {
		return 0, err
	}
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			slog.Warn("unexpected JWT signing method", "alg", t.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingKey, nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}
	if !parsedToken.Valid {
		return 0, fmt.Errorf("invalid token")
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	userId := int64(claims["user_id"].(float64))
	return userId, nil
}
