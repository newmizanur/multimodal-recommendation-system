package model

type Product struct {
	ID         int64
	TenantID   string
	SKU        string
	Title      string
	Price      float64
	Currency   string
	StockQty   int
	InStock    bool
}
