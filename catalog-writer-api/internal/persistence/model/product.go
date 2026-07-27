// Package model holds structs that map 1:1 to database rows. Nothing
// outside internal/repository should construct or depend on these
// directly — the dto/converter layer is the boundary.
package model

import "time"

type Product struct {
	ID          int64
	TenantID    string
	SKU         string
	Title       string
	Description string
	CategoryID  string
	Price       float64
	Currency    string
	StockQty    int
	Status      string
	Attributes  map[string]any // maps to the `attributes jsonb` column
	UpdatedAt   time.Time
}
