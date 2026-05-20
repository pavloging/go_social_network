package outbox

import (
	"context"
	"log/slog"
	"time"

	"post-service/internal/domain"

	"github.com/google/uuid"
)

type Worker struct {
	repo         domain.OutboxRepository
	producer     domain.EventProducer
	log          *slog.Logger
	pollInterval time.Duration
	retryDelay   time.Duration
	maxAttempts  int
	batchSize    int
	workerID     string
}

func NewWorker(
	repo domain.OutboxRepository,
	producer domain.EventProducer,
	log *slog.Logger,
	pollInterval time.Duration,
	retryDelay time.Duration,
	maxAttempts int,
	batchSize int,
	workerID string,
) *Worker {
	if workerID == "" {
		workerID = uuid.NewString()
	}

	return &Worker{
		repo:         repo,
		producer:     producer,
		log:          log,
		pollInterval: pollInterval,
		retryDelay:   retryDelay,
		maxAttempts:  maxAttempts,
		batchSize:    batchSize,
		workerID:     workerID,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.log.Info("outbox worker started", slog.String("worker_id", w.workerID))

	for {
		select {
		case <-ctx.Done():
			w.log.Info("outbox worker stopped", slog.String("worker_id", w.workerID))
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.repo.ClaimPending(ctx, w.workerID, w.batchSize, w.maxAttempts)
	if err != nil {
		w.log.Error("failed to claim pending outbox events",
			slog.String("worker_id", w.workerID),
			slog.Any("error", err))
		return
	}

	if len(events) == 0 {
		return
	}

	for _, event := range events {
		w.log.Info("publishing outbox event",
			slog.String("worker_id", w.workerID),
			slog.String("event_id", event.ID),
			slog.String("event_type", event.EventType))

		if err := w.producer.PublishEvent(ctx, event.Payload); err != nil {
			w.log.Error("failed to publish outbox event",
				slog.String("worker_id", w.workerID),
				slog.String("event_id", event.ID),
				slog.Any("error", err))

			if markErr := w.repo.MarkFailedAttempt(ctx, event.ID, err.Error(), w.maxAttempts, w.retryDelay); markErr != nil {
				w.log.Error("failed to mark outbox event as failed attempt",
					slog.String("worker_id", w.workerID),
					slog.String("event_id", event.ID),
					slog.Any("error", markErr))
			}
			continue
		}

		if err := w.repo.MarkPublished(ctx, event.ID); err != nil {
			w.log.Error("failed to mark outbox event as published",
				slog.String("worker_id", w.workerID),
				slog.String("event_id", event.ID),
				slog.Any("error", err))
			continue
		}

		w.log.Info("outbox event published",
			slog.String("worker_id", w.workerID),
			slog.String("event_id", event.ID))
	}
}
