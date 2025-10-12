package postgres

import (
	"context"
	"post-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPostRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPostRepository(pool *pgxpool.Pool) *PostgresPostRepository {
	return &PostgresPostRepository{pool: pool}
}

// Прокинуть контекст из usecase
// Это нужно для отмены запросов, таймаутов и т.д.
// Потому что в текущем виде контекст всегда будет background
// То есть всегда без отмены и таймаутов
func (r *PostgresPostRepository) Save(ctx context.Context, post *domain.Post) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO posts (id, title, author, content, tags, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		post.ID, post.Title, post.Author, post.Content, post.Tags, post.CreatedAt)
	return err
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
