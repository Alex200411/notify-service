package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/a1234/notify-service/internal/model"
	"github.com/a1234/notify-service/internal/queue"
	"github.com/a1234/notify-service/internal/repository"
	tmpl "github.com/a1234/notify-service/internal/template"
	"github.com/google/uuid"
)

type DeliveryService struct {
	notifRepo  repository.NotificationRepository
	vendorRepo repository.VendorConfigRepository
	queue      queue.Queue
	httpClient *http.Client
}

func NewDeliveryService(
	notifRepo repository.NotificationRepository,
	vendorRepo repository.VendorConfigRepository,
	q queue.Queue,
) *DeliveryService {
	return &DeliveryService{
		notifRepo:  notifRepo,
		vendorRepo: vendorRepo,
		queue:      q,
		httpClient: &http.Client{},
	}
}

func (s *DeliveryService) DeliverOne(ctx context.Context, notificationID string) error {
	id, err := uuid.Parse(notificationID)
	if err != nil {
		return fmt.Errorf("parse notification id: %w", err)
	}

	notif, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get notification: %w", err)
	}

	if notif.Status == model.StatusDelivered || notif.Status == model.StatusDeadLetter {
		return nil
	}

	vendor, err := s.vendorRepo.GetByID(ctx, notif.VendorID)
	if err != nil {
		return fmt.Errorf("get vendor config: %w", err)
	}

	attempts := notif.Attempts + 1
	if err := s.notifRepo.UpdateStatus(ctx, id, model.StatusProcessing, attempts, nil, nil); err != nil {
		return fmt.Errorf("set processing: %w", err)
	}

	deliveryErr := s.sendHTTPRequest(ctx, vendor, notif)

	if deliveryErr == nil {
		if err := s.notifRepo.UpdateStatus(ctx, id, model.StatusDelivered, attempts, nil, nil); err != nil {
			slog.Error("failed to mark delivered", "notification_id", id, "error", err)
		}
		slog.Info("notification delivered", "notification_id", id, "vendor", vendor.Name, "attempts", attempts)
		return nil
	}

	errMsg := deliveryErr.Error()
	slog.Warn("delivery failed", "notification_id", id, "vendor", vendor.Name, "attempts", attempts, "error", errMsg)

	if attempts >= vendor.MaxRetries {
		if err := s.notifRepo.UpdateStatus(ctx, id, model.StatusDeadLetter, attempts, &errMsg, nil); err != nil {
			slog.Error("failed to mark dead letter", "notification_id", id, "error", err)
		}
		slog.Error("notification moved to dead letter", "notification_id", id, "vendor", vendor.Name, "attempts", attempts)
		return nil
	}

	nextRetry := time.Now().Add(backoff(attempts))
	if err := s.notifRepo.UpdateStatus(ctx, id, model.StatusFailed, attempts, &errMsg, &nextRetry); err != nil {
		slog.Error("failed to mark failed", "notification_id", id, "error", err)
	}
	return nil
}

func (s *DeliveryService) sendHTTPRequest(ctx context.Context, vendor *model.VendorConfig, notif *model.Notification) error {
	var bodyStr string
	if vendor.BodyTemplate != "" {
		var err error
		bodyStr, err = tmpl.RenderBody(vendor.BodyTemplate, notif.Payload)
		if err != nil {
			return fmt.Errorf("render body: %w", err)
		}
	} else {
		raw, err := json.Marshal(notif.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		bodyStr = string(raw)
	}

	timeout := time.Duration(vendor.TimeoutMs) * time.Millisecond
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, vendor.Method, vendor.URL, strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	for k, v := range vendor.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("vendor returned status %d", resp.StatusCode)
}

func (s *DeliveryService) RunDeliveryLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		notifID, err := s.queue.Dequeue(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("dequeue error", "error", err)
			time.Sleep(time.Second)
			continue
		}
		if notifID == "" {
			continue
		}

		if err := s.DeliverOne(ctx, notifID); err != nil {
			slog.Error("deliver error", "notification_id", notifID, "error", err)
		}
	}
}

func (s *DeliveryService) RunRetrySweep(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := s.notifRepo.FindRetryable(ctx, 50)
			if err != nil {
				slog.Error("retry sweep query error", "error", err)
				continue
			}
			for _, id := range ids {
				if err := s.queue.Enqueue(ctx, id.String()); err != nil {
					slog.Error("retry sweep enqueue error", "notification_id", id, "error", err)
				}
			}
			if len(ids) > 0 {
				slog.Info("retry sweep enqueued", "count", len(ids))
			}
		}
	}
}

func (s *DeliveryService) RunRecoverySweep(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := s.notifRepo.FindStale(ctx, 50)
			if err != nil {
				slog.Error("recovery sweep query error", "error", err)
				continue
			}
			for _, id := range ids {
				if err := s.queue.Enqueue(ctx, id.String()); err != nil {
					slog.Error("recovery sweep enqueue error", "notification_id", id, "error", err)
				}
			}
			if len(ids) > 0 {
				slog.Info("recovery sweep enqueued", "count", len(ids))
			}
		}
	}
}

func backoff(attempt int) time.Duration {
	base := 2 * time.Second
	maxDelay := 10 * time.Minute
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(delay) / 5))
	return delay + jitter
}
