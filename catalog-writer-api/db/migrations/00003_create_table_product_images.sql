-- +goose Up
-- +goose StatementBegin
CREATE TABLE product_images (
    id            BIGSERIAL PRIMARY KEY,
    product_id    BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url           TEXT NOT NULL, -- S3-compatible object key/URL
    display_order INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_product_images_product ON product_images(product_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE product_images;
-- +goose StatementEnd
