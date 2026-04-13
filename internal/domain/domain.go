// Package domain provides cross-service domain type registry backed by Redis.
//
// analysis-srv publishes the available domain list to Redis on startup.
// This package reads that data so project-srv can validate domain_type_code
// and expose a domain list endpoint without a direct DB dependency on
// schema_analysis.
package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

// RedisKeyDomains is the shared key used by analysis-srv (writer) and
// project-srv (reader).  Must match analysis-srv ConsumerServer.REDIS_KEY_DOMAINS.
const RedisKeyDomains = "smap:domains"

// Domain represents a single domain entry published by analysis-srv.
type Domain struct {
	DomainCode  string `json:"domain_code"`
	DisplayName string `json:"display_name"`
}

// Repository defines read-only access to the domain registry.
type Repository interface {
	// ListActive returns all domains currently published by analysis-srv.
	ListActive(ctx context.Context) ([]Domain, error)
	// Exists checks whether a domain_code is present in the registry.
	Exists(ctx context.Context, code string) (bool, error)
}

// redisRepo implements Repository by reading from Redis.
type redisRepo struct {
	redis redis.IRedis
	l     log.Logger
}

// NewRepository creates a Redis-backed domain repository.
func NewRepository(r redis.IRedis, l log.Logger) Repository {
	return &redisRepo{redis: r, l: l}
}

func (r *redisRepo) ListActive(ctx context.Context) ([]Domain, error) {
	raw, err := r.redis.Get(ctx, RedisKeyDomains)
	if err != nil {
		return nil, fmt.Errorf("domain.ListActive: redis GET %s: %w", RedisKeyDomains, err)
	}
	if raw == "" {
		// Key does not exist — analysis-srv has not started yet.
		return nil, fmt.Errorf("domain.ListActive: key %s not found in Redis (analysis-srv may not have started)", RedisKeyDomains)
	}

	var domains []Domain
	if err := json.Unmarshal([]byte(raw), &domains); err != nil {
		return nil, fmt.Errorf("domain.ListActive: unmarshal: %w", err)
	}
	return domains, nil
}

func (r *redisRepo) Exists(ctx context.Context, code string) (bool, error) {
	domains, err := r.ListActive(ctx)
	if err != nil {
		return false, err
	}
	for _, d := range domains {
		if d.DomainCode == code {
			return true, nil
		}
	}
	return false, nil
}
