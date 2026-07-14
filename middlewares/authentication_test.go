package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"events-rest-api/models"
	"events-rest-api/utils"

	"github.com/gin-gonic/gin"
)

func setupAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	utils.SetJWTSigningKeyForTest([]byte("middleware-test-secret"))
	t.Cleanup(utils.ResetJWTSigningKeyForTest)

	r := gin.New()
	r.Use(Authenticate)
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userId": c.GetInt64("userId")})
	})
	return r
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	r := setupAuthRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusUnauthorized)
	}
	assertUnauthorizedBody(t, w.Body.Bytes())
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	r := setupAuthRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "not-a-jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusUnauthorized)
	}
	assertUnauthorizedBody(t, w.Body.Bytes())
}

func TestAuthenticate_ValidToken(t *testing.T) {
	r := setupAuthRouter(t)
	token, err := utils.GenerateToken(models.User{ID: 99, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if int64(body["userId"].(float64)) != 99 {
		t.Fatalf("userId: got %v want 99", body["userId"])
	}
}

func TestAuthenticate_BearerPrefix(t *testing.T) {
	r := setupAuthRouter(t)
	token, err := utils.GenerateToken(models.User{ID: 7, Email: "bearer@example.com"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if int64(body["userId"].(float64)) != 7 {
		t.Fatalf("userId: got %v want 7", body["userId"])
	}
}

func TestAuthenticate_WrongKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	utils.SetJWTSigningKeyForTest([]byte("key-a"))
	token, err := utils.GenerateToken(models.User{ID: 1, Email: "a@example.com"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	utils.SetJWTSigningKeyForTest([]byte("key-b"))
	t.Cleanup(utils.ResetJWTSigningKeyForTest)

	r := gin.New()
	r.Use(Authenticate)
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusUnauthorized)
	}
}

func assertUnauthorizedBody(t *testing.T, raw []byte) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json: %v body=%s", err, string(raw))
	}
	if body["error"] != "Unauthorized" {
		t.Fatalf("error field: got %q", body["error"])
	}
}
