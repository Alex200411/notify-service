package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/a1234/notify-service/internal/model"
	"github.com/google/uuid"
)

type mockNotifRepo struct {
	mu      sync.Mutex
	notifs  map[uuid.UUID]*model.Notification
	created []*model.Notification
}

func newMockNotifRepo() *mockNotifRepo {
	return &mockNotifRepo{notifs: make(map[uuid.UUID]*model.Notification)}
}

func (m *mockNotifRepo) Create(ctx context.Context, req *model.CreateNotificationRequest) (*model.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := &model.Notification{
		ID:             uuid.New(),
		VendorID:       req.VendorID,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		Status:         model.StatusPending,
		Attempts:       0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	m.notifs[n.ID] = n
	m.created = append(m.created, n)
	return n, nil
}

func (m *mockNotifRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notifs[id]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	copy := *n
	return &copy, nil
}

func (m *mockNotifRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, attempts int, lastError *string, nextRetryAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.notifs[id]
	n.Status = status
	n.Attempts = attempts
	n.LastError = lastError
	n.NextRetryAt = nextRetryAt
	n.UpdatedAt = time.Now()
	return nil
}

func (m *mockNotifRepo) FindRetryable(ctx context.Context, limit int) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockNotifRepo) FindStale(ctx context.Context, limit int) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockNotifRepo) getStatus(id uuid.UUID) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifs[id].Status
}

type mockVendorRepo struct {
	vendors map[uuid.UUID]*model.VendorConfig
}

func (m *mockVendorRepo) Create(ctx context.Context, req *model.CreateVendorRequest) (*model.VendorConfig, error) {
	return nil, nil
}
func (m *mockVendorRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.VendorConfig, error) {
	v, ok := m.vendors[id]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return v, nil
}
func (m *mockVendorRepo) List(ctx context.Context) ([]model.VendorConfig, error) {
	return nil, nil
}
func (m *mockVendorRepo) Update(ctx context.Context, id uuid.UUID, req *model.UpdateVendorRequest) (*model.VendorConfig, error) {
	return nil, nil
}
func (m *mockVendorRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockQueue struct {
	mu    sync.Mutex
	items []string
}

func (m *mockQueue) Enqueue(ctx context.Context, notificationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, notificationID)
	return nil
}

func (m *mockQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 {
		return "", nil
	}
	item := m.items[0]
	m.items = m.items[1:]
	return item, nil
}

func TestDeliverOne_Success(t *testing.T) {
	var receivedBody string
	var receivedHeaders http.Header
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	vendorID := uuid.New()
	vendorRepo := &mockVendorRepo{
		vendors: map[uuid.UUID]*model.VendorConfig{
			vendorID: {
				ID:           vendorID,
				Name:         "test_vendor",
				URL:          mockServer.URL,
				Method:       "POST",
				Headers:      map[string]string{"Content-Type": "application/json", "X-Api-Key": "secret123"},
				BodyTemplate: `{"uid": "{{.user_id}}", "action": "{{.event}}"}`,
				TimeoutMs:    5000,
				MaxRetries:   3,
			},
		},
	}

	notifRepo := newMockNotifRepo()
	q := &mockQueue{}

	notifSvc := NewNotificationService(notifRepo, q)
	n, err := notifSvc.Create(context.Background(), &model.CreateNotificationRequest{
		VendorID: vendorID,
		Payload:  map[string]interface{}{"user_id": "u123", "event": "signup"},
	})
	if err != nil {
		t.Fatalf("create notification: %v", err)
	}

	deliverySvc := NewDeliveryService(notifRepo, vendorRepo, q)
	if err := deliverySvc.DeliverOne(context.Background(), n.ID.String()); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	if notifRepo.getStatus(n.ID) != model.StatusDelivered {
		t.Errorf("expected status %s, got %s", model.StatusDelivered, notifRepo.getStatus(n.ID))
	}

	expectedBody := `{"uid": "u123", "action": "signup"}`
	if receivedBody != expectedBody {
		t.Errorf("body mismatch:\n  got:  %s\n  want: %s", receivedBody, expectedBody)
	}

	if receivedHeaders.Get("X-Api-Key") != "secret123" {
		t.Errorf("expected X-Api-Key header, got %q", receivedHeaders.Get("X-Api-Key"))
	}
}

