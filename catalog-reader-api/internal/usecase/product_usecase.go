package usecase

import (
	"context"

	"github.com/multimodal-recsys/catalog-reader-api/internal/apperror"
	"github.com/multimodal-recsys/catalog-reader-api/internal/client"
	"github.com/multimodal-recsys/catalog-reader-api/internal/dto"
	"github.com/multimodal-recsys/catalog-reader-api/internal/dto/converter"
	"github.com/multimodal-recsys/catalog-reader-api/internal/repository"
)

type productUsecase struct {
	products repository.ProductRepository
	ranking  *client.RankingClient
}

func NewProductUsecase(products repository.ProductRepository, ranking *client.RankingClient) ProductUsecase {
	return &productUsecase{products: products, ranking: ranking}
}

func (u *productUsecase) Get(ctx context.Context, tenantID string, id int64) (dto.ProductResponse, error) {
	p, err := u.products.Get(ctx, tenantID, id)
	if err != nil {
		return dto.ProductResponse{}, apperror.NotFound("product not found")
	}
	return converter.ToProductResponse(p), nil
}

// Search accepts a text query, an image query, or both, and delegates
// straight to search-ranking-service — this usecase's job is just
// tenant-scoping the request, not ranking anything itself.
func (u *productUsecase) Search(ctx context.Context, tenantID string, req dto.SearchRequest) ([]dto.SearchResultItem, error) {
	results, err := u.ranking.Search(ctx, tenantID, req)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return results, nil
}
