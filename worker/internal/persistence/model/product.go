package model

// ProductContent is the subset of a product's catalog data the worker
// needs to embed it — deliberately narrower than catalog-writer-api's
// full Product model, since this service only ever reads for embedding.
type ProductContent struct {
	ID          int64
	TenantID    string
	Title       string
	Description string
	CategoryID  string
	InStock     bool
	ImageKeys   []string // S3-compatible object keys, one per product image
}
