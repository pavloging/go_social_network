package domain

import (
	"encoding/json"
	"testing"
)

func TestRawMessageForDLQValidJSON(t *testing.T) {
	raw := []byte(`{"event_id":"1"}`)
	result := RawMessageForDLQ(raw)

	if !json.Valid(result) {
		t.Fatalf("expected valid json, got %s", string(result))
	}

	var obj map[string]any
	if err := json.Unmarshal(result, &obj); err != nil {
		t.Fatalf("expected json object, got error: %v", err)
	}
	if obj["event_id"] != "1" {
		t.Fatalf("expected event_id=1, got %#v", obj["event_id"])
	}
}

func TestRawMessageForDLQInvalidJSON(t *testing.T) {
	raw := []byte("not-json")
	result := RawMessageForDLQ(raw)

	var s string
	if err := json.Unmarshal(result, &s); err != nil {
		t.Fatalf("expected json string, got error: %v", err)
	}
	if s != "not-json" {
		t.Fatalf("expected string value %q, got %q", "not-json", s)
	}
}
