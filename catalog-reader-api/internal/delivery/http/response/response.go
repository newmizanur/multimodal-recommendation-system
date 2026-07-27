// Package response gives every controller one way to render a success
// or an error, so the JSON shape is consistent across every endpoint in
// every service without each handler reinventing it.
package response

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/catalog-reader-api/internal/apperror"
)

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success(c echo.Context, status int, data any) error {
	return c.JSON(status, envelope{Data: data})
}

// Error renders an *apperror.AppError with its intended status/code, or
// falls back to a generic 500 for anything unexpected — callers never
// need to know which case they're in.
func Error(c echo.Context, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return c.JSON(appErr.Status, envelope{Error: &errorBody{Code: appErr.Code, Message: appErr.Message}})
	}
	return c.JSON(http.StatusInternalServerError, envelope{Error: &errorBody{Code: "INTERNAL_ERROR", Message: "internal server error"}})
}
