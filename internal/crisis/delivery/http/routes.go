package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes maps crisis config routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	projects := r.Group("/projects")
	projects.Use(mw.Auth())
	{
		projects.PUT("/:project_id/crisis-config", h.Upsert)
		projects.GET("/:project_id/crisis-config", h.Detail)
		projects.DELETE("/:project_id/crisis-config", h.Delete)
	}
}
