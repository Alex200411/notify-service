CREATE TABLE IF NOT EXISTS vendor_configs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(100) NOT NULL UNIQUE,
    url           TEXT NOT NULL,
    method        VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers       JSONB NOT NULL DEFAULT '{}',
    body_template TEXT NOT NULL DEFAULT '',
    timeout_ms    INT NOT NULL DEFAULT 5000,
    max_retries   INT NOT NULL DEFAULT 5,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id       UUID NOT NULL REFERENCES vendor_configs(id),
    payload         JSONB NOT NULL,
    idempotency_key VARCHAR(255),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts        INT NOT NULL DEFAULT 0,
    next_retry_at   TIMESTAMPTZ,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_next_retry ON notifications(status, next_retry_at) WHERE status = 'failed';
CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_idempotency ON notifications(vendor_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
