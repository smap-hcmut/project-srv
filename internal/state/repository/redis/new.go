package redis

import (
	"smap-project/internal/state/repository"
	pkgLog "smap-project/pkg/log"
	pkgRedis "smap-project/pkg/redis"
)

// redisStateRepository implements StateRepository using Redis Hash.
type redisStateRepository struct {
	client pkgRedis.Client
	logger pkgLog.Logger
}

// NewStateRepository creates a new Redis-based state repository.
func NewStateRepository(client pkgRedis.Client, logger pkgLog.Logger) repository.StateRepository {
	return &redisStateRepository{
		client: client,
		logger: logger,
	}
}
