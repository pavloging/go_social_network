package usecase

import (
	"context"
	"log/slog"
	"post-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type PostUsecase struct {
	repo     domain.PostRepository  // db - postgres
	producer domain.EventProducer   // kafka
	cache    domain.CacheRepository // redis
}

func NewPostUsecase(poolRepo domain.PostRepository, producer domain.EventProducer, cache domain.CacheRepository) *PostUsecase {
	return &PostUsecase{
		repo:     poolRepo,
		producer: producer,
		cache:    cache,
	}
}

func (u *PostUsecase) List(ctx context.Context) ([]*domain.Post, error) {
	posts, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (u *PostUsecase) GetByID(log *slog.Logger, ctx context.Context, id string) (*domain.Post, error) {
	// 1. Проверяем кеш
	post, err := u.cache.GetPost(ctx, id)
	if err != nil {
		log.Error("redis get error", slog.Any("err", err))
	}
	if post != nil {
		return post, nil
	}

	// 2. Достаём из Postgres
	post, err = u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Кладём в кеш
	_ = u.cache.SavePost(ctx, post, 5*time.Minute)

	return post, nil

}

func (u *PostUsecase) CreatePost(ctx context.Context, title, author, content string, tags []string) (*domain.Post, error) {
	post := &domain.Post{
		ID:        uuid.NewString(),
		Title:     title,
		Author:    author,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
	}

	// 1. Сохраняем в БД
	if err := u.repo.Save(ctx, post); err != nil {
		return nil, err
	}

	// 2. Публикуем событие в Kafka
	if err := u.producer.Publish(ctx, post); err != nil {
		return nil, err
	}

	// 3. Возращаем пост
	return post, nil
}
