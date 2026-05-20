package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"post-service/internal/domain"
)

type fakeOutboxRepo struct {
	events                     []*domain.OutboxEvent
	published                  []string
	markPublishedCalled        bool
	markRetryableFailureCalled bool
	lastError                  string
	lastRetryDelay             time.Duration
	claimErr                   error
	markPubErr                 error
	markRetryableFailureErr    error
}

func (f *fakeOutboxRepo) ClaimPending(ctx context.Context, workerID string, limit int) ([]*domain.OutboxEvent, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return f.events, nil
}

func (f *fakeOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	f.markPublishedCalled = true
	if f.markPubErr != nil {
		return f.markPubErr
	}
	f.published = append(f.published, id)
	return nil
}

func (f *fakeOutboxRepo) MarkRetryableFailure(ctx context.Context, id string, errText string, retryDelay time.Duration) error {
	f.markRetryableFailureCalled = true
	f.lastError = errText
	f.lastRetryDelay = retryDelay
	if f.markRetryableFailureErr != nil {
		return f.markRetryableFailureErr
	}
	return nil
}

type fakeProducer struct {
	called bool
	err    error
}

func (f *fakeProducer) PublishEvent(ctx context.Context, event domain.EventEnvelope) error {
	f.called = true
	return f.err
}

func (f *fakeProducer) Close() error {
	return nil
}

func newTestWorker(repo domain.OutboxRepository, producer domain.EventProducer) *Worker {
	return NewWorker(
		repo,
		producer,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Second,
		5*time.Second,
		5,
		10,
		"worker-test",
	)
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

	worker := newTestWorker(repo, producer)
	worker.processBatch(context.Background())

	if !producer.called {
		t.Fatal("expected PublishEvent to be called")
	}
	if !repo.markPublishedCalled {
		t.Fatal("expected MarkPublished to be called")
	}
	if len(repo.published) != 1 || repo.published[0] != "event-1" {
		t.Fatalf("expected MarkPublished for event-1, got %#v", repo.published)
	}
	if repo.markRetryableFailureCalled {
		t.Fatal("expected MarkRetryableFailure not to be called on success")
	}
}

func TestWorkerProcessBatchPublishErrorSchedulesRetry(t *testing.T) {
	repo := &fakeOutboxRepo{
		events: []*domain.OutboxEvent{
			{
				ID:       "event-2",
				Attempts: 4,
				Payload: domain.EventEnvelope{
					EventID: "event-2",
					Payload: domain.Post{ID: "post-2"},
				},
			},
		},
	}
	producer := &fakeProducer{err: errors.New("circuit breaker is open")}

	worker := newTestWorker(repo, producer)
	worker.processBatch(context.Background())

	if !producer.called {
		t.Fatal("expected PublishEvent to be called")
	}
	if repo.markPublishedCalled || len(repo.published) != 0 {
		t.Fatal("expected MarkPublished not to be called on publish error")
	}
	if !repo.markRetryableFailureCalled {
		t.Fatal("expected MarkRetryableFailure to be called on publish error")
	}
	if repo.lastError != "circuit breaker is open" {
		t.Fatalf("expected last error to be stored, got %q", repo.lastError)
	}
	if repo.lastRetryDelay != 5*time.Second {
		t.Fatalf("expected retry delay 5s, got %v", repo.lastRetryDelay)
	}
}

func TestWorkerProcessBatchEmptyDoesNotPanic(t *testing.T) {
	repo := &fakeOutboxRepo{events: nil}
	producer := &fakeProducer{}

	worker := newTestWorker(repo, producer)
	worker.processBatch(context.Background())

	if producer.called {
		t.Fatal("expected PublishEvent not to be called for empty batch")
	}
	if repo.markPublishedCalled || repo.markRetryableFailureCalled {
		t.Fatal("expected no repository side effects for empty batch")
	}
}
