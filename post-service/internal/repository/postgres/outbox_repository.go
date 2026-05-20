package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"post-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOutboxRepository(pool *pgxpool.Pool) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{pool: pool}
}

func (r *PostgresOutboxRepository) ClaimPending(
	ctx context.Context,
	workerID string,
	limit int,
	maxAttempts int,
) ([]*domain.OutboxEvent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
WITH picked AS (
    SELECT id
    FROM outbox_events
    WHERE
        status IN ('pending', 'processing')
        AND attempts < $1
        AND next_attempt_at <= NOW()
        AND (
            status = 'pending'
            OR (
                status = 'processing'
                AND locked_at < NOW() - INTERVAL '2 minutes'
            )
        )
    ORDER BY created_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events oe
SET
    status = 'processing',
    locked_by = $3,
    locked_at = NOW(),
    updated_at = NOW()
FROM picked
WHERE oe.id = picked.id
RETURNING
    oe.id,
    oe.event_type,
    oe.payload,
    oe.status,
    oe.attempts,
    oe.next_attempt_at,
    oe.locked_at,
    oe.locked_by,
    oe.created_at,
    oe.updated_at,
    oe.published_at,
    oe.last_error`,
		maxAttempts, limit, workerID)
	if err != nil {
		return nil, fmt.Errorf("claim pending events: %w", err)
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		event, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed events: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}

	return events, nil
}

func (r *PostgresOutboxRepository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE outbox_events
SET
    status = 'published',
    published_at = NOW(),
    updated_at = NOW(),
    locked_at = NULL,
    locked_by = NULL,
    last_error = NULL
WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

func (r *PostgresOutboxRepository) MarkFailedAttempt(
	ctx context.Context,
	id string,
	errText string,
	maxAttempts int,
	retryDelay time.Duration,
) error {
	_, err := r.pool.Exec(ctx, `
UPDATE outbox_events
SET
    attempts = attempts + 1,
    last_error = $2,
    status = CASE
        WHEN attempts + 1 >= $3 THEN 'failed'
        ELSE 'pending'
    END,
    next_attempt_at = CASE
        WHEN attempts + 1 >= $3 THEN NOW()
        ELSE NOW() + ($4::text)::interval
    END,
    locked_at = NULL,
    locked_by = NULL,
    updated_at = NOW()
WHERE id = $1`,
		id, errText, maxAttempts, retryDelay.String())
	if err != nil {
		return fmt.Errorf("mark failed attempt: %w", err)
	}
	return nil
}

type outboxRowScanner interface {
	Scan(dest ...any) error
}

func scanOutboxEvent(row outboxRowScanner) (*domain.OutboxEvent, error) {
	var (
		event       domain.OutboxEvent
		status      string
		payloadJSON []byte
	)

	err := row.Scan(
		&event.ID,
		&event.EventType,
		&payloadJSON,
		&status,
		&event.Attempts,
		&event.NextAttemptAt,
		&event.LockedAt,
		&event.LockedBy,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.PublishedAt,
		&event.LastError,
	)
	if err != nil {
		return nil, fmt.Errorf("scan outbox event: %w", err)
	}

	event.Status = domain.OutboxStatus(status)
	if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal outbox payload: %w", err)
	}

	return &event, nil
}
