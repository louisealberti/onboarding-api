CREATE TABLE idempotency_keys (
    key         TEXT        PRIMARY KEY,
    status_code INT         NOT NULL,
    response    JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys (expires_at);