package db

import (
	"strings"
	"testing"
	"time"

	"events-rest-api/models"
)

func TestInsertEvent_Success(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "owner@example.com", "pass")

	event := models.Event{
		Name:        "GopherCon",
		Description: "Go conference",
		Location:    "San Diego",
		DateTime:    fixedEventTime(),
		UserID:      user.ID,
	}
	if err := InsertEvent(&event); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if event.ID <= 0 {
		t.Fatalf("expected positive event ID, got %d", event.ID)
	}

	got, err := GetEventById(event.ID)
	if err != nil {
		t.Fatalf("GetEventById: %v", err)
	}
	if got.Name != event.Name || got.Description != event.Description || got.Location != event.Location {
		t.Fatalf("field mismatch: %+v", got)
	}
	if got.UserID != user.ID {
		t.Fatalf("UserID: got %d want %d", got.UserID, user.ID)
	}
	if !got.DateTime.Equal(fixedEventTime()) && !almostEqualTime(got.DateTime, fixedEventTime()) {
		t.Fatalf("DateTime: got %v want %v", got.DateTime, fixedEventTime())
	}
}

func TestGetAllEvents_Empty(t *testing.T) {
	setupTestDB(t)

	events, err := GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty list, got %d", len(events))
	}
}

func TestGetAllEvents_Multiple(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "owner@example.com", "pass")
	e1 := seedEvent(t, user.ID, "Event A")
	e2 := seedEvent(t, user.ID, "Event B")

	events, err := GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	ids := map[int64]bool{events[0].ID: true, events[1].ID: true}
	if !ids[e1.ID] || !ids[e2.ID] {
		t.Fatalf("missing expected ids: got %+v", events)
	}
}

func TestGetEventById_NotFound(t *testing.T) {
	setupTestDB(t)

	_, err := GetEventById(999)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEvent_Owner(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "owner@example.com", "pass")
	original := seedEvent(t, user.ID, "Original")

	updatedInput := models.Event{
		Name:        "Updated",
		Description: "Updated desc",
		Location:    "New York",
		DateTime:    fixedEventTime().Add(24 * time.Hour),
	}
	updated, err := UpdateEvent(original.ID, updatedInput, user.ID)
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if updated.Name != "Updated" || updated.Location != "New York" {
		t.Fatalf("unexpected update result: %+v", updated)
	}
	if updated.UserID != user.ID {
		t.Fatalf("UserID changed: got %d", updated.UserID)
	}

	got, err := GetEventById(original.ID)
	if err != nil {
		t.Fatalf("GetEventById: %v", err)
	}
	if got.Name != "Updated" || got.Description != "Updated desc" {
		t.Fatalf("db not updated: %+v", got)
	}
}

func TestUpdateEvent_NotOwner(t *testing.T) {
	setupTestDB(t)
	owner := seedUser(t, "owner@example.com", "pass")
	other := seedUser(t, "other@example.com", "pass")
	original := seedEvent(t, owner.ID, "Owned Event")

	_, err := UpdateEvent(original.ID, models.Event{
		Name:        "Hacked",
		Description: "nope",
		Location:    "nowhere",
		DateTime:    fixedEventTime(),
	}, other.ID)
	if err == nil {
		t.Fatal("expected authorization error")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := GetEventById(original.ID)
	if err != nil {
		t.Fatalf("GetEventById: %v", err)
	}
	if got.Name != "Owned Event" {
		t.Fatalf("event should be unchanged, got %q", got.Name)
	}
}

func TestUpdateEvent_NotFound(t *testing.T) {
	setupTestDB(t)

	_, err := UpdateEvent(999, models.Event{
		Name:        "X",
		Description: "Y",
		Location:    "Z",
		DateTime:    fixedEventTime(),
	}, 1)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteEvent_Owner(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "owner@example.com", "pass")
	event := seedEvent(t, user.ID, "To Delete")

	if err := DeleteEvent(event.ID, user.ID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
	_, err := GetEventById(event.ID)
	if err == nil {
		t.Fatal("expected event to be deleted")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteEvent_NotOwner(t *testing.T) {
	setupTestDB(t)
	owner := seedUser(t, "owner@example.com", "pass")
	other := seedUser(t, "other@example.com", "pass")
	event := seedEvent(t, owner.ID, "Still Here")

	err := DeleteEvent(event.ID, other.ID)
	if err == nil {
		t.Fatal("expected authorization error")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := GetEventById(event.ID); err != nil {
		t.Fatalf("event should still exist: %v", err)
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	setupTestDB(t)

	err := DeleteEvent(999, 1)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func almostEqualTime(a, b time.Time) bool {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Second
}
