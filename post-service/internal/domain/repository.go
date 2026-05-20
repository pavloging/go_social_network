package domain

import (
	"context"
	"time"
)

// PostRepository — интерфейс для работы с БД
type PostRepository interface {
	Save(ctx context.Context, post *Post) error
	SaveWithOutbox(ctx context.Context, post *Post, event EventEnvelope) error
	GetByID(ctx context.Context, id string) (*Post, error)
	List(ctx context.Context) ([]*Post, error)
}

type OutboxRepository interface {
	ClaimPending(ctx context.Context, workerID string, limit int) ([]*OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	// MarkRetryableFailure schedules another publish attempt after a temporary Kafka error.
	MarkRetryableFailure(ctx context.Context, id string, errText string, retryDelay time.Duration) error
}

type CacheRepository interface {
	GetPost(ctx context.Context, id string) (*Post, error)
	SavePost(ctx context.Context, post *Post, ttl time.Duration) error
}

// EventProducer — интерфейс для отправки событий (Kafka)
type EventProducer interface {
	PublishEvent(ctx context.Context, event EventEnvelope) error
	Close() error
}
