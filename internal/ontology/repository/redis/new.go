package redis

import (
	"project-srv/internal/ontology/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	sharedredis "github.com/smap-hcmut/shared-libs/go/redis"
)

type implRepository struct {
	redis sharedredis.IRedis
	l     log.Logger
}

func New(client sharedredis.IRedis, l log.Logger) repository.Repository {
	return &implRepository{redis: client, l: l}
}
