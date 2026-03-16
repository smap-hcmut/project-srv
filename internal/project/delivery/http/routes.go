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
		projects.GET("/:project_id", h.Detail)
		projects.PUT("/:project_id", h.Update)
		projects.DELETE("/:project_id", h.Archive)
	}
}
