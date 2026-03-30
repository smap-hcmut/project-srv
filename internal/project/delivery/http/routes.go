package http

import (
	"github.com/smap-hcmut/shared-libs/go/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps project routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	// Nested under campaign — uses :id (same wildcard as campaign routes)
	campaigns := r.Group("/campaigns")
	campaigns.Use(mw.Auth())
	{
		campaigns.POST("/:id/projects", h.Create)
		campaigns.GET("/:id/projects", h.List)
	}

	// Direct project routes
	projects := r.Group("/projects")
	projects.Use(mw.Auth())
	{
		projects.GET("/favorites", h.ListFavorites)
		projects.GET("/:project_id", h.Detail)
		projects.PUT("/:project_id", h.Update)
		projects.POST("/:project_id/favorite", h.Favorite)
		projects.DELETE("/:project_id/favorite", h.Unfavorite)
		projects.GET("/:project_id/activation-readiness", h.ActivationReadiness)
		projects.POST("/:project_id/activate", h.Activate)
		projects.POST("/:project_id/pause", h.Pause)
		projects.POST("/:project_id/resume", h.Resume)
		projects.POST("/:project_id/archive", h.Archive)
		projects.POST("/:project_id/unarchive", h.Unarchive)
		projects.DELETE("/:project_id", h.Delete)
	}
}

func (h *handler) RegisterInternalRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	projects := r.Group("/projects")
	projects.Use(mw.InternalAuth())
	{
		projects.GET("/:project_id", h.InternalDetail)
	}
}
