package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDelivered  = "delivered"
	StatusFailed     = "failed"
	StatusDeadLetter = "dead_letter"
)

type VendorConfig struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	BodyTemplate string            `json:"body_template"`
	TimeoutMs    int               `json:"timeout_ms"`
	MaxRetries   int               `json:"max_retries"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type Notification struct {
	ID             uuid.UUID              `json:"id"`
	VendorID       uuid.UUID              `json:"vendor_id"`
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey *string                `json:"idempotency_key,omitempty"`
	Status         string                 `json:"status"`
	Attempts       int                    `json:"attempts"`
	NextRetryAt    *time.Time             `json:"next_retry_at,omitempty"`
	LastError      *string                `json:"last_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type CreateNotificationRequest struct {
	VendorID       uuid.UUID              `json:"vendor_id"`
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey *string                `json:"idempotency_key,omitempty"`
}

type CreateVendorRequest struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyTemplate string            `json:"body_template,omitempty"`
	TimeoutMs    int               `json:"timeout_ms,omitempty"`
	MaxRetries   int               `json:"max_retries,omitempty"`
}

type UpdateVendorRequest struct {
	Name         *string            `json:"name,omitempty"`
	URL          *string            `json:"url,omitempty"`
	Method       *string            `json:"method,omitempty"`
	Headers      *map[string]string `json:"headers,omitempty"`
	BodyTemplate *string            `json:"body_template,omitempty"`
	TimeoutMs    *int               `json:"timeout_ms,omitempty"`
	MaxRetries   *int               `json:"max_retries,omitempty"`
}
