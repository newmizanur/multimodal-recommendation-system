package route

import (
	"github.com/labstack/echo/v4"

	"github.com/multimodal-recsys/catalog-reader-api/internal/auth"
	httpdelivery "github.com/multimodal-recsys/catalog-reader-api/internal/delivery/http"
	"github.com/multimodal-recsys/catalog-reader-api/internal/delivery/http/middleware"
)

func Register(e *echo.Echo, products *httpdelivery.ProductController, authStore auth.Store) {
	e.GET("/healthz", products.Health)

	api := e.Group("/v1/tenants/:tenant_id", middleware.TenantAuth(authStore))
	api.GET("/products/:id", products.Get)
	api.POST("/search", products.Search)
}
