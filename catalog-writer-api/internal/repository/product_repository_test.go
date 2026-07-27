package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"

	"github.com/multimodal-recsys/catalog-writer-api/internal/config"
	"github.com/multimodal-recsys/catalog-writer-api/internal/persistence/model"
)

const (
	testTenantA = "repotest-tenant-a"
	testTenantB = "repotest-tenant-b"
)

// testPool connects with the same config.NewDatabase wiring main.go uses,
// against the docker-compose dev Postgres. Skips (not fails) if it isn't
// reachable, so `go test ./...` still passes on a machine without Docker
// running — `task test:all` is what guarantees it actually runs.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	v := viper.New()
	v.Set("DATABASE_URL", "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable")
	pool, err := config.NewDatabase(v)
	if err != nil {
		t.Skipf("db connect: %v (run `task dev:up && task migrate:up` first)", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres not reachable: %v (run `task dev:up && task migrate:up` first)", err)
	}
	return pool
}

func cleanupTenants(pool *pgxpool.Pool, tenants ...string) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM products WHERE tenant_id = ANY($1)", tenants)
	_, _ = pool.Exec(ctx, "DELETE FROM categories WHERE tenant_id = ANY($1)", tenants)
}

func seedCategory(t *testing.T, pool *pgxpool.Pool, tenantID string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO categories (tenant_id, name) VALUES ($1, 'repo-test-category') RETURNING id`,
		tenantID).Scan(&id)
	if err != nil {
		t.Fatalf("seed category: %v", err)
	}
	return id
}

// TestProductRepository_TenantIsolation backs the article/README claim
// that "every query confirmed tenant-scoped" for real: it creates a
// product under tenant A and asserts tenant B can never read, modify, or
// delete it through the repository — the one invariant the docs call
// "the difference between a slow query and one tenant reading another
// tenant's catalog."
func TestProductRepository_TenantIsolation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	cleanupTenants(pool, testTenantA, testTenantB)
	defer cleanupTenants(pool, testTenantA, testTenantB)

	catA := seedCategory(t, pool, testTenantA)
	repo := NewProductRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, model.Product{
		TenantID:    testTenantA,
		SKU:         "ISO-TEST-SKU",
		Title:       "Isolation Test Product",
		Description: "seeded by TestProductRepository_TenantIsolation",
		CategoryID:  strconv.FormatInt(catA, 10),
		Price:       9.99,
		Currency:    "USD",
		StockQty:    1,
		Status:      "active",
		Attributes:  map[string]any{},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Sanity: the owning tenant can read it back.
	got, err := repo.Get(ctx, testTenantA, created.ID)
	if err != nil {
		t.Fatalf("owning tenant Get: %v", err)
	}
	if got.SKU != "ISO-TEST-SKU" {
		t.Fatalf("got wrong product back: %+v", got)
	}

	// A different tenant must never be able to read it.
	if _, err := repo.Get(ctx, testTenantB, created.ID); err == nil {
		t.Fatalf("tenant isolation violated: tenant B successfully read tenant A's product %d", created.ID)
	}

	// Wrong-tenant Update must not touch the row (0 rows affected, not an error).
	if err := repo.Update(ctx, model.Product{
		ID: created.ID, TenantID: testTenantB, Title: "hijacked",
		Price: 1, StockQty: 1, Attributes: map[string]any{},
	}); err == nil {
		t.Fatalf("tenant isolation violated: tenant B successfully updated tenant A's product")
	}
	stillOwned, err := repo.Get(ctx, testTenantA, created.ID)
	if err != nil || stillOwned.Title != "Isolation Test Product" {
		t.Fatalf("product was modified by a cross-tenant update: %+v (err=%v)", stillOwned, err)
	}

	// Wrong-tenant Delete must not soft-delete the row.
	if err := repo.Delete(ctx, testTenantB, created.ID); err != nil {
		t.Fatalf("cross-tenant delete should silently affect 0 rows, not error: %v", err)
	}
	stillActive, err := repo.Get(ctx, testTenantA, created.ID)
	if err != nil || stillActive.Status != "active" {
		t.Fatalf("tenant isolation violated: cross-tenant delete soft-deleted the row: %+v (err=%v)", stillActive, err)
	}

	// The owning tenant's delete does soft-delete it.
	if err := repo.Delete(ctx, testTenantA, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	deleted, err := repo.Get(ctx, testTenantA, created.ID)
	if err != nil || deleted.Status != "deleted" {
		t.Fatalf("expected status=deleted after owning-tenant delete, got %+v (err=%v)", deleted, err)
	}
}
