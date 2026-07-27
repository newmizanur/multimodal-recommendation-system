package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"

	"github.com/multimodal-recsys/worker/internal/client"
)

const (
	testTenantA = "vecrepotest-tenant-a"
	testTenantB = "vecrepotest-tenant-b"

	testDatabaseURL = "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable"
)

// testPool connects with the same AfterConnect hook config.NewDatabase
// uses in production — registering pgvector's `vector` type per
// connection, without which Upsert fails as soon as it actually runs
// against Postgres. Duplicated here rather than imported from
// internal/config, which itself imports this package (repository) to
// build NewVectorRepository — importing it back would be a cycle.
// Skips rather than fails if Postgres isn't reachable.
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

func cleanupProducts(pool *pgxpool.Pool, productIDs ...int64) {
	_, _ = pool.Exec(context.Background(), "DELETE FROM product_embeddings WHERE product_id = ANY($1)", productIDs)
}

func countEmbeddings(t *testing.T, pool *pgxpool.Pool, productID int64) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM product_embeddings WHERE product_id = $1`, productID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	return count
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

// TestVectorRepository_UpsertIsIdempotent backs the idempotent-upsert
// claim: a worker can be killed after processing a Kafka message but
// before committing its offset, so it will occasionally reprocess the
// same event. Reprocessing must update the one row in place, not insert
// a second one.
func TestVectorRepository_UpsertIsIdempotent(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const productID int64 = 9100001
	defer cleanupProducts(pool, productID)

	repo := NewVectorRepository(pool)
	ctx := context.Background()

	v := Vector{
		ProductID: "9100001", TenantID: testTenantA, EmbeddingType: "image",
		Values: vecDir(0.1, 0.9), SourceRef: "img1", CategoryID: "1", InStock: true,
		Metadata: map[string]any{},
	}
	if err := repo.Upsert(ctx, v); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	v.Values = vecDir(0.9, 0.1) // simulates a Kafka redelivery reprocessing the same event
	if err := repo.Upsert(ctx, v); err != nil {
		t.Fatalf("second upsert (redelivery): %v", err)
	}

	if got := countEmbeddings(t, pool, productID); got != 1 {
		t.Fatalf("expected exactly 1 row after two upserts on the same (product_id, embedding_type, source_ref), got %d", got)
	}
}

// TestVectorRepository_HardDeleteRespectsTenantBoundary backs the
// tenant-scoping claim for the write side of the vector store: deleting
// under the wrong tenant must be a no-op, not a cross-tenant deletion.
func TestVectorRepository_HardDeleteRespectsTenantBoundary(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const productID int64 = 9100002
	defer cleanupProducts(pool, productID)

	repo := NewVectorRepository(pool)
	ctx := context.Background()

	if err := repo.Upsert(ctx, Vector{
		ProductID: "9100002", TenantID: testTenantA, EmbeddingType: "image",
		Values: vecDir(0.1, 0.9), SourceRef: "img1", CategoryID: "1", InStock: true,
		Metadata: map[string]any{},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Deleting under a DIFFERENT tenant must not remove the row.
	if err := repo.HardDeleteByProduct(ctx, testTenantB, "9100002"); err != nil {
		t.Fatalf("cross-tenant delete call: %v", err)
	}
	if got := countEmbeddings(t, pool, productID); got != 1 {
		t.Fatalf("tenant isolation violated: wrong-tenant HardDeleteByProduct removed the row (count=%d, want 1)", got)
	}

	// The owning tenant's delete actually removes it.
	if err := repo.HardDeleteByProduct(ctx, testTenantA, "9100002"); err != nil {
		t.Fatalf("owning-tenant delete: %v", err)
	}
	if got := countEmbeddings(t, pool, productID); got != 0 {
		t.Fatalf("expected row gone after owning-tenant hard delete, got count=%d", got)
	}
}

// TestVectorRepository_EmbedClientHTTPRoundTrip is what backs the
// article/README's "exercised with a real HTTP round-trip through a
// stand-in embedding server matching the /embed contract" claim: a real
// client.EmbedClient making a real HTTP call to a real net/http server
// (not a mocked interface), with the response written to Postgres
// through the real VectorRepository.
func TestVectorRepository_EmbedClientHTTPRoundTrip(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const productID int64 = 9100003
	defer cleanupProducts(pool, productID)

	stubVector := vecDir(0.42, 0.13)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req client.EmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(client.EmbedResponse{Vector: stubVector})
	}))
	defer server.Close()

	embedClient := client.NewEmbedClient(server.URL)
	resp, err := embedClient.Embed(client.EmbedRequest{Kind: "text", Prompt: "catalog_text", Text: "test product"})
	if err != nil {
		t.Fatalf("embed call: %v", err)
	}
	if len(resp.Vector) != 768 {
		t.Fatalf("expected 768-dim vector from stand-in embed server, got %d", len(resp.Vector))
	}

	repo := NewVectorRepository(pool)
	if err := repo.Upsert(context.Background(), Vector{
		ProductID: "9100003", TenantID: testTenantA, EmbeddingType: "text_title",
		Values: resp.Vector, SourceRef: "text_title", CategoryID: "1", InStock: true,
		Metadata: map[string]any{},
	}); err != nil {
		t.Fatalf("upsert vector produced by the embed client: %v", err)
	}

	if got := countEmbeddings(t, pool, productID); got != 1 {
		t.Fatalf("expected the embed-client-produced vector to land in Postgres, got count=%d", got)
	}
}
