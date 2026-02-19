package http

import (
	"project-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps campaign routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	campaigns := r.Group("/campaigns")
	campaigns.Use(mw.Auth())
	{
		campaigns.POST("", h.Create)
		campaigns.GET("", h.List)
		campaigns.GET("/:id", h.Detail)
		campaigns.PUT("/:id", h.Update)
		campaigns.DELETE("/:id", h.Archive)
	}
}
