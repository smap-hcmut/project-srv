package http

import (
	"project-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps crisis config routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	projects := r.Group("/projects")
	projects.Use(mw.Auth())
	{
		projects.PUT("/:projectId/crisis-config", h.Upsert)
		projects.GET("/:projectId/crisis-config", h.Detail)
		projects.DELETE("/:projectId/crisis-config", h.Delete)
	}
}
