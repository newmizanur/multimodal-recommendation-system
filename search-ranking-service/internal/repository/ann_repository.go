// Package repository holds the two stateful dependencies this service
// owns directly: the ANN/vector index and the Redis caches. Outbound
// calls to other *services* (embedding server, cross-encoder) live in
// internal/client instead — same "our data" vs "someone else's service"
// split as catalog-reader-api.
package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// AnnRepository wraps whichever ANN backend you chose (Milvus / Qdrant /
// pgvector). HNSW is the default index; IVF-PQ past ~10M vectors.
type AnnRepository interface {
	Search(ctx context.Context, p AnnSearchParams) ([]AnnCandidate, error)
}

type AnnSearchParams struct {
	TenantID string
	Vector   []float32
	TopK     int
	Filters  map[string]any
}

type AnnCandidate struct {
	ProductID string
	Score     float64
}

type annRepository struct {
	db *pgxpool.Pool
}

func NewAnnRepository(db *pgxpool.Pool) AnnRepository {
	return &annRepository{db: db}
}

// Search MUST apply the tenant_id filter (and any other pre-filters) at
// the index level, not after retrieval — post-filtering risks
// cross-tenant leakage and wastes retrieval budget on candidates you'd
// discard anyway.
func (r *annRepository) Search(ctx context.Context, p AnnSearchParams) ([]AnnCandidate, error) {
	query := `
		SELECT product_id, 1 - (embedding <=> $1) AS score
		FROM product_embeddings
		WHERE tenant_id = $2
	`
	args := []any{pgvector.NewVector(p.Vector), p.TenantID}

	if v, ok := p.Filters["category_id"]; ok {
		categoryID, err := toInt64(v)
		if err != nil {
			return nil, err
		}
		args = append(args, categoryID)
		query += fmt.Sprintf(" AND category_id = $%d", len(args))
	}

	if _, ok := p.Filters["in_stock"]; ok {
		query += " AND in_stock = true"
	}

	args = append(args, p.TopK)
	query += fmt.Sprintf(" ORDER BY embedding <=> $1 LIMIT $%d", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []AnnCandidate
	for rows.Next() {
		var productID int64
		var c AnnCandidate
		if err := rows.Scan(&productID, &c.Score); err != nil {
			return nil, err
		}
		c.ProductID = strconv.FormatInt(productID, 10)
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case string:
		return strconv.ParseInt(n, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported filter value type %T", v)
	}
}
