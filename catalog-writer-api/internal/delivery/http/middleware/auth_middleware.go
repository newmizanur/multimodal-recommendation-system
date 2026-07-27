package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/catalog-writer-api/internal/apperror"
	"github.com/multimodal-recsys/catalog-writer-api/internal/auth"
	"github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http/response"
)

const TenantIDContextKey = "tenant_id"

// TenantAuth checks X-API-Key against the tenant named in the :tenant_id
// path param and, on success, stashes the tenant ID in the request
// context so downstream handlers/usecases never re-parse it.
func TenantAuth(store auth.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Param("tenant_id")
			apiKey := c.Request().Header.Get("X-API-Key")

			if !store.Validate(c.Request().Context(), tenantID, apiKey) {
				return response.Error(c, apperror.Unauthorized("invalid or missing API key for tenant"))
			}

			c.Set(TenantIDContextKey, tenantID)
			return next(c)
		}
	}
}
