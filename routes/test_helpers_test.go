package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"events-rest-api/db"
	"events-rest-api/utils"

	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	if err := db.InitInMemory(); err != nil {
		t.Fatalf("InitInMemory: %v", err)
	}
	utils.SetJWTSigningKeyForTest([]byte("route-test-secret"))
	t.Cleanup(utils.ResetJWTSigningKeyForTest)

	r := gin.New()
	RegisterRoutes(r)
	return r
}

func doRequest(r http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body == "" {
		reqBody = bytes.NewBuffer(nil)
	} else {
		reqBody = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func signupAndLogin(t *testing.T, r http.Handler, email, password string) string {
	t.Helper()
	signupBody := map[string]string{"email": email, "password": password}
	raw, _ := json.Marshal(signupBody)
	w := doRequest(r, http.MethodPost, "/signup", string(raw), "")
	if w.Code != http.StatusCreated {
		t.Fatalf("signup status=%d body=%s", w.Code, w.Body.String())
	}

	w = doRequest(r, http.MethodPost, "/login", string(raw), "")
	if w.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("login json: %v", err)
	}
	token, _ := resp["token"].(string)
	if token == "" {
		t.Fatal("expected token in login response")
	}
	return token
}

func eventJSON(name string) string {
	return `{
		"name":"` + name + `",
		"description":"` + name + ` description",
		"location":"Test City",
		"date_time":"2026-07-14T12:00:00Z"
	}`
}
