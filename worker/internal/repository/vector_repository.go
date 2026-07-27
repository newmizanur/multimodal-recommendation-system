package repository

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// VectorRepository wraps whichever vector DB you pick (Milvus / Qdrant /
// pgvector). Swap the implementation, not the call sites in the ingest
// usecase.
type VectorRepository interface {
	// Upsert must be idempotent, keyed on (product_id, embedding_type,
	// source_ref) — a message can be reprocessed after a crash between
	// handling it and committing the Kafka offset.
	Upsert(ctx context.Context, v Vector) error

	// HardDeleteByProduct removes ALL vector rows for a product. Never
	// soft-delete: a soft-deleted row still costs HNSW traversal time
	// and silently shrinks result counts.
	HardDeleteByProduct(ctx context.Context, tenantID, productID string) error
}

type Vector struct {
	ProductID     string
	TenantID      string
	EmbeddingType string // "image" | "text_title" | "text_desc" | "text_review"
	Values        []float32
	SourceRef     string
	CategoryID    string
	InStock       bool
	Metadata      map[string]any
}

type vectorRepository struct {
	db *pgxpool.Pool
}

func NewVectorRepository(db *pgxpool.Pool) VectorRepository {
	return &vectorRepository{db: db}
}

func (r *vectorRepository) Upsert(ctx context.Context, v Vector) error {
	productID, err := strconv.ParseInt(v.ProductID, 10, 64)
	if err != nil {
		return err
	}

	var categoryID *int64
	if v.CategoryID != "" {
		parsed, err := strconv.ParseInt(v.CategoryID, 10, 64)
		if err != nil {
			return err
		}
		categoryID = &parsed
	}

	metadata, err := json.Marshal(v.Metadata)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO product_embeddings (product_id, tenant_id, embedding_type, embedding, source_ref, category_id, in_stock, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (product_id, embedding_type, source_ref)
		DO UPDATE SET embedding = EXCLUDED.embedding, category_id = EXCLUDED.category_id,
		              in_stock = EXCLUDED.in_stock, metadata = EXCLUDED.metadata, updated_at = now()
	`, productID, v.TenantID, v.EmbeddingType, pgvector.NewVector(v.Values), v.SourceRef, categoryID, v.InStock, metadata)
	return err
}

func (r *vectorRepository) HardDeleteByProduct(ctx context.Context, tenantID, productID string) error {
	id, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM product_embeddings WHERE tenant_id = $1 AND product_id = $2`, tenantID, id)
	return err
}
