// Catalog Reader API — public-facing search entry point. Read-only
// against Postgres; the actual retrieval/ranking work is delegated to
// the Search & Ranking Service so this tier stays cheap to scale
// horizontally on plain CPU/RPS.
package main

import (
	"github.com/multimodal-recsys/catalog-reader-api/internal/auth"
	"github.com/multimodal-recsys/catalog-reader-api/internal/config"
	httpdelivery "github.com/multimodal-recsys/catalog-reader-api/internal/delivery/http"
	"github.com/multimodal-recsys/catalog-reader-api/internal/delivery/http/route"
	"github.com/multimodal-recsys/catalog-reader-api/internal/repository"
	"github.com/multimodal-recsys/catalog-reader-api/internal/usecase"
)

func main() {
	v := config.NewViper()
	logger := config.NewLogger(v)
	appCfg := config.LoadAppConfig(v)

	db, err := config.NewDatabase(v)
	if err != nil {
		logger.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	ranking := config.NewRankingClient(v)

	// TODO: same static-store caveat as catalog-writer-api — replace
	// with a real tenant-key store before production use.
	authStore := auth.NewStaticStore(map[string]string{
		"acme-marketplace": v.GetString("DEMO_TENANT_API_KEY"),
	})

	products := repository.NewProductRepository(db)
	productUC := usecase.NewProductUsecase(products, ranking)
	productController := httpdelivery.NewProductController(productUC)

	e := config.NewEcho()
	route.Register(e, productController, authStore)

	logger.Infof("catalog-reader-api listening on :%s", appCfg.Port)
	e.Logger.Fatal(e.Start(":" + appCfg.Port))
}
