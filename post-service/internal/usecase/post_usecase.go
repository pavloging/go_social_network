package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"post-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type PostUsecase struct {
	repo  domain.PostRepository
	cache domain.CacheRepository
	log   *slog.Logger
}

func NewPostUsecase(postRepo domain.PostRepository, cache domain.CacheRepository, log *slog.Logger) *PostUsecase {
	return &PostUsecase{
		repo:  postRepo,
		cache: cache,
		log:   log,
	}
}

func (u *PostUsecase) List(ctx context.Context) ([]*domain.Post, error) {
	posts, err := u.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}

	return posts, nil
}

func (u *PostUsecase) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	post, _ := u.cache.GetPost(ctx, id)
	if post != nil {
		return post, nil
	}

	post, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get post by id %s: %w", id, err)
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = u.cache.SavePost(cacheCtx, post, 5*time.Minute)
	}()

	return post, nil
}

func (u *PostUsecase) CreatePost(ctx context.Context, title, author, content string, tags []string) (*domain.Post, error) {
	post := &domain.Post{
		ID:        uuid.NewString(),
		Title:     title,
		Author:    author,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now().UTC(),
	}

	event := domain.EventEnvelope{
		EventID:    uuid.NewString(),
		EventType:  domain.EventTypePostCreated,
		Version:    domain.EventVersion,
		OccurredAt: time.Now().UTC(),
		Payload:    *post,
	}

	if err := u.repo.SaveWithOutbox(ctx, post, event); err != nil {
		return nil, fmt.Errorf("failed to save post with outbox event: %w", err)
	}

	u.log.Info("post created",
		slog.String("post_id", post.ID),
		slog.String("title", post.Title),
		slog.String("author", post.Author))
	u.log.Info("outbox event created",
		slog.String("event_id", event.EventID),
		slog.String("post_id", post.ID))

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = u.cache.SavePost(cacheCtx, post, 5*time.Minute)
	}()

	return post, nil
}
