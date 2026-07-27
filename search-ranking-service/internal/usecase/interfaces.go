package usecase

import (
	"context"

	"github.com/multimodal-recsys/search-ranking-service/internal/dto"
)

type SearchUsecase interface {
	Search(ctx context.Context, req dto.SearchRequest) ([]dto.SearchResultItem, error)
}
