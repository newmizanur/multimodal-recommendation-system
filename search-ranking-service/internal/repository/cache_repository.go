package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetQueryResult(ctx context.Context, key string) (any, bool)
	SetQueryResult(ctx context.Context, key string, value any)
}

type redisCacheRepository struct {
	rdb *redis.Client
}

func NewCacheRepository(addr string) CacheRepository {
	return &redisCacheRepository{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

// QueryResultKey deliberately excludes user_id — personalization happens
// at rerank on a shared candidate set, not at retrieval. Keying by user
// fragments the cache and tanks hit rate for no accuracy gain.
func QueryResultKey(tenantID, query string, filters map[string]any) string {
	f, _ := json.Marshal(filters)
	sum := sha256.Sum256([]byte(tenantID + "|" + query + "|" + string(f)))
	return "qrc:" + hex.EncodeToString(sum[:])
}

func (r *redisCacheRepository) GetQueryResult(ctx context.Context, key string) (any, bool) {
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	var out any
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return nil, false
	}
	return out, true
}

// SetQueryResult uses a short 30-60s TTL — low-stakes staleness is an
// acceptable tradeoff for hit rate here, unlike the enrichment cache
// (see article), which needs active write-through on price/stock change.
func (r *redisCacheRepository) SetQueryResult(ctx context.Context, key string, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		return
	}
	r.rdb.Set(ctx, key, body, 45*time.Second)
}
