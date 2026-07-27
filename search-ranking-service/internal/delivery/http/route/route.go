// This service is internal-only — called by catalog-reader-api, never
// exposed to tenants directly — so, unlike the two catalog-*-api
// services, there's no tenant API-key middleware here. Restrict access
// at the network layer (NetworkPolicy / service mesh / private
// ClusterIP) rather than re-implementing tenant auth a second time.
package route

import (
	"github.com/labstack/echo/v4"

	httpdelivery "github.com/multimodal-recsys/search-ranking-service/internal/delivery/http"
)

func Register(e *echo.Echo, search *httpdelivery.SearchController) {
	e.GET("/healthz", search.Health)
	e.POST("/v1/search", search.Search)
}
