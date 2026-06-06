package http

import (
	"github.com/smap-hcmut/shared-libs/go/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps project routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	// Domain registry — available domains for project creation
	domains := r.Group("/domains")
	domains.Use(mw.Auth())
	{
		domains.GET("", h.ListDomains)
	}

	// Nested under campaign — uses :id (same wildcard as campaign routes)
	campaigns := r.Group("/campaigns")
	campaigns.Use(mw.Auth())
	{
		campaigns.POST("/:id/projects", mw.AdminOnly(), h.Create)
		campaigns.GET("/:id/projects", h.List)
	}

	// Direct project routes
	projects := r.Group("/projects")
	projects.Use(mw.Auth())
	{
		projects.GET("/:project_id", h.Detail)
		projects.GET("/:project_id/access", h.Access)
		projects.PUT("/:project_id", mw.AdminOnly(), h.Update)
		projects.GET("/:project_id/activation-readiness", h.ActivationReadiness)
		projects.POST("/:project_id/activate", mw.AdminOnly(), h.Activate)
		projects.POST("/:project_id/pause", mw.AdminOnly(), h.Pause)
		projects.POST("/:project_id/resume", mw.AdminOnly(), h.Resume)
		projects.POST("/:project_id/archive", mw.AdminOnly(), h.Archive)
		projects.POST("/:project_id/unarchive", mw.AdminOnly(), h.Unarchive)
		projects.DELETE("/:project_id", mw.AdminOnly(), h.Delete)
	}
}

func (h *handler) RegisterInternalRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	projects := r.Group("/projects")
	projects.Use(mw.InternalAuth())
	{
		projects.GET("/:project_id", h.InternalDetail)
	}
}