func TestDeliverOne_VendorFailureThenRetry(t *testing.T) {
	var attempts int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	vendorID := uuid.New()
	vendorRepo := &mockVendorRepo{
		vendors: map[uuid.UUID]*model.VendorConfig{
			vendorID: {
				ID:         vendorID,
				Name:       "flaky_vendor",
				URL:        mockServer.URL,
				Method:     "POST",
				Headers:    map[string]string{"Content-Type": "application/json"},
				TimeoutMs:  5000,
				MaxRetries: 5,
			},
		},
	}

	notifRepo := newMockNotifRepo()
	q := &mockQueue{}

	notifSvc := NewNotificationService(notifRepo, q)
	n, err := notifSvc.Create(context.Background(), &model.CreateNotificationRequest{
		VendorID: vendorID,
		Payload:  map[string]interface{}{"test": true},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	deliverySvc := NewDeliveryService(notifRepo, vendorRepo, q)

	// First attempt — should fail
	deliverySvc.DeliverOne(context.Background(), n.ID.String())
	if notifRepo.getStatus(n.ID) != model.StatusFailed {
		t.Fatalf("expected failed, got %s", notifRepo.getStatus(n.ID))
	}

	// Second attempt — should fail again
	deliverySvc.DeliverOne(context.Background(), n.ID.String())
	if notifRepo.getStatus(n.ID) != model.StatusFailed {
		t.Fatalf("expected failed, got %s", notifRepo.getStatus(n.ID))
	}

	// Third attempt — should succeed
	deliverySvc.DeliverOne(context.Background(), n.ID.String())
	if notifRepo.getStatus(n.ID) != model.StatusDelivered {
		t.Errorf("expected delivered after retry, got %s", notifRepo.getStatus(n.ID))
	}
}

func TestDeliverOne_DeadLetter(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	vendorID := uuid.New()
	vendorRepo := &mockVendorRepo{
		vendors: map[uuid.UUID]*model.VendorConfig{
			vendorID: {
				ID:         vendorID,
				Name:       "dead_vendor",
				URL:        mockServer.URL,
				Method:     "POST",
				Headers:    map[string]string{},
				TimeoutMs:  5000,
				MaxRetries: 2,
			},
		},
	}

	notifRepo := newMockNotifRepo()
	q := &mockQueue{}
	notifSvc := NewNotificationService(notifRepo, q)
	n, _ := notifSvc.Create(context.Background(), &model.CreateNotificationRequest{
		VendorID: vendorID,
		Payload:  map[string]interface{}{"test": true},
	})

	deliverySvc := NewDeliveryService(notifRepo, vendorRepo, q)

	// Exhaust all retries
	for i := 0; i < 3; i++ {
		deliverySvc.DeliverOne(context.Background(), n.ID.String())
	}

	if notifRepo.getStatus(n.ID) != model.StatusDeadLetter {
		t.Errorf("expected dead_letter, got %s", notifRepo.getStatus(n.ID))
	}
}

func TestDeliverOne_FallbackPayloadWhenNoTemplate(t *testing.T) {
	var receivedBody string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	vendorID := uuid.New()
	vendorRepo := &mockVendorRepo{
		vendors: map[uuid.UUID]*model.VendorConfig{
			vendorID: {
				ID:        vendorID,
				Name:      "raw_vendor",
				URL:       mockServer.URL,
				Method:    "POST",
				Headers:   map[string]string{"Content-Type": "application/json"},
				TimeoutMs: 5000,
				MaxRetries: 3,
			},
		},
	}

	notifRepo := newMockNotifRepo()
	q := &mockQueue{}
	notifSvc := NewNotificationService(notifRepo, q)
	payload := map[string]interface{}{"key": "value", "num": float64(42)}
	n, _ := notifSvc.Create(context.Background(), &model.CreateNotificationRequest{
		VendorID: vendorID,
		Payload:  payload,
	})

	deliverySvc := NewDeliveryService(notifRepo, vendorRepo, q)
	deliverySvc.DeliverOne(context.Background(), n.ID.String())

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(receivedBody), &parsed); err != nil {
		t.Fatalf("received body is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed["key"])
	}
}
