package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/catalog-reader-api/internal/apperror"
	"github.com/multimodal-recsys/catalog-reader-api/internal/delivery/http/response"
	"github.com/multimodal-recsys/catalog-reader-api/internal/dto"
	"github.com/multimodal-recsys/catalog-reader-api/internal/usecase"
)

type ProductController struct {
	usecase usecase.ProductUsecase
}

func NewProductController(uc usecase.ProductUsecase) *ProductController {
	return &ProductController{usecase: uc}
}

func (h *ProductController) Health(c echo.Context) error {
	return response.Success(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ProductController) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, apperror.Validation("invalid product id"))
	}

	tenantID := c.Param("tenant_id")
	result, err := h.usecase.Get(c.Request().Context(), tenantID, id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, result)
}

// Search accepts a text query, an image query, or both, and delegates to
// the Search & Ranking Service. This handler does no ANN work itself.
func (h *ProductController) Search(c echo.Context) error {
	var req dto.SearchRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}

	tenantID := c.Param("tenant_id")
	results, err := h.usecase.Search(c.Request().Context(), tenantID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, results)
}
