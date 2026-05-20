package usecase

import (
	"context"
	"log/slog"

	"notification-service/internal/domain"
)

type NotificationStore interface {
	SaveNotificationOnce(ctx context.Context, eventID string, notification domain.Notification) (bool, error)
}

type NotificationUsecase struct {
	store NotificationStore
	log   *slog.Logger
}

func NewNotificationUsecase(store NotificationStore, log *slog.Logger) *NotificationUsecase {
	return &NotificationUsecase{
		store: store,
		log:   log,
	}
}

func (u *NotificationUsecase) Process(ctx context.Context, event domain.EventEnvelope) (bool, error) {
	notification := domain.Notification{
		EventID:   event.EventID,
		PostID:    event.Payload.ID,
		Author:    event.Payload.Author,
		Title:     event.Payload.Title,
		CreatedAt: event.Payload.CreatedAt,
	}

	created, err := u.store.SaveNotificationOnce(ctx, event.EventID, notification)
	if err != nil {
		return false, err
	}

	return created, nil
}
