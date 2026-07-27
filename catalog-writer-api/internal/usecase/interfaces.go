// Package usecase holds business logic. Controllers depend only on
// these interfaces, never on repository or messaging types directly —
// that's what keeps the HTTP layer thin and swappable (e.g. adding a
// gRPC delivery layer later touches nothing in here).
package usecase

import (
	"context"

	"github.com/multimodal-recsys/catalog-writer-api/internal/dto"
)

type ProductUsecase interface {
	Create(ctx context.Context, tenantID string, req dto.CreateProductRequest) (dto.ProductResponse, error)
	Update(ctx context.Context, tenantID string, id int64, req dto.UpdateProductRequest) (dto.ProductResponse, error)
	Delete(ctx context.Context, tenantID string, id int64) error
	PresignImageUpload(ctx context.Context, tenantID string, productID int64) (dto.PresignResponse, error)
}
