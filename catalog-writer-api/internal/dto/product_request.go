// Package dto holds request/response shapes for the HTTP boundary — kept
// separate from persistence/model so a public API field rename never
// forces a DB migration and vice versa.
package dto

type CreateProductRequest struct {
	SKU         string         `json:"sku" validate:"required"`
	Title       string         `json:"title" validate:"required,max=200"`
	Description string         `json:"description"`
	CategoryID  string         `json:"category_id" validate:"required"`
	Price       float64        `json:"price" validate:"required,gt=0"`
	Currency    string         `json:"currency" validate:"required,len=3"`
	StockQty    int            `json:"stock_qty" validate:"gte=0"`
	Attributes  map[string]any `json:"attributes"`
}

type UpdateProductRequest struct {
	Title       *string        `json:"title" validate:"omitempty,max=200"`
	Description *string        `json:"description"`
	Price       *float64       `json:"price" validate:"omitempty,gt=0"`
	StockQty    *int           `json:"stock_qty" validate:"omitempty,gte=0"`
	Attributes  map[string]any `json:"attributes"`
}
