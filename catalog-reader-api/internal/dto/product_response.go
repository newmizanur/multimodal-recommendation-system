package dto

type ProductResponse struct {
	ID       int64   `json:"id"`
	SKU      string  `json:"sku"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	InStock  bool    `json:"in_stock"`
}

type SearchRequest struct {
	Query       string         `json:"query,omitempty"`
	ImageBase64 string         `json:"image_base64,omitempty"`
	UserID      string         `json:"user_id,omitempty" validate:"omitempty"` // used for rerank personalization only, never the cache key
	Filters     map[string]any `json:"filters,omitempty"`
}

type SearchResultItem struct {
	ProductID string  `json:"product_id"`
	Score     float64 `json:"score"`
}
