package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/search-ranking-service/internal/apperror"
	"github.com/multimodal-recsys/search-ranking-service/internal/delivery/http/response"
	"github.com/multimodal-recsys/search-ranking-service/internal/dto"
	"github.com/multimodal-recsys/search-ranking-service/internal/usecase"
)

type SearchController struct {
	usecase usecase.SearchUsecase
}

func NewSearchController(uc usecase.SearchUsecase) *SearchController {
	return &SearchController{usecase: uc}
}

func (h *SearchController) Health(c echo.Context) error {
	return response.Success(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SearchController) Search(c echo.Context) error {
	var req dto.SearchRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, apperror.Validation(err.Error()))
	}

	results, err := h.usecase.Search(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, results)
}
