package config

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"

	"github.com/multimodal-recsys/search-ranking-service/internal/client"
	"github.com/multimodal-recsys/search-ranking-service/internal/repository"
)

func NewCacheRepository(v *viper.Viper) repository.CacheRepository {
	return repository.NewCacheRepository(v.GetString("REDIS_URL"))
}

func NewAnnRepository(db *pgxpool.Pool) repository.AnnRepository {
	return repository.NewAnnRepository(db)
}

func NewEmbedClient(v *viper.Viper) *client.EmbedClient {
	return client.NewEmbedClient(v.GetString("EMBEDDING_SERVER_URL"))
}

func NewRerankClient(v *viper.Viper) *client.RerankClient {
	return client.NewRerankClient(v.GetString("RERANK_MODEL_URL"))
}
