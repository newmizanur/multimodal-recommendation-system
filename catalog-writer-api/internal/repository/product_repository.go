// Package repository is the only layer allowed to hold SQL. Usecases
// depend on the ProductRepository interface, not on *pgxpool.Pool
// directly, so business logic can be unit tested against a fake.
package repository

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multimodal-recsys/catalog-writer-api/internal/persistence/model"
)

type ProductRepository interface {
	Create(ctx context.Context, p model.Product) (model.Product, error)
	Update(ctx context.Context, p model.Product) error
	Get(ctx context.Context, tenantID string, id int64) (model.Product, error)
	Delete(ctx context.Context, tenantID string, id int64) error
}

type pgProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) ProductRepository {
	return &pgProductRepository{db: db}
}

func (r *pgProductRepository) Create(ctx context.Context, p model.Product) (model.Product, error) {
	categoryID, err := strconv.ParseInt(p.CategoryID, 10, 64)
	if err != nil {
		return model.Product{}, err
	}

	attributes, err := json.Marshal(p.Attributes)
	if err != nil {
		return model.Product{}, err
	}

	err = r.db.QueryRow(ctx, `
		INSERT INTO products (tenant_id, sku, title, description, category_id, price, currency, stock_qty, status, attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, updated_at
	`, p.TenantID, p.SKU, p.Title, p.Description, categoryID, p.Price, p.Currency, p.StockQty, p.Status, attributes,
	).Scan(&p.ID, &p.UpdatedAt)
	if err != nil {
		return model.Product{}, err
	}

	return p, nil
}

func (r *pgProductRepository) Update(ctx context.Context, p model.Product) error {
	attributes, err := json.Marshal(p.Attributes)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE products
		SET title = $1, description = $2, price = $3, stock_qty = $4, attributes = $5, updated_at = now()
		WHERE tenant_id = $6 AND id = $7
	`, p.Title, p.Description, p.Price, p.StockQty, attributes, p.TenantID, p.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *pgProductRepository) Get(ctx context.Context, tenantID string, id int64) (model.Product, error) {
	var p model.Product
	var categoryID int64
	var attributes []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, sku, title, description, category_id, price, currency, stock_qty, status, attributes, updated_at
		FROM products
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Title, &p.Description, &categoryID,
		&p.Price, &p.Currency, &p.StockQty, &p.Status, &attributes, &p.UpdatedAt,
	)
	if err != nil {
		return model.Product{}, err
	}

	if err := json.Unmarshal(attributes, &p.Attributes); err != nil {
		return model.Product{}, err
	}
	p.CategoryID = strconv.FormatInt(categoryID, 10)

	return p, nil
}

func (r *pgProductRepository) Delete(ctx context.Context, tenantID string, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE products SET status = 'deleted', updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	return err
}
