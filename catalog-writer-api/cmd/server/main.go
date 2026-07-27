// Catalog Writer API — the single writer to the Catalog DB. Wiring only:
// build each layer (config -> repository -> usecase -> delivery) and
// start the server. Business logic lives in internal/usecase; SQL lives
// in internal/repository; HTTP shape lives in internal/delivery/http.
package main

import (
	"github.com/multimodal-recsys/catalog-writer-api/internal/auth"
	"github.com/multimodal-recsys/catalog-writer-api/internal/config"
	httpdelivery "github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http"
	"github.com/multimodal-recsys/catalog-writer-api/internal/delivery/http/route"
	"github.com/multimodal-recsys/catalog-writer-api/internal/repository"
	"github.com/multimodal-recsys/catalog-writer-api/internal/usecase"
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

	producer := config.NewKafkaProducer(v)
	defer producer.Close()

	uploads, err := config.NewObjectStorage(v)
	if err != nil {
		logger.Fatalf("object storage init: %v", err)
	}

	// TODO: replace with a real tenant-key store (DB or secrets manager)
	// before production use — see internal/auth/apikey.go.
	authStore := auth.NewStaticStore(map[string]string{
		"acme-marketplace": v.GetString("DEMO_TENANT_API_KEY"),
	})

	products := repository.NewProductRepository(db)
	productUC := usecase.NewProductUsecase(products, producer, uploads)
	productController := httpdelivery.NewProductController(productUC)

	e := config.NewEcho()
	route.Register(e, productController, authStore)

	logger.Infof("catalog-writer-api listening on :%s", appCfg.Port)
	e.Logger.Fatal(e.Start(":" + appCfg.Port))
}
