package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"events-rest-api/models"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtMu         sync.Mutex
	jwtSigningKey []byte
)

// SetJWTSigningKeyForTest injects a signing key for unit tests (bypasses env).
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

// EnsureJWTSigningKey loads the signing key if not already cached.
// Call at process startup so missing config fails fast.
//
// Resolution order:
//  1. JWT_SIGNING_KEY
//  2. JWT_SECRET (legacy alias)
func EnsureJWTSigningKey() error {
	_, err := loadJWTSigningKey()
	return err
}

// loadJWTSigningKey returns the cached key, or loads it from the environment.
// Failed loads are not cached so a later retry can succeed after the env is set.
func loadJWTSigningKey() ([]byte, error) {
	jwtMu.Lock()
	defer jwtMu.Unlock()

	if len(jwtSigningKey) > 0 {
		return jwtSigningKey, nil
	}

	if key, source, ok := signingKeyFromEnv(); ok {
		jwtSigningKey = key
		slog.Info("JWT signing key loaded from environment", "source", source)
		return jwtSigningKey, nil
	}

	err := fmt.Errorf("JWT signing key not set: set JWT_SIGNING_KEY (or JWT_SECRET) environment variable")
	slog.Error("failed to load JWT signing key", "err", err)
	return nil, err
}

// signingKeyFromEnv returns the key from JWT_SIGNING_KEY or JWT_SECRET when non-empty.
func signingKeyFromEnv() (key []byte, source string, ok bool) {
	if v := strings.TrimSpace(os.Getenv("JWT_SIGNING_KEY")); v != "" {
		return []byte(v), "JWT_SIGNING_KEY", true
	}
	if v := strings.TrimSpace(os.Getenv("JWT_SECRET")); v != "" {
		return []byte(v), "JWT_SECRET", true
	}
	return nil, "", false
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
