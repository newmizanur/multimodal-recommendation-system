// Package client holds outbound calls to other services in the system —
// kept separate from repository (which is DB-only) so it's obvious at a
// glance which dependencies are "our data" vs "someone else's service".
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/multimodal-recsys/catalog-reader-api/internal/dto"
)

type RankingClient struct {
	baseURL string
	http    *http.Client
}

func NewRankingClient(baseURL string) *RankingClient {
	return &RankingClient{baseURL: baseURL, http: &http.Client{}}
}

// Search delegates the actual retrieval/ranking pipeline (cache -> embed
// -> ANN -> enrich -> rerank) to search-ranking-service. This client
// does no ANN work itself.
func (c *RankingClient) Search(ctx context.Context, tenantID string, req dto.SearchRequest) ([]dto.SearchResultItem, error) {
	body, err := json.Marshal(struct {
		TenantID string `json:"tenant_id"`
		dto.SearchRequest
	}{TenantID: tenantID, SearchRequest: req})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []dto.SearchResultItem
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}
