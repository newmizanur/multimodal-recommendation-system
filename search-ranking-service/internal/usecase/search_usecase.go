package usecase

import (
	"context"

	"github.com/multimodal-recsys/search-ranking-service/internal/apperror"
	"github.com/multimodal-recsys/search-ranking-service/internal/client"
	"github.com/multimodal-recsys/search-ranking-service/internal/dto"
	"github.com/multimodal-recsys/search-ranking-service/internal/repository"
)

type searchUsecase struct {
	cache  repository.CacheRepository
	ann    repository.AnnRepository
	embed  *client.EmbedClient
	rerank *client.RerankClient
}

func NewSearchUsecase(cache repository.CacheRepository, ann repository.AnnRepository, embed *client.EmbedClient, rerank *client.RerankClient) SearchUsecase {
	return &searchUsecase{cache: cache, ann: ann, embed: embed, rerank: rerank}
}

// Search implements the read path: cache check -> encode -> ANN ->
// enrich -> rerank. The query result cache key deliberately excludes
// UserID — personalization is applied at rerank, not retrieval.
func (u *searchUsecase) Search(ctx context.Context, req dto.SearchRequest) ([]dto.SearchResultItem, error) {
	cacheKey := repository.QueryResultKey(req.TenantID, req.Query, req.Filters)
	if cached, ok := u.cache.GetQueryResult(ctx, cacheKey); ok {
		if results, ok := cached.([]dto.SearchResultItem); ok {
			return results, nil
		}
	}

	var vector []float32
	var err error
	if req.ImageBase64 != "" {
		vector, err = u.embed.EmbedImage(req.ImageBase64)
	} else {
		vector, err = u.embed.EmbedText(req.Query)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	candidates, err := u.ann.Search(ctx, repository.AnnSearchParams{
		TenantID: req.TenantID,
		Vector:   vector,
		TopK:     300,
		Filters:  req.Filters,
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}

	// TODO: enrich candidates with live price/stock from Redis
	// (write-through cache), falling back to the Catalog DB on miss —
	// via a call to catalog-reader-api or a shared read repository.

	ranked, err := u.rerank.Rerank(ctx, req.Query, candidates, req.UserID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	results := make([]dto.SearchResultItem, 0, len(ranked))
	for _, r := range ranked {
		results = append(results, dto.SearchResultItem{ProductID: r.ProductID, Score: r.Score})
	}

	u.cache.SetQueryResult(ctx, cacheKey, results)
	return results, nil
}
