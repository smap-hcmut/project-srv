package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	projects := r.Group("/projects")
	projects.Use(mw.Auth())
	{
		projects.GET("/:project_id/ontology-rules", h.Detail)
		projects.PUT("/:project_id/ontology-rules", h.Upsert)
		projects.POST("/:project_id/ontology-rules/test", h.Test)
		projects.DELETE("/:project_id/ontology-rules", h.Delete)
	}

	internalProjects := r.Group("/internal/projects")
	internalProjects.Use(mw.InternalAuth())
	{
		internalProjects.GET("/:project_id/ontology-rules", h.Runtime)
	}
}
