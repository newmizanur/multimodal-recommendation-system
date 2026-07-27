package client

import (
	"context"

	"github.com/multimodal-recsys/search-ranking-service/internal/repository"
)

type RerankClient struct {
	crossEncoderURL string
}

func NewRerankClient(crossEncoderURL string) *RerankClient {
	return &RerankClient{crossEncoderURL: crossEncoderURL}
}

type RerankedResult struct {
	ProductID string  `json:"product_id"`
	Score     float64 `json:"score"`
}

// Rerank runs the cross-encoder on the ANN candidates (top-k only —
// it's O(N) per candidate), then applies business rules (stock, price
// bucket, seller trust, events-table personalization signals for
// userID) on top of that ranking.
func (c *RerankClient) Rerank(ctx context.Context, query string, candidates []repository.AnnCandidate, userID string) ([]RerankedResult, error) {
	// TODO: call cross-encoder service on top 50-200 candidates by ANN score
	// TODO: apply business rule layer (stock/price/trust/events) after
	results := make([]RerankedResult, 0, len(candidates))
	for _, cand := range candidates {
		results = append(results, RerankedResult{ProductID: cand.ProductID, Score: cand.Score})
	}
	return results, nil
}
