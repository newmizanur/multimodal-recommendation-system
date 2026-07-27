package repository

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multimodal-recsys/worker/internal/persistence/model"
)

// CatalogRepository is read-only by construction — the worker never
// writes to the catalog schema. See catalog-writer-api for the single
// writer.
type CatalogRepository interface {
	Get(ctx context.Context, tenantID, productID string) (model.ProductContent, error)
}

type pgCatalogRepository struct {
	db *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) CatalogRepository {
	return &pgCatalogRepository{db: db}
}

func (r *pgCatalogRepository) Get(ctx context.Context, tenantID, productID string) (model.ProductContent, error) {
	id, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		return model.ProductContent{}, err
	}

	var p model.ProductContent
	var categoryID sql.NullInt64

	err = r.db.QueryRow(ctx, `
		SELECT p.id, p.tenant_id, p.title, p.description, p.category_id, p.stock_qty > 0 AS in_stock,
		       array_remove(array_agg(pi.url ORDER BY pi.display_order), NULL) AS image_keys
		FROM products p
		LEFT JOIN product_images pi ON pi.product_id = p.id
		WHERE p.tenant_id = $1 AND p.id = $2
		GROUP BY p.id
	`, tenantID, id).Scan(&p.ID, &p.TenantID, &p.Title, &p.Description, &categoryID, &p.InStock, &p.ImageKeys)
	if err != nil {
		return model.ProductContent{}, err
	}

	if categoryID.Valid {
		p.CategoryID = strconv.FormatInt(categoryID.Int64, 10)
	}

	return p, nil
}
