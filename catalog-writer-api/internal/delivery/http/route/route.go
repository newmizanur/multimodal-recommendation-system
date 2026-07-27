// Package route is the one place that knows the URL shape. Adding an
// endpoint means adding a line here, not hunting through main.go.
package route

import (
	"github.com/labstack/echo/v4"

	httpdelivery "github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http"
	"github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http/middleware"
	"github.com/multimodal-recsys/catalog-writer-api/internal/auth"
)

func Register(e *echo.Echo, products *httpdelivery.ProductController, authStore auth.Store) {
	e.GET("/healthz", products.Health)

	api := e.Group("/v1/tenants/:tenant_id/products", middleware.TenantAuth(authStore))
	api.POST("", products.Create)
	api.PUT("/:id", products.Update)
	api.DELETE("/:id", products.Delete)
	api.POST("/:id/image-upload-url", products.PresignImageUpload)
}
