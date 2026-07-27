package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/catalog-writer-api/internal/apperror"
	"github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http/response"
	"github.com/multimodal-recsys/catalog-writer-api/internal/dto"
	"github.com/multimodal-recsys/catalog-writer-api/internal/usecase"
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

func (h *ProductController) Create(c echo.Context) error {
	var req dto.CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}

	tenantID := c.Param("tenant_id")
	result, err := h.usecase.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusCreated, result)
}

func (h *ProductController) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, apperror.Validation("invalid product id"))
	}

	var req dto.UpdateProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}

	tenantID := c.Param("tenant_id")
	result, err := h.usecase.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, result)
}

func (h *ProductController) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, apperror.Validation("invalid product id"))
	}

	tenantID := c.Param("tenant_id")
	if err := h.usecase.Delete(c.Request().Context(), tenantID, id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ProductController) PresignImageUpload(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, apperror.Validation("invalid product id"))
	}

	tenantID := c.Param("tenant_id")
	result, err := h.usecase.PresignImageUpload(c.Request().Context(), tenantID, id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, result)
}
