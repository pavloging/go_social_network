package kafka

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

func TestValidateEventSuccess(t *testing.T) {
	event := domain.EventEnvelope{
		EventID:   "event-1",
		EventType: domain.EventTypePostCreated,
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-1",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := validateEvent(event); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateEventEmptyEventID(t *testing.T) {
	event := domain.EventEnvelope{
		EventType: domain.EventTypePostCreated,
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-1",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := validateEvent(event); err == nil || err.Error() != "event_id is required" {
		t.Fatalf("expected event_id is required, got %v", err)
	}
}

func TestValidateEventInvalidEventType(t *testing.T) {
	event := domain.EventEnvelope{
		EventID:   "event-1",
		EventType: "post.updated",
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-1",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := validateEvent(event); err == nil || err.Error() != "unsupported event_type" {
		t.Fatalf("expected unsupported event_type, got %v", err)
	}
}

func TestValidateEventInvalidVersion(t *testing.T) {
	event := domain.EventEnvelope{
		EventID:   "event-1",
		EventType: domain.EventTypePostCreated,
		Version:   2,
		Payload: domain.Post{
			ID:        "post-1",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := validateEvent(event); err == nil || err.Error() != "unsupported event version" {
		t.Fatalf("expected unsupported event version, got %v", err)
	}
}

type fakeNotificationStore struct {
	err     error
	created bool
}

func (f *fakeNotificationStore) SaveNotificationOnce(ctx context.Context, eventID string, notification domain.Notification) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.created, nil
}

func TestProcessWithRetrySuccessFirstAttempt(t *testing.T) {
	store := &fakeNotificationStore{created: true}
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(os.Stdout, nil)))

	consumer := NewConsumer(slog.New(slog.NewTextHandler(os.Stdout, nil)), uc, nil, 3, time.Millisecond)

	event := domain.EventEnvelope{
		EventID:   "event-1",
		EventType: domain.EventTypePostCreated,
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-1",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := consumer.processWithRetry(context.Background(), event); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestProcessWithRetryMaxAttempts(t *testing.T) {
	store := &fakeNotificationStore{err: errors.New("redis down")}
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(os.Stdout, nil)))

	consumer := NewConsumer(slog.New(slog.NewTextHandler(os.Stdout, nil)), uc, nil, 3, time.Millisecond)

	event := domain.EventEnvelope{
		EventID:   "event-2",
		EventType: domain.EventTypePostCreated,
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-2",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	err := consumer.processWithRetry(context.Background(), event)
	if err == nil {
		t.Fatal("expected error after max attempts")
	}
}

func TestProcessWithRetryDuplicateIsSuccess(t *testing.T) {
	store := &fakeNotificationStore{created: false}
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(os.Stdout, nil)))

	consumer := NewConsumer(slog.New(slog.NewTextHandler(os.Stdout, nil)), uc, nil, 3, time.Millisecond)

	event := domain.EventEnvelope{
		EventID:   "event-3",
		EventType: domain.EventTypePostCreated,
		Version:   domain.EventVersion,
		Payload: domain.Post{
			ID:        "post-3",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := consumer.processWithRetry(context.Background(), event); err != nil {
		t.Fatalf("expected duplicate to be success, got %v", err)
	}
}
