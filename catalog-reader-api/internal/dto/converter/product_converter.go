package converter

import (
	"github.com/multimodal-recsys/catalog-reader-api/internal/dto"
	"github.com/multimodal-recsys/catalog-reader-api/internal/persistence/model"
)

func ToProductResponse(p model.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID:       p.ID,
		SKU:      p.SKU,
		Title:    p.Title,
		Price:    p.Price,
		Currency: p.Currency,
		InStock:  p.InStock,
	}
}
