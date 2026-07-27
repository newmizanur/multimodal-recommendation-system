// Worker consumes catalog change events from Kafka, calls the embedding
// server, and writes vectors to the vector DB. Scaled by KEDA on
// consumer group lag (see infra/helm/worker). Wiring only — see
// internal/usecase/ingest_usecase.go for the actual embedding logic.
package main

import (
	"context"

	"github.com/multimodal-recsys/worker/internal/config"
	"github.com/multimodal-recsys/worker/internal/repository"
	"github.com/multimodal-recsys/worker/internal/usecase"
)

func main() {
	ctx := context.Background()

	v := config.NewViper()
	logger := config.NewLogger(v)
	_ = config.LoadAppConfig(v)

	db, err := config.NewDatabase(v)
	if err != nil {
		logger.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	images, err := config.NewObjectStorage(v)
	if err != nil {
		logger.Fatalf("object storage init: %v", err)
	}

	embedder := config.NewEmbedClient(v)

	vectors := config.NewVectorRepository(db)

	catalog := repository.NewCatalogRepository(db)
	ingestUC := usecase.NewIngestUsecase(catalog, vectors, images, embedder)

	consumer := config.NewKafkaConsumer(v)

	logger.Info("worker started, consuming from Kafka")
	consumer.Run(ctx, ingestUC.HandleEvent, func(err error) {
		logger.WithError(err).Warn("event processing error")
	})
}
