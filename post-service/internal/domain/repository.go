package domain

import (
	"context"
	"time"
)

// PostRepository — интерфейс для работы с БД
type PostRepository interface {
	Save(ctx context.Context, post *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	List(ctx context.Context) ([]*Post, error)
}

type CacheRepository interface {
	// List(ctx context.Context) ([]*Post, error)
	GetPost(ctx context.Context, id string) (*Post, error)
	SavePost(ctx context.Context, post *Post, ttl time.Duration) error
}

// EventProducer — интерфейс для отправки событий (Kafka)
type EventProducer interface {
	Publish(ctx context.Context, post *Post) error
}
