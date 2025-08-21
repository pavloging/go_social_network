// internal/usecase/post_usecase.go
package usecase

import (
	"post-service/internal/domain"
	"post-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type PostUsecase struct {
	Producer *repository.KafkaProducer
}

func NewPostUsecase(producer *repository.KafkaProducer) *PostUsecase {
	return &PostUsecase{Producer: producer}
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

	if err := u.Producer.Publish(post); err != nil {
		return nil, err
	}

	return post, nil
}
