package db

import (
	"strings"
	"testing"
)

func TestRegisterForEvent_Success(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")
	event := seedEvent(t, user.ID, "Meetup")

	if err := RegisterForEvent(event.ID, user.ID); err != nil {
		t.Fatalf("RegisterForEvent: %v", err)
	}

	events, err := GetAllRegisteredEventsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetAllRegisteredEventsForUser: %v", err)
	}
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("expected registered event %d, got %+v", event.ID, events)
	}
}

func TestRegisterForEvent_EventNotFound(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")

	err := RegisterForEvent(999, user.ID)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterForEvent_Duplicate(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")
	event := seedEvent(t, user.ID, "Meetup")

	if err := RegisterForEvent(event.ID, user.ID); err != nil {
		t.Fatalf("first RegisterForEvent: %v", err)
	}
	err := RegisterForEvent(event.ID, user.ID)
	if err == nil {
		t.Fatal("expected duplicate registration error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAllRegisteredEventsForUser_Empty(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "nobody@example.com", "pass")

	events, err := GetAllRegisteredEventsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetAllRegisteredEventsForUser: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty list, got %d", len(events))
	}
}

func TestGetAllRegisteredEventsForUser_OnlyOwn(t *testing.T) {
	setupTestDB(t)
	userA := seedUser(t, "a@example.com", "pass")
	userB := seedUser(t, "b@example.com", "pass")
	event1 := seedEvent(t, userA.ID, "Event 1")
	event2 := seedEvent(t, userA.ID, "Event 2")

	if err := RegisterForEvent(event1.ID, userA.ID); err != nil {
		t.Fatalf("register A->1: %v", err)
	}
	if err := RegisterForEvent(event2.ID, userB.ID); err != nil {
		t.Fatalf("register B->2: %v", err)
	}

	eventsA, err := GetAllRegisteredEventsForUser(userA.ID)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if len(eventsA) != 1 || eventsA[0].ID != event1.ID {
		t.Fatalf("user A regs: %+v", eventsA)
	}

	eventsB, err := GetAllRegisteredEventsForUser(userB.ID)
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if len(eventsB) != 1 || eventsB[0].ID != event2.ID {
		t.Fatalf("user B regs: %+v", eventsB)
	}
}

func TestDeleteRegistration_Success(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")
	event := seedEvent(t, user.ID, "Meetup")

	if err := RegisterForEvent(event.ID, user.ID); err != nil {
		t.Fatalf("RegisterForEvent: %v", err)
	}
	if err := DeleteRegistration(event.ID, user.ID); err != nil {
		t.Fatalf("DeleteRegistration: %v", err)
	}

	events, err := GetAllRegisteredEventsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetAllRegisteredEventsForUser: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no registrations, got %+v", events)
	}
}

func TestDeleteRegistration_EventNotFound(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")

	err := DeleteRegistration(999, user.ID)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "Event not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteRegistration_NoPriorRegistration(t *testing.T) {
	setupTestDB(t)
	user := seedUser(t, "attendee@example.com", "pass")
	event := seedEvent(t, user.ID, "Meetup")

	// Fixed behavior: returns error when no registration row exists.
	err := DeleteRegistration(event.ID, user.ID)
	if err == nil {
		t.Fatal("expected registration not found error")
	}
	if err != ErrRegistrationNotFound {
		t.Fatalf("expected ErrRegistrationNotFound, got %v", err)
	}
}

func TestDeleteEvent_CascadesRegistrations(t *testing.T) {
	setupTestDB(t)
	owner := seedUser(t, "owner@example.com", "pass")
	attendee := seedUser(t, "attendee@example.com", "pass")
	event := seedEvent(t, owner.ID, "Cascade Me")

	if err := RegisterForEvent(event.ID, attendee.ID); err != nil {
		t.Fatalf("RegisterForEvent: %v", err)
	}
	if err := DeleteEvent(event.ID, owner.ID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}

	// Event gone
	if _, err := GetEventById(event.ID); err != ErrEventNotFound {
		t.Fatalf("expected event not found, got %v", err)
	}
	// Registration row cascaded away — no rows for user
	events, err := GetAllRegisteredEventsForUser(attendee.ID)
	if err != nil {
		t.Fatalf("GetAllRegisteredEventsForUser: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected cascade delete of registrations, got %+v", events)
	}
}
