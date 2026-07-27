package dto

type SearchRequest struct {
	TenantID    string         `json:"tenant_id" validate:"required"`
	UserID      string         `json:"user_id,omitempty"` // used for rerank personalization, NEVER the cache key
	Query       string         `json:"query,omitempty"`
	ImageBase64 string         `json:"image_base64,omitempty"`
	Filters     map[string]any `json:"filters,omitempty"`
}

type SearchResultItem struct {
	ProductID string  `json:"product_id"`
	Score     float64 `json:"score"`
}
