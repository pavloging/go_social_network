package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"

	"github.com/IBM/sarama"
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
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	consumer := newTestConsumer(nil, uc)

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
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	consumer := newTestConsumer(nil, uc)

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
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	consumer := newTestConsumer(nil, uc)

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

type fakeDLQ struct {
	err       error
	published int
}

func (f *fakeDLQ) Publish(ctx context.Context, event domain.DeadLetterEvent) error {
	f.published++
	return f.err
}

type fakeSession struct {
	ctx         context.Context
	markCount   int
	commitCount int
}

func (s *fakeSession) Claims() map[string][]int32                  { return nil }
func (s *fakeSession) MemberID() string                            { return "test-member" }
func (s *fakeSession) GenerationID() int32                         { return 1 }
func (s *fakeSession) MarkOffset(string, int32, int64, string)     {}
func (s *fakeSession) ResetOffset(string, int32, int64, string)    {}
func (s *fakeSession) MarkMessage(*sarama.ConsumerMessage, string) { s.markCount++ }
func (s *fakeSession) Commit()                                     { s.commitCount++ }
func (s *fakeSession) Context() context.Context                    { return s.ctx }

func newTestConsumer(dlq dlqPublisher, uc *usecase.NotificationUsecase) *Consumer {
	return &Consumer{
		log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		uc:           uc,
		dlq:          dlq,
		retryMax:     3,
		retryBackoff: time.Millisecond,
	}
}

func TestHandleMessageDLQPublishFailedNoCommit(t *testing.T) {
	dlq := &fakeDLQ{err: errors.New("dlq unavailable")}
	consumer := newTestConsumer(dlq, usecase.NewNotificationUsecase(&fakeNotificationStore{}, slog.New(slog.NewTextHandler(io.Discard, nil))))

	sess := &fakeSession{ctx: context.Background()}
	msg := &sarama.ConsumerMessage{
		Topic:     "posts",
		Partition: 0,
		Offset:    10,
		Key:       []byte("key-1"),
		Value:     []byte("not-json"),
	}

	err := consumer.handleMessage(sess, msg)
	if err == nil {
		t.Fatal("expected error when DLQ publish fails")
	}
	if sess.markCount != 0 {
		t.Fatalf("expected mark count 0, got %d", sess.markCount)
	}
	if sess.commitCount != 0 {
		t.Fatalf("expected commit count 0, got %d", sess.commitCount)
	}
	if dlq.published != 1 {
		t.Fatalf("expected one DLQ publish attempt, got %d", dlq.published)
	}
}

func TestHandleMessageDLQPublishSuccessCommits(t *testing.T) {
	dlq := &fakeDLQ{}
	consumer := newTestConsumer(dlq, usecase.NewNotificationUsecase(&fakeNotificationStore{}, slog.New(slog.NewTextHandler(io.Discard, nil))))

	sess := &fakeSession{ctx: context.Background()}
	msg := &sarama.ConsumerMessage{
		Topic:     "posts",
		Partition: 0,
		Offset:    11,
		Key:       []byte("key-2"),
		Value:     []byte("not-json"),
	}

	err := consumer.handleMessage(sess, msg)
	if err != nil {
		t.Fatalf("expected nil error on successful DLQ publish, got %v", err)
	}
	if sess.markCount != 1 {
		t.Fatalf("expected mark count 1, got %d", sess.markCount)
	}
	if sess.commitCount != 1 {
		t.Fatalf("expected commit count 1, got %d", sess.commitCount)
	}
	if dlq.published != 1 {
		t.Fatalf("expected one DLQ publish, got %d", dlq.published)
	}
}

func TestHandleMessageSuccessfulProcessCommits(t *testing.T) {
	store := &fakeNotificationStore{created: true}
	uc := usecase.NewNotificationUsecase(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	consumer := newTestConsumer(&fakeDLQ{}, uc)

	event := domain.EventEnvelope{
		EventID:    "event-ok",
		EventType:  domain.EventTypePostCreated,
		Version:    domain.EventVersion,
		OccurredAt: time.Now().UTC(),
		Payload: domain.Post{
			ID:        "post-ok",
			Author:    "author",
			CreatedAt: time.Now().UTC(),
		},
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	sess := &fakeSession{ctx: context.Background()}
	msg := &sarama.ConsumerMessage{
		Topic:     "posts",
		Partition: 0,
		Offset:    12,
		Value:     raw,
	}

	if err := consumer.handleMessage(sess, msg); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if sess.markCount != 1 || sess.commitCount != 1 {
		t.Fatalf("expected one mark and one commit, got mark=%d commit=%d", sess.markCount, sess.commitCount)
	}
}
