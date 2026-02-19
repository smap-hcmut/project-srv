package http

import (
	"project-srv/internal/campaign"
	"project-srv/internal/middleware"
	"project-srv/pkg/discord"
	"project-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler defines the HTTP handler interface for Campaign.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      campaign.UseCase
	discord discord.IDiscord
}

// New creates a new Campaign HTTP handler.
func New(l log.Logger, uc campaign.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
