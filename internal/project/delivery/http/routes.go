package http

import (
	"project-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps project routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
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
		projects.GET("/:projectId", h.Detail)
		projects.PUT("/:projectId", h.Update)
		projects.DELETE("/:projectId", h.Archive)
	}
}
