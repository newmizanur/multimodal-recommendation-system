package usecase

import (
	"context"
	"encoding/json"
	"log"

	"github.com/multimodal-recsys/worker/internal/client"
	"github.com/multimodal-recsys/worker/internal/repository"
	"github.com/multimodal-recsys/worker/internal/storage"
)

type event struct {
	Type      string `json:"type"` // product.created | product.updated | product.deleted | product.reembed
	TenantID  string `json:"tenant_id"`
	ProductID string `json:"product_id"`
}

type ingestUsecase struct {
	catalog repository.CatalogRepository
	vectors repository.VectorRepository
	images  *storage.S3Reader
	embed   *client.EmbedClient
}

func NewIngestUsecase(catalog repository.CatalogRepository, vectors repository.VectorRepository, images *storage.S3Reader, embed *client.EmbedClient) IngestUsecase {
	return &ingestUsecase{catalog: catalog, vectors: vectors, images: images, embed: embed}
}

// HandleEvent is idempotent: reprocessing the same event twice (a
// redelivery after a crash before the Kafka offset commits) must be
// safe — vector writes are upserts keyed on (product_id, embedding_type,
// source_ref).
func (u *ingestUsecase) HandleEvent(ctx context.Context, body []byte) error {
	var ev event
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}

	switch ev.Type {
	case "product.deleted":
		// Hard delete — soft delete wastes HNSW traversal and silently
		// degrades result counts.
		return u.vectors.HardDeleteByProduct(ctx, ev.TenantID, ev.ProductID)

	case "product.created", "product.updated", "product.reembed":
		return u.embedAndUpsert(ctx, ev.TenantID, ev.ProductID)

	default:
		log.Printf("unknown event type: %s", ev.Type)
		return nil
	}
}

func (u *ingestUsecase) embedAndUpsert(ctx context.Context, tenantID, productID string) error {
	content, err := u.catalog.Get(ctx, tenantID, productID)
	if err != nil {
		return err
	}

	// One row per embedding, not per product — a product with five
	// photos and three text fields produces eight vector rows.
	for _, key := range content.ImageKeys {
		bytes, err := u.images.ReadObject(ctx, key)
		if err != nil {
			return err
		}
		resp, err := u.embed.Embed(client.EmbedRequest{
			Kind:        "image",
			Prompt:      "catalog_image",
			ImageBase64: encodeBase64(bytes),
		})
		if err != nil {
			return err
		}
		if err := u.vectors.Upsert(ctx, repository.Vector{
			ProductID: productID, TenantID: tenantID, EmbeddingType: "image",
			Values: resp.Vector, SourceRef: key, CategoryID: content.CategoryID, InStock: content.InStock,
		}); err != nil {
			return err
		}
	}

	textFields := map[string]string{
		"text_title": content.Title,
		"text_desc":  content.Description,
	}
	for embeddingType, text := range textFields {
		if text == "" {
			continue
		}
		resp, err := u.embed.Embed(client.EmbedRequest{Kind: "text", Prompt: "catalog_text", Text: text})
		if err != nil {
			return err
		}
		if err := u.vectors.Upsert(ctx, repository.Vector{
			ProductID: productID, TenantID: tenantID, EmbeddingType: embeddingType,
			Values: resp.Vector, SourceRef: embeddingType, CategoryID: content.CategoryID, InStock: content.InStock,
		}); err != nil {
			return err
		}
	}

	return nil
}
