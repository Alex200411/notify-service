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

type VendorConfigRepository interface {
	Create(ctx context.Context, req *model.CreateVendorRequest) (*model.VendorConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.VendorConfig, error)
	List(ctx context.Context) ([]model.VendorConfig, error)
	Update(ctx context.Context, id uuid.UUID, req *model.UpdateVendorRequest) (*model.VendorConfig, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type vendorConfigRepo struct {
	pool *pgxpool.Pool
}

func NewVendorConfigRepository(pool *pgxpool.Pool) VendorConfigRepository {
	return &vendorConfigRepo{pool: pool}
}

func (r *vendorConfigRepo) Create(ctx context.Context, req *model.CreateVendorRequest) (*model.VendorConfig, error) {
	headersJSON, err := json.Marshal(req.Headers)
	if err != nil {
		return nil, fmt.Errorf("marshal headers: %w", err)
	}

	method := req.Method
	if method == "" {
		method = "POST"
	}
	timeoutMs := req.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 5000
	}
	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5
	}

	var v model.VendorConfig
	var headersRaw []byte
	err = r.pool.QueryRow(ctx,
		`INSERT INTO vendor_configs (name, url, method, headers, body_template, timeout_ms, max_retries)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, url, method, headers, body_template, timeout_ms, max_retries, created_at, updated_at`,
		req.Name, req.URL, method, headersJSON, req.BodyTemplate, timeoutMs, maxRetries,
	).Scan(&v.ID, &v.Name, &v.URL, &v.Method, &headersRaw, &v.BodyTemplate, &v.TimeoutMs, &v.MaxRetries, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert vendor config: %w", err)
	}
	if err := json.Unmarshal(headersRaw, &v.Headers); err != nil {
		return nil, fmt.Errorf("unmarshal headers: %w", err)
	}
	return &v, nil
}

func (r *vendorConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.VendorConfig, error) {
	var v model.VendorConfig
	var headersRaw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, url, method, headers, body_template, timeout_ms, max_retries, created_at, updated_at
		 FROM vendor_configs WHERE id = $1`, id,
	).Scan(&v.ID, &v.Name, &v.URL, &v.Method, &headersRaw, &v.BodyTemplate, &v.TimeoutMs, &v.MaxRetries, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get vendor config: %w", err)
	}
	if err := json.Unmarshal(headersRaw, &v.Headers); err != nil {
		return nil, fmt.Errorf("unmarshal headers: %w", err)
	}
	return &v, nil
}

func (r *vendorConfigRepo) List(ctx context.Context) ([]model.VendorConfig, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, url, method, headers, body_template, timeout_ms, max_retries, created_at, updated_at
		 FROM vendor_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list vendor configs: %w", err)
	}
	defer rows.Close()

	var configs []model.VendorConfig
	for rows.Next() {
		var v model.VendorConfig
		var headersRaw []byte
		if err := rows.Scan(&v.ID, &v.Name, &v.URL, &v.Method, &headersRaw, &v.BodyTemplate, &v.TimeoutMs, &v.MaxRetries, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vendor config: %w", err)
		}
		if err := json.Unmarshal(headersRaw, &v.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}
		configs = append(configs, v)
	}
	return configs, nil
}

func (r *vendorConfigRepo) Update(ctx context.Context, id uuid.UUID, req *model.UpdateVendorRequest) (*model.VendorConfig, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.URL != nil {
		existing.URL = *req.URL
	}
	if req.Method != nil {
		existing.Method = *req.Method
	}
	if req.Headers != nil {
		existing.Headers = *req.Headers
	}
	if req.BodyTemplate != nil {
		existing.BodyTemplate = *req.BodyTemplate
	}
	if req.TimeoutMs != nil {
		existing.TimeoutMs = *req.TimeoutMs
	}
	if req.MaxRetries != nil {
		existing.MaxRetries = *req.MaxRetries
	}

	headersJSON, err := json.Marshal(existing.Headers)
	if err != nil {
		return nil, fmt.Errorf("marshal headers: %w", err)
	}

	var v model.VendorConfig
	var headersRaw []byte
	err = r.pool.QueryRow(ctx,
		`UPDATE vendor_configs
		 SET name=$1, url=$2, method=$3, headers=$4, body_template=$5, timeout_ms=$6, max_retries=$7, updated_at=$8
		 WHERE id=$9
		 RETURNING id, name, url, method, headers, body_template, timeout_ms, max_retries, created_at, updated_at`,
		existing.Name, existing.URL, existing.Method, headersJSON, existing.BodyTemplate, existing.TimeoutMs, existing.MaxRetries, time.Now(), id,
	).Scan(&v.ID, &v.Name, &v.URL, &v.Method, &headersRaw, &v.BodyTemplate, &v.TimeoutMs, &v.MaxRetries, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update vendor config: %w", err)
	}
	if err := json.Unmarshal(headersRaw, &v.Headers); err != nil {
		return nil, fmt.Errorf("unmarshal headers: %w", err)
	}
	return &v, nil
}

func (r *vendorConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM vendor_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete vendor config: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vendor config not found")
	}
	return nil
}
