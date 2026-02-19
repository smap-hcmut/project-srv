package middleware

import (
	"project-srv/config"
	pkgLog "project-srv/pkg/log"
	pkgScope "project-srv/pkg/scope"
)

type Middleware struct {
	l            pkgLog.Logger
	jwtManager   pkgScope.Manager
	cookieConfig config.CookieConfig
	InternalKey  string
}

func New(l pkgLog.Logger, jwtManager pkgScope.Manager, cookieConfig config.CookieConfig, internalKey string) Middleware {
	return Middleware{
		l:            l,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
		InternalKey:  internalKey,
	}
}
