package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/a1234/notify-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository interface {
	Create(ctx context.Context, req *model.CreateNotificationRequest) (*model.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, attempts int, lastError *string, nextRetryAt *time.Time) error
	FindRetryable(ctx context.Context, limit int) ([]uuid.UUID, error)
	FindStale(ctx context.Context, limit int) ([]uuid.UUID, error)
}

type notificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &notificationRepo{pool: pool}
}

func (r *notificationRepo) Create(ctx context.Context, req *model.CreateNotificationRequest) (*model.Notification, error) {
	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var n model.Notification
	var payloadRaw []byte
	err = r.pool.QueryRow(ctx,
		`INSERT INTO notifications (vendor_id, payload, idempotency_key)
		 VALUES ($1, $2, $3)
		 RETURNING id, vendor_id, payload, idempotency_key, status, attempts, next_retry_at, last_error, created_at, updated_at`,
		req.VendorID, payloadJSON, req.IdempotencyKey,
	).Scan(&n.ID, &n.VendorID, &payloadRaw, &n.IdempotencyKey, &n.Status, &n.Attempts, &n.NextRetryAt, &n.LastError, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert notification: %w", err)
	}
	if err := json.Unmarshal(payloadRaw, &n.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return &n, nil
}

func (r *notificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	var n model.Notification
	var payloadRaw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, vendor_id, payload, idempotency_key, status, attempts, next_retry_at, last_error, created_at, updated_at
		 FROM notifications WHERE id = $1`, id,
	).Scan(&n.ID, &n.VendorID, &payloadRaw, &n.IdempotencyKey, &n.Status, &n.Attempts, &n.NextRetryAt, &n.LastError, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get notification: %w", err)
	}
	if err := json.Unmarshal(payloadRaw, &n.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return &n, nil
}

func (r *notificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, attempts int, lastError *string, nextRetryAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status=$1, attempts=$2, last_error=$3, next_retry_at=$4, updated_at=now() WHERE id=$5`,
		status, attempts, lastError, nextRetryAt, id,
	)
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}
	return nil
}

func (r *notificationRepo) FindRetryable(ctx context.Context, limit int) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM notifications
		 WHERE status = 'failed' AND next_retry_at <= now()
		 ORDER BY next_retry_at
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find retryable: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan retryable id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *notificationRepo) FindStale(ctx context.Context, limit int) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM notifications
		 WHERE (status = 'pending' AND created_at < now() - interval '1 minute')
		    OR (status = 'processing' AND updated_at < now() - interval '2 minutes')
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find stale: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan stale id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
