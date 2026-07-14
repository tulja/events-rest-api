package models

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUser_JSONRoundTrip(t *testing.T) {
	original := User{ID: 1, Email: "a@example.com", Password: "secret"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got User
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != original {
		t.Fatalf("got %+v want %+v", got, original)
	}
}

func TestUser_Bind_Valid(t *testing.T) {
	err := bindJSON(t, `{"email":"a@example.com","password":"secret"}`, &User{})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
}

func TestUser_Bind_MissingEmail(t *testing.T) {
	err := bindJSON(t, `{"password":"secret"}`, &User{})
	if err == nil {
		t.Fatal("expected bind error for missing email")
	}
}

func TestUser_Bind_MissingPassword(t *testing.T) {
	err := bindJSON(t, `{"email":"a@example.com"}`, &User{})
	if err == nil {
		t.Fatal("expected bind error for missing password")
	}
}

func bindJSON(t *testing.T, body string, dst any) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c.ShouldBindJSON(dst)
}
