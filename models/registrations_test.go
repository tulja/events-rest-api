package models

import (
	"encoding/json"
	"testing"
)

func TestRegistration_JSONRoundTrip(t *testing.T) {
	original := Registration{ID: 3, EventID: 10, UserID: 20}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Registration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != original {
		t.Fatalf("got %+v want %+v", got, original)
	}

	// Confirm JSON field names.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("raw unmarshal: %v", err)
	}
	for _, key := range []string{"id", "event_id", "user_id"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing json key %q in %s", key, string(data))
		}
	}
}
