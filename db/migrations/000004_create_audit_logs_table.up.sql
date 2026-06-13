CREATE TABLE audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id   UUID        NOT NULL REFERENCES customers(id),
    action        TEXT        NOT NULL,  -- 'created', 'updated', 'status_changed'
    old_value     JSONB,                 -- previous state (null for 'created')
    new_value     JSONB       NOT NULL,  -- new state
    changed_by    TEXT        NOT NULL DEFAULT 'system',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_customer_id ON audit_logs (customer_id, created_at DESC);