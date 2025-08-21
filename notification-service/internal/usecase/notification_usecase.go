package usecase

import (
	"context"
	"fmt"
	"notification-service/internal/domain"
)

type NotificationUsecase struct{}

func NewNotificationUsecase() *NotificationUsecase {
	return &NotificationUsecase{}
}

// Обработка входящего события
func (u *NotificationUsecase) Process(ctx context.Context, n domain.Notification) error {
	// здесь бизнес-логика (например, логирование или отправка e-mail)
	// сейчас просто выводим
	// println("Got notification for post:", n.PostID, "title:", n.Title)
	fmt.Printf("Notification: %+v\n", n)
	return nil
}
