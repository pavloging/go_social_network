package domain

import "time"

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
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
