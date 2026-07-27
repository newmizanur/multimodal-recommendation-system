package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

const (
	testTenantA = "annrepotest-tenant-a"
	testTenantB = "annrepotest-tenant-b"

	testDatabaseURL = "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable"
)

// testPool registers pgvector's `vector` type per connection, the same
// way config.NewDatabase does in production — duplicated here rather
// than imported, since internal/config imports this package (repository)
// to build NewAnnRepository, and importing it back would be a cycle.
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

func seedEmbedding(t *testing.T, pool *pgxpool.Pool, productID int64, tenantID string, vec []float32, categoryID int64, inStock bool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO product_embeddings (product_id, tenant_id, embedding_type, embedding, source_ref, category_id, in_stock, metadata)
		VALUES ($1, $2, 'image', $3, 'img1', $4, $5, '{}')
	`, productID, tenantID, pgvector.NewVector(vec), categoryID, inStock)
	if err != nil {
		t.Fatalf("seed embedding for product %d: %v", productID, err)
	}
}

func containsOnly(results []AnnCandidate, allowed ...string) (violator string, ok bool) {
	allowedSet := map[string]bool{}
	for _, a := range allowed {
		allowedSet[a] = true
	}
	for _, r := range results {
		if !allowedSet[r.ProductID] {
			return r.ProductID, false
		}
	}
	return "", true
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

// TestAnnRepository_TenantIsolation backs the "tenant/category/in-stock
// pre-filtering... confirmed against a live database" claim, specifically
// the tenant half: tenant B's vector is an exact match for the query, so
// if the tenant_id pre-filter were ever dropped or post-applied, it would
// out-score tenant A's own (slightly different) vector and leak into
// tenant A's results.
func TestAnnRepository_TenantIsolation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const (
		productA int64 = 9200001
		productB int64 = 9200002
	)
	defer cleanupProducts(pool, productA, productB)

	query := vecDir(0.9, 0.1)
	seedEmbedding(t, pool, productA, testTenantA, vecDir(0.8, 0.2), 1, true)
	seedEmbedding(t, pool, productB, testTenantB, query, 1, true)

	repo := NewAnnRepository(pool)
	results, err := repo.Search(context.Background(), AnnSearchParams{
		TenantID: testTenantA, Vector: query, TopK: 10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, r := range results {
		if r.ProductID == "9200002" {
			t.Fatalf("tenant isolation violated: tenant B's product leaked into tenant A's results: %+v", results)
		}
	}
	found := false
	for _, r := range results {
		if r.ProductID == "9200001" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tenant A's own product in results, got %+v", results)
	}
}

// TestAnnRepository_CategoryAndInStockPreFilter backs the category/in-stock
// half of the same claim.
func TestAnnRepository_CategoryAndInStockPreFilter(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	const (
		matching      int64 = 9200011
		wrongCategory int64 = 9200012
		outOfStock    int64 = 9200013
	)
	defer cleanupProducts(pool, matching, wrongCategory, outOfStock)

	q := vecDir(0.9, 0.1)
	seedEmbedding(t, pool, matching, testTenantA, q, 1, true)
	seedEmbedding(t, pool, wrongCategory, testTenantA, q, 2, true)
	seedEmbedding(t, pool, outOfStock, testTenantA, q, 1, false)

	repo := NewAnnRepository(pool)

	byCategory, err := repo.Search(context.Background(), AnnSearchParams{
		TenantID: testTenantA, Vector: q, TopK: 10,
		Filters: map[string]any{"category_id": "1"},
	})
	if err != nil {
		t.Fatalf("search by category: %v", err)
	}
	if violator, ok := containsOnly(byCategory, "9200011", "9200013"); !ok {
		t.Fatalf("category_id pre-filter leaked product %s: %+v", violator, byCategory)
	}

	inStockOnly, err := repo.Search(context.Background(), AnnSearchParams{
		TenantID: testTenantA, Vector: q, TopK: 10,
		Filters: map[string]any{"in_stock": true},
	})
	if err != nil {
		t.Fatalf("search in_stock: %v", err)
	}
	if violator, ok := containsOnly(inStockOnly, "9200011", "9200012"); !ok {
		t.Fatalf("in_stock pre-filter leaked product %s: %+v", violator, inStockOnly)
	}
}
