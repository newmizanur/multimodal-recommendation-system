package config

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
	"github.com/spf13/viper"
)

// NewDatabase opens a read-only-usage Postgres pool — the worker only
// ever reads product content to embed; it never writes to the catalog
// schema (see catalog-writer-api for the single writer). The pool is
// also used by the vector repository, so every connection registers
// pgvector's `vector` type on connect.
func NewDatabase(v *viper.Viper) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(v.GetString("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}
	return pgxpool.NewWithConfig(context.Background(), cfg)
}
