package config

import (
	"github.com/spf13/viper"

	"github.com/multimodal-recsys/catalog-reader-api/internal/client"
)

func NewRankingClient(v *viper.Viper) *client.RankingClient {
	return client.NewRankingClient(v.GetString("SEARCH_RANKING_URL"))
}
