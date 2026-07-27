// Search & Ranking Service — the two-stage retrieval pipeline: query
// result cache -> embed query -> ANN retrieval -> enrichment -> rerank
// (cross-encoder + business rules). Internal-only; see internal/delivery/http/route.
package main

import (
	"github.com/multimodal-recsys/search-ranking-service/internal/config"
	httpdelivery "github.com/multimodal-recsys/search-ranking-service/internal/delivery/http"
	"github.com/multimodal-recsys/search-ranking-service/internal/delivery/http/route"
	"github.com/multimodal-recsys/search-ranking-service/internal/usecase"
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

	cache := config.NewCacheRepository(v)
	ann := config.NewAnnRepository(db)
	embed := config.NewEmbedClient(v)
	rerank := config.NewRerankClient(v)

	searchUC := usecase.NewSearchUsecase(cache, ann, embed, rerank)
	searchController := httpdelivery.NewSearchController(searchUC)

	e := config.NewEcho()
	route.Register(e, searchController)

	logger.Infof("search-ranking-service listening on :%s", appCfg.Port)
	e.Logger.Fatal(e.Start(":" + appCfg.Port))
}
