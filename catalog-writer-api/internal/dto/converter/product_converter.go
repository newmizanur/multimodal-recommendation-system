package converter

import (
	"github.com/multimodal-recsys/catalog-writer-api/internal/dto"
	"github.com/multimodal-recsys/catalog-writer-api/internal/persistence/model"
)

func NewProductFromCreateRequest(tenantID string, req dto.CreateProductRequest) model.Product {
	return model.Product{
		TenantID:    tenantID,
		SKU:         req.SKU,
		Title:       req.Title,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Price:       req.Price,
		Currency:    req.Currency,
		StockQty:    req.StockQty,
		Status:      "active",
		Attributes:  req.Attributes,
	}
}

// ApplyUpdate mutates p in place with any fields set in req — pointer
// fields left nil mean "unchanged", matching PATCH-style semantics even
// though the route is a PUT for simplicity.
func ApplyUpdate(p *model.Product, req dto.UpdateProductRequest) {
	if req.Title != nil {
		p.Title = *req.Title
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	if req.StockQty != nil {
		p.StockQty = *req.StockQty
	}
	if req.Attributes != nil {
		p.Attributes = req.Attributes
	}
}

func ToProductResponse(p model.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID:         p.ID,
		SKU:        p.SKU,
		Title:      p.Title,
		CategoryID: p.CategoryID,
		Price:      p.Price,
		Currency:   p.Currency,
		StockQty:   p.StockQty,
		Status:     p.Status,
		Attributes: p.Attributes,
	}
}
