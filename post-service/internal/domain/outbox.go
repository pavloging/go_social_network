package domain

import "time"

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusPublished  OutboxStatus = "published"
	// OutboxStatusFailed is reserved for future non-recoverable outbox errors.
	// Temporary Kafka publish errors keep the event in pending and retry until published.
	OutboxStatusFailed OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID            string
	EventType     string
	Payload       EventEnvelope
	Status        OutboxStatus
	Attempts      int
	NextAttemptAt time.Time
	LockedAt      *time.Time
	LockedBy      *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PublishedAt   *time.Time
	LastError     *string
}
