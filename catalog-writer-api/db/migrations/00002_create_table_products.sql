-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    sku         TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category_id BIGINT REFERENCES categories(id),
    price       NUMERIC(12,2) NOT NULL,
    currency    CHAR(3) NOT NULL,
    stock_qty   INT NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active', -- active | deleted
    attributes  JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every read/write path in this system scopes by tenant first — this
-- index is load-bearing, not optional.
CREATE INDEX idx_products_tenant ON products(tenant_id);
CREATE UNIQUE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_products_attributes_gin ON products USING GIN (attributes);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE products;
-- +goose StatementEnd
