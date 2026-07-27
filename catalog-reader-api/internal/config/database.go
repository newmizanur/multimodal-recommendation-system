package config

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// NewDatabase opens a read-only-usage Postgres pool — this service never
// writes; see catalog-writer-api for the single writer (article:
// "single writer" design decision).
func NewDatabase(v *viper.Viper) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), v.GetString("DATABASE_URL"))
}
