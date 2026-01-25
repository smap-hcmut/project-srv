package usecase

import (
	"smap-project/internal/webhook"
	pkgLog "smap-project/pkg/log"
	pkgRedis "smap-project/pkg/redis"
)

type usecase struct {
	l           pkgLog.Logger
	redisClient pkgRedis.Client
}

func New(l pkgLog.Logger, redisClient pkgRedis.Client) webhook.UseCase {
	return &usecase{
		l:           l,
		redisClient: redisClient,
	}
}
