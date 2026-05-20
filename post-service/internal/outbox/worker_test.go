package outbox

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"post-service/internal/domain"
)

type fakeOutboxRepo struct {
	events      []*domain.OutboxEvent
	published   []string
	failed      []string
	claimErr    error
	publishErr  error
	markPubErr  error
	markFailErr error
}

func (f *fakeOutboxRepo) ClaimPending(ctx context.Context, workerID string, limit int, maxAttempts int) ([]*domain.OutboxEvent, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return f.events, nil
}

func (f *fakeOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	if f.markPubErr != nil {
		return f.markPubErr
	}
	f.published = append(f.published, id)
	return nil
}

func (f *fakeOutboxRepo) MarkFailedAttempt(ctx context.Context, id string, errText string, maxAttempts int, retryDelay time.Duration) error {
	if f.markFailErr != nil {
		return f.markFailErr
	}
	f.failed = append(f.failed, id)
	return nil
}

type fakeProducer struct {
	err error
}

func (f *fakeProducer) PublishEvent(ctx context.Context, event domain.EventEnvelope) error {
	return f.err
}

func (f *fakeProducer) Close() error {
	return nil
}

func TestWorkerProcessBatchPublishSuccess(t *testing.T) {
	repo := &fakeOutboxRepo{
		events: []*domain.OutboxEvent{
			{
				ID: "event-1",
				Payload: domain.EventEnvelope{
					EventID:   "event-1",
					EventType: domain.EventTypePostCreated,
					Version:   domain.EventVersion,
					Payload: domain.Post{
						ID: "post-1",
					},
				},
			},
		},
	}
	producer := &fakeProducer{}

	worker := NewWorker(repo, producer, slog.New(slog.NewTextHandler(os.Stdout, nil)), time.Second, time.Second, 5, 10, "worker-test")
	worker.processBatch(context.Background())

	if len(repo.published) != 1 || repo.published[0] != "event-1" {
		t.Fatalf("expected MarkPublished for event-1, got %#v", repo.published)
	}
	if len(repo.failed) != 0 {
		t.Fatalf("expected no failed marks, got %#v", repo.failed)
	}
}

func TestWorkerProcessBatchPublishFailure(t *testing.T) {
	repo := &fakeOutboxRepo{
		events: []*domain.OutboxEvent{
			{
				ID: "event-2",
				Payload: domain.EventEnvelope{
					EventID: "event-2",
					Payload: domain.Post{ID: "post-2"},
				},
			},
		},
	}
	producer := &fakeProducer{err: errors.New("kafka unavailable")}

	worker := NewWorker(repo, producer, slog.New(slog.NewTextHandler(os.Stdout, nil)), time.Second, time.Second, 5, 10, "worker-test")
	worker.processBatch(context.Background())

	if len(repo.failed) != 1 || repo.failed[0] != "event-2" {
		t.Fatalf("expected MarkFailedAttempt for event-2, got %#v", repo.failed)
	}
	if len(repo.published) != 0 {
		t.Fatalf("expected no published marks, got %#v", repo.published)
	}
}

func TestWorkerProcessBatchEmptyDoesNotPanic(t *testing.T) {
	repo := &fakeOutboxRepo{events: nil}
	producer := &fakeProducer{}

	worker := NewWorker(repo, producer, slog.New(slog.NewTextHandler(os.Stdout, nil)), time.Second, time.Second, 5, 10, "worker-test")
	worker.processBatch(context.Background())
}
