package http

import (
	"project-srv/internal/middleware"
	"project-srv/internal/project"
	"project-srv/pkg/discord"
	"project-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler defines the HTTP handler interface for Project.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
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
