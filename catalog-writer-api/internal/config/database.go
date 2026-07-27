package config

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// NewDatabase opens the Postgres pool. This service is the ONLY writer
// to the catalog schema — the worker and reader services must never
// write here (see article: "single writer" design decision).
func NewDatabase(v *viper.Viper) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), v.GetString("DATABASE_URL"))
}
