package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestGetEvents_Empty(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodGet, "/events", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	// nil or empty JSON array are both acceptable for empty slice encoding.
	body := strings.TrimSpace(w.Body.String())
	if body != "null" && body != "[]" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestGetEventById_InvalidId(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodGet, "/events/abc", "", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGetEventById_NotFound(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodGet, "/events/999", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not found") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestCreateEvent_Unauthorized(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodPost, "/events", eventJSON("NoAuth"), "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateEvent_Success(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "owner@example.com", "secret123")
	w := doRequest(r, http.MethodPost, "/events", eventJSON("Created"), token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var event map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &event); err != nil {
		t.Fatalf("json: %v", err)
	}
	if int64(event["id"].(float64)) <= 0 {
		t.Fatalf("id: %v", event["id"])
	}
	if int64(event["user_id"].(float64)) <= 0 {
		t.Fatalf("user_id: %v", event["user_id"])
	}
	if event["name"] != "Created" {
		t.Fatalf("name: %v", event["name"])
	}
}

func TestUpdateEvent_Owner(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "owner@example.com", "secret123")
	create := doRequest(r, http.MethodPost, "/events", eventJSON("Original"), token)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := int64(created["id"].(float64))

	w := doRequest(r, http.MethodPut, fmt.Sprintf("/events/%d", id), eventJSON("Updated"), token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var updated map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated["name"] != "Updated" {
		t.Fatalf("name: %v", updated["name"])
	}
}

func TestUpdateEvent_NotOwner(t *testing.T) {
	r := setupRouter(t)
	ownerToken := signupAndLogin(t, r, "owner@example.com", "secret123")
	otherToken := signupAndLogin(t, r, "other@example.com", "secret123")

	create := doRequest(r, http.MethodPost, "/events", eventJSON("Owned"), ownerToken)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := int64(created["id"].(float64))

	w := doRequest(r, http.MethodPut, fmt.Sprintf("/events/%d", id), eventJSON("Hacked"), otherToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not authorized") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestDeleteEvent_Owner(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "owner@example.com", "secret123")
	create := doRequest(r, http.MethodPost, "/events", eventJSON("ToDelete"), token)
	var created map[string]any
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := int64(created["id"].(float64))

	w := doRequest(r, http.MethodDelete, fmt.Sprintf("/events/%d", id), "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "deleted") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestDeleteEvent_Unauthorized(t *testing.T) {
	r := setupRouter(t)
	w := doRequest(r, http.MethodDelete, "/events/1", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
