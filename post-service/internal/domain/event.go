package domain

import "time"

const (
	EventTypePostCreated = "post.created"
	EventVersion         = 1
)

type EventEnvelope struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    Post      `json:"payload"`
}
