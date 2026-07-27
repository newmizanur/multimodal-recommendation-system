package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvector "github.com/pgvector/pgvector-go/pgx"

	"github.com/multimodal-recsys/search-ranking-service/internal/repository"
)

const testDatabaseURL = "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable"

// testPool registers pgvector's `vector` type per connection, the same
// way config.NewDatabase does in production. Skips rather than fails if
// Postgres isn't reachable, so `go test ./...` still passes on a machine
// without Docker running.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(testDatabaseURL)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("db connect: %v (run `task dev:up && task migrate:up` first)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres not reachable: %v (run `task dev:up && task migrate:up` first)", err)
	}
	return pool
}

// vecDir builds a direction, not just a magnitude — a uniform-fill vector
// is cosine-identical to any other uniform-fill vector regardless of
// scale, so tests need vectors that actually differ in angle.
func vecDir(a, b float32) []float32 {
	v := make([]float32, 768)
	for i := range v {
		if i%2 == 0 {
			v[i] = a
		} else {
			v[i] = b
		}
	}
	return v
}

// TestEmbedClient_HTTPRoundTripFeedsAnnSearch is what backs the
// article/README's "exercised with a real HTTP round-trip through a
// stand-in embedding server matching the /embed contract" claim on the
// read path: a real EmbedClient making a real HTTP call to a real
// net/http server (not a mocked interface), with the returned vector fed
// into a real AnnRepository.Search against Postgres.
func TestEmbedClient_HTTPRoundTripFeedsAnnSearch(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const (
		tenantID  = "embedclienttest-tenant"
		productID = int64(9300001)
	)
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM product_embeddings WHERE product_id = $1", productID)
	}()

	stubVector := vecDir(0.7, 0.3)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO product_embeddings (product_id, tenant_id, embedding_type, embedding, source_ref, category_id, in_stock, metadata)
		VALUES ($1, $2, 'image', $3, 'img1', 1, true, '{}')
	`, productID, tenantID, pgvector.NewVector(stubVector))
	if err != nil {
		t.Fatalf("seed embedding: %v", err)
	}

	// A minimal stand-in for the real embedding server, matching its
	// /embed contract exactly (see embedding-server/app/main.go).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Vector []float32 `json:"vector"`
		}{Vector: stubVector})
	}))
	defer server.Close()

	embedClient := NewEmbedClient(server.URL)
	queryVec, err := embedClient.EmbedText("test query")
	if err != nil {
		t.Fatalf("embed text: %v", err)
	}

	ann := repository.NewAnnRepository(pool)
	results, err := ann.Search(context.Background(), repository.AnnSearchParams{
		TenantID: tenantID, Vector: queryVec, TopK: 10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	found := false
	for _, r := range results {
		if r.ProductID == "9300001" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the stand-in-embed-server query vector to find the seeded product, got %+v", results)
	}
}
