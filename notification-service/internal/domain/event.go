package domain

import (
	"encoding/json"
	"time"
)

const (
	EventTypePostCreated = "post.created"
	EventVersion         = 1
)

type Post struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

type EventEnvelope struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    Post      `json:"payload"`
}

type DeadLetterEvent struct {
	OriginalTopic string          `json:"original_topic"`
	OriginalKey   string          `json:"original_key"`
	OriginalValue json.RawMessage `json:"original_value"`
	Error         string          `json:"error"`
	Attempts      int             `json:"attempts"`
	FailedAt      time.Time       `json:"failed_at"`
}

func RawMessageForDLQ(value []byte) json.RawMessage {
	if json.Valid(value) {
		return json.RawMessage(value)
	}

	b, _ := json.Marshal(string(value))
	return json.RawMessage(b)
}
