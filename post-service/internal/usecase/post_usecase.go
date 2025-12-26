package usecase

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}

	return posts, nil
}

func (u *PostUsecase) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	// 1. Проверяем кеш (игнорируем ошибку, но логгируем)
	// Кэш не должен ломать логику приложения
	post, _ := u.cache.GetPost(ctx, id)
	if post != nil {
		return post, nil
	}

	// 2. Достаём из Postgres
	post, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get post by id %s: %w", id, err)
	}

	// 3. Кладём в кеш (асинхронно, чтобы не блокировать ответ)
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
		CreatedAt: time.Now(),
	}

	// 1. Сохраняем в БД
	if err := u.repo.Save(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to save post to database: %w", err)
	}

	// 2. Публикуем событие в Kafka
	if err := u.producer.Publish(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to publish post to kafka: %w", err)
	}

	// 3. Асинхронно сохраняем в кеш
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = u.cache.SavePost(cacheCtx, post, 5*time.Minute)
	}()

	// 3. Возвращаем пост
	return post, nil
}
