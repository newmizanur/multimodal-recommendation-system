// Package auth implements tenant authentication for this SaaS API: each
// tenant (e.g. "acme-marketplace") is issued an API key, sent as
// X-API-Key and checked against the :tenant_id path segment.
//
// The lookup here is an in-memory map seeded from config for the
// scaffold. Swap Store for a real backing table (or a secrets manager
// lookup) without touching the middleware in
// internal/delivery/http/middleware — it only depends on this interface.
package auth

import "context"

type Store interface {
	// Validate returns true if apiKey is the current, valid key for tenantID.
	Validate(ctx context.Context, tenantID, apiKey string) bool
}

type StaticStore struct {
	// keys maps tenant_id -> API key. TODO: replace with a DB-backed or
	// secrets-manager-backed implementation before production use — a
	// static map means every key rotation is a redeploy.
	keys map[string]string
}

func NewStaticStore(keys map[string]string) *StaticStore {
	return &StaticStore{keys: keys}
}

func (s *StaticStore) Validate(_ context.Context, tenantID, apiKey string) bool {
	expected, ok := s.keys[tenantID]
	return ok && apiKey != "" && apiKey == expected
}
