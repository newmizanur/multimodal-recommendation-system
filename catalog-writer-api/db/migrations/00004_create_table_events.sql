-- +goose Up
-- +goose StatementBegin
CREATE TABLE events (
    tenant_id  TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    product_id BIGINT NOT NULL,
    event_type TEXT NOT NULL, -- click | purchase | ...
    ts         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Always query this table by (tenant_id, user_id), never user_id alone —
-- see article: composite tenant scoping for personalisation signals.
CREATE INDEX idx_events_tenant_user ON events(tenant_id, user_id);
CREATE INDEX idx_events_tenant_product ON events(tenant_id, product_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd
