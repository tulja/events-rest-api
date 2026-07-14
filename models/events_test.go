package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvent_JSONRoundTrip(t *testing.T) {
	dt := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	original := Event{
		ID:          5,
		Name:        "Meetup",
		Description: "Go meetup",
		Location:    "SF",
		DateTime:    dt,
		UserID:      9,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != original.ID || got.Name != original.Name || got.Description != original.Description ||
		got.Location != original.Location || got.UserID != original.UserID {
		t.Fatalf("got %+v want %+v", got, original)
	}
	if !got.DateTime.Equal(original.DateTime) {
		t.Fatalf("DateTime: got %v want %v", got.DateTime, original.DateTime)
	}
}

func TestEvent_Bind_Valid(t *testing.T) {
	body := `{
		"name":"Meetup",
		"description":"Go meetup",
		"location":"SF",
		"date_time":"2026-07-14T12:00:00Z"
	}`
	var event Event
	if err := bindJSON(t, body, &event); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if event.Name != "Meetup" {
		t.Fatalf("name: %q", event.Name)
	}
}

func TestEvent_Bind_MissingRequiredFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing name", `{"description":"d","location":"l","date_time":"2026-07-14T12:00:00Z"}`},
		{"missing description", `{"name":"n","location":"l","date_time":"2026-07-14T12:00:00Z"}`},
		{"missing location", `{"name":"n","description":"d","date_time":"2026-07-14T12:00:00Z"}`},
		{"missing date_time", `{"name":"n","description":"d","location":"l"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var event Event
			if err := bindJSON(t, tc.body, &event); err == nil {
				t.Fatal("expected bind error")
			}
		})
	}
}
