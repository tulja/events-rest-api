package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func createEventWithToken(t *testing.T, r http.Handler, token, name string) int64 {
	t.Helper()
	w := doRequest(r, http.MethodPost, "/events", eventJSON(name), token)
	if w.Code != http.StatusCreated {
		t.Fatalf("create event status=%d body=%s", w.Code, w.Body.String())
	}
	var event map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &event); err != nil {
		t.Fatalf("json: %v", err)
	}
	return int64(event["id"].(float64))
}

func TestRegisterForEvent_Success(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "attendee@example.com", "secret123")
	eventID := createEventWithToken(t, r, token, "Meetup")

	w := doRequest(r, http.MethodPost, fmt.Sprintf("/events/%d/register", eventID), "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Registered") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestRegisterForEvent_Duplicate(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "attendee@example.com", "secret123")
	eventID := createEventWithToken(t, r, token, "Meetup")

	path := fmt.Sprintf("/events/%d/register", eventID)
	if w := doRequest(r, http.MethodPost, path, "", token); w.Code != http.StatusOK {
		t.Fatalf("first register status=%d body=%s", w.Code, w.Body.String())
	}
	w := doRequest(r, http.MethodPost, path, "", token)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "already registered") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestCancelRegistration_Success(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "attendee@example.com", "secret123")
	eventID := createEventWithToken(t, r, token, "Meetup")

	path := fmt.Sprintf("/events/%d/register", eventID)
	if w := doRequest(r, http.MethodPost, path, "", token); w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	w := doRequest(r, http.MethodDelete, path, "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Cancelled") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestGetAllRegistrations(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "attendee@example.com", "secret123")
	eventID := createEventWithToken(t, r, token, "Meetup")
	if w := doRequest(r, http.MethodPost, fmt.Sprintf("/events/%d/register", eventID), "", token); w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}

	w := doRequest(r, http.MethodGet, "/events/registrations", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var events []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("json: %v body=%s", err, w.Body.String())
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(events))
	}
	if int64(events[0]["id"].(float64)) != eventID {
		t.Fatalf("event id: %v want %d", events[0]["id"], eventID)
	}
}

func TestRegister_InvalidEventId(t *testing.T) {
	r := setupRouter(t)
	token := signupAndLogin(t, r, "attendee@example.com", "secret123")
	w := doRequest(r, http.MethodPost, "/events/abc/register", "", token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
