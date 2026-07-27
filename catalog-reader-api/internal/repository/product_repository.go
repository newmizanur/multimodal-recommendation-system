package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multimodal-recsys/catalog-reader-api/internal/persistence/model"
)

// ProductRepository is read-only by construction: no Create/Update/Delete
// methods exist here at all, so a future usecase literally cannot write
// through this service by accident — see catalog-writer-api for writes.
type ProductRepository interface {
	Get(ctx context.Context, tenantID string, id int64) (model.Product, error)
	GetMany(ctx context.Context, tenantID string, ids []string) ([]model.Product, error) // used to enrich search results
}

type pgProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) ProductRepository {
	return &pgProductRepository{db: db}
}

func (r *pgProductRepository) Get(ctx context.Context, tenantID string, id int64) (model.Product, error) {
	var p model.Product

	err := r.db.QueryRow(ctx, `
		SELECT id, sku, title, price, currency, stock_qty > 0 AS in_stock
		FROM products
		WHERE tenant_id = $1 AND id = $2 AND status = 'active'
	`, tenantID, id).Scan(&p.ID, &p.SKU, &p.Title, &p.Price, &p.Currency, &p.InStock)
	if err != nil {
		return model.Product{}, err
	}

	return p, nil
}

func (r *pgProductRepository) GetMany(ctx context.Context, tenantID string, ids []string) ([]model.Product, error) {
	intIDs := make([]int64, len(ids))
	for i, id := range ids {
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		intIDs[i] = parsed
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, sku, title, price, currency, stock_qty > 0 AS in_stock
		FROM products
		WHERE tenant_id = $1 AND id = ANY($2) AND status = 'active'
	`, tenantID, intIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Title, &p.Price, &p.Currency, &p.InStock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
