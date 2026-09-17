package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a1234/notify-service/internal/model"
	"github.com/a1234/notify-service/internal/queue"
	"github.com/a1234/notify-service/internal/repository"
	"github.com/google/uuid"
)

type NotificationService struct {
	repo  repository.NotificationRepository
	queue queue.Queue
}

func NewNotificationService(repo repository.NotificationRepository, q queue.Queue) *NotificationService {
	return &NotificationService{repo: repo, queue: q}
}

func (s *NotificationService) Create(ctx context.Context, req *model.CreateNotificationRequest) (*model.Notification, error) {
	n, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("persist notification: %w", err)
	}

	if err := s.queue.Enqueue(ctx, n.ID.String()); err != nil {
		slog.Error("failed to enqueue notification, recovery sweep will pick it up",
			"notification_id", n.ID, "error", err)
	}

	return n, nil
}

func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return s.repo.GetByID(ctx, id)
}
