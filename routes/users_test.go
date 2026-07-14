package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSignup_Success(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodPost, "/signup", `{"email":"new@example.com","password":"secret123"}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["message"] != "Signup successful!" {
		t.Fatalf("message: %q", body["message"])
	}
}

func TestSignup_InvalidJSON(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodPost, "/signup", `{"email":"only@example.com"}`, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	r := setupRouter(t)
	body := `{"email":"dup@example.com","password":"secret123"}`
	if w := doRequest(r, http.MethodPost, "/signup", body, ""); w.Code != http.StatusCreated {
		t.Fatalf("first signup status=%d body=%s", w.Code, w.Body.String())
	}
	w := doRequest(r, http.MethodPost, "/signup", body, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "login@example.com", "secret123")
	if token == "" {
		t.Fatal("empty token")
	}
}

func TestLogin_BadPassword(t *testing.T) {
	r := setupRouter(t)
	_ = signupAndLogin(t, r, "badpw@example.com", "secret123")
	w := doRequest(r, http.MethodPost, "/login", `{"email":"badpw@example.com","password":"wrong"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Invalid credentials") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodPost, "/login", `{"email":"missing@example.com","password":"secret123"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
