package config

import (
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"

	"github.com/multimodal-recsys/worker/internal/client"
	"github.com/multimodal-recsys/worker/internal/delivery/kafka"
	"github.com/multimodal-recsys/worker/internal/repository"
	"github.com/multimodal-recsys/worker/internal/storage"
)

func NewKafkaConsumer(v *viper.Viper) *kafka.Consumer {
	brokers := strings.Split(v.GetString("KAFKA_BROKERS"), ",")
	return kafka.NewConsumer(brokers, v.GetString("KAFKA_TOPIC"), v.GetString("KAFKA_CONSUMER_GROUP"))
}

// NewObjectStorage talks to any S3-compatible store. Set S3_ENDPOINT for
// MinIO/R2/etc; leave it unset to use AWS S3 directly — same convention
// as catalog-writer-api.
func NewObjectStorage(v *viper.Viper) (*storage.S3Reader, error) {
	return storage.NewS3Reader(v.GetString("S3_BUCKET"), v.GetString("S3_ENDPOINT"))
}

func NewEmbedClient(v *viper.Viper) *client.EmbedClient {
	return client.NewEmbedClient(v.GetString("EMBEDDING_SERVER_URL"))
}

func NewVectorRepository(db *pgxpool.Pool) repository.VectorRepository {
	return repository.NewVectorRepository(db)
}
