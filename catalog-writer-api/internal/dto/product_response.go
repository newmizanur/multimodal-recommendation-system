package dto

type ProductResponse struct {
	ID         int64          `json:"id"`
	SKU        string         `json:"sku"`
	Title      string         `json:"title"`
	CategoryID string         `json:"category_id"`
	Price      float64        `json:"price"`
	Currency   string         `json:"currency"`
	StockQty   int            `json:"stock_qty"`
	Status     string         `json:"status"`
	Attributes map[string]any `json:"attributes"`
}

type PresignResponse struct {
	UploadURL string `json:"upload_url"`
}
