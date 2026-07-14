package routes

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHealth_OK(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodGet, "/health", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field: %q", body["status"])
	}
}
