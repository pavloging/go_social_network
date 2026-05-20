package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"post-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPostRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPostRepository(pool *pgxpool.Pool) *PostgresPostRepository {
	return &PostgresPostRepository{pool: pool}
}

func (r *PostgresPostRepository) Save(ctx context.Context, post *domain.Post) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO posts (id, title, author, content, tags, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		post.ID, post.Title, post.Author, post.Content, post.Tags, post.CreatedAt)
	return err
}

func (r *PostgresPostRepository) SaveWithOutbox(ctx context.Context, post *domain.Post, event domain.EventEnvelope) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO posts (id, title, author, content, tags, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		post.ID, post.Title, post.Author, post.Content, post.Tags, post.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event envelope: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox_events (
			id,
			event_type,
			payload,
			status,
			attempts,
			next_attempt_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, 'pending', 0, NOW(), NOW(), NOW())`,
		event.EventID, event.EventType, payload)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresPostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, author, content, tags, created_at FROM posts WHERE id=$1`, id)

	var p domain.Post
	err := row.Scan(&p.ID, &p.Title, &p.Author, &p.Content, &p.Tags, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresPostRepository) List(ctx context.Context) ([]*domain.Post, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, author, content, tags, created_at FROM posts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*domain.Post
	for rows.Next() {
		var p domain.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Author, &p.Content, &p.Tags, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, nil
}
