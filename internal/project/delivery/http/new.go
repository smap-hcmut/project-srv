package http

import (
	"project-srv/internal/project"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// Handler defines the HTTP handler interface for Project.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      project.UseCase
	discord discord.IDiscord
}

// New creates a new Project HTTP handler.
func New(l log.Logger, uc project.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
