package usecase

import (
	"context"

	"github.com/multimodal-recsys/catalog-reader-api/internal/dto"
)

type ProductUsecase interface {
	Get(ctx context.Context, tenantID string, id int64) (dto.ProductResponse, error)
	Search(ctx context.Context, tenantID string, req dto.SearchRequest) ([]dto.SearchResultItem, error)
}
