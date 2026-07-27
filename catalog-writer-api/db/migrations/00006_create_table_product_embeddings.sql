-- +goose Up
-- +goose StatementBegin
CREATE TABLE product_embeddings (
    id             BIGSERIAL PRIMARY KEY,
    product_id     BIGINT NOT NULL,
    tenant_id      TEXT NOT NULL,           -- ANN pre-filter, never post-filter
    embedding_type TEXT NOT NULL,           -- 'image' | 'text_title' | 'text_desc' | 'text_review'
    embedding      vector(768) NOT NULL,
    source_ref     TEXT NOT NULL,           -- S3 key or text-field name this vector came from
    category_id    BIGINT,                  -- promoted for pre-filtering
    in_stock       BOOLEAN NOT NULL DEFAULT true,
    metadata       JSONB NOT NULL DEFAULT '{}',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per embedding, not per product (see article). This uniqueness
-- constraint is what makes worker's Upsert idempotent under Kafka's
-- at-least-once delivery.
CREATE UNIQUE INDEX idx_product_embeddings_identity
    ON product_embeddings(product_id, embedding_type, source_ref);

CREATE INDEX idx_product_embeddings_tenant ON product_embeddings(tenant_id);

-- HNSW is the default index per the article (best recall/latency
-- balance for most catalog sizes); switch to ivfflat only past ~10M
-- vectors where the compression tradeoff starts paying for itself.
CREATE INDEX idx_product_embeddings_hnsw
    ON product_embeddings USING hnsw (embedding vector_cosine_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE product_embeddings;
-- +goose StatementEnd
