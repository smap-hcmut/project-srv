package http

import (
	"project-srv/internal/crisis"
	"project-srv/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Handler defines the HTTP handler interface for Crisis Config.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      crisis.UseCase
	discord discord.IDiscord
}

// New creates a new Crisis Config HTTP handler.
func New(l log.Logger, uc crisis.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
