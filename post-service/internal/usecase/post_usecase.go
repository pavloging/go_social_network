package usecase

import (
	"post-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type PostUsecase struct {
	repo     domain.PostRepository
	producer domain.EventProducer
}

func NewPostUsecase(poolRepo domain.PostRepository, producer domain.EventProducer) *PostUsecase {
	return &PostUsecase{
		repo:     poolRepo,
		producer: producer,
	}
}

func (u *PostUsecase) CreatePost(title, author, content string, tags []string) (*domain.Post, error) {
	post := &domain.Post{
		ID:        uuid.NewString(),
		Title:     title,
		Author:    author,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
	}

	// 1. Сохраняем в БД
	if err := u.repo.Save(post); err != nil {
		return nil, err
	}

	// 2. Публикуем событие в Kafka
	if err := u.producer.Publish(post); err != nil {
		return nil, err
	}

	// 3. Возращаем пост
	return post, nil
}
