-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
    id        BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name      TEXT NOT NULL,
    parent_id BIGINT REFERENCES categories(id)
);
CREATE INDEX idx_categories_tenant ON categories(tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE categories;
-- +goose StatementEnd
