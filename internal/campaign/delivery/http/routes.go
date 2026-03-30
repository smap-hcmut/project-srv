package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes maps campaign routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	campaigns := r.Group("/campaigns")
	campaigns.Use(mw.Auth())
	{
		campaigns.POST("", h.Create)
		campaigns.GET("/favorites", h.ListFavorites)
		campaigns.GET("", h.List)
		campaigns.GET("/:id", h.Detail)
		campaigns.PUT("/:id", h.Update)
		campaigns.POST("/:id/favorite", h.Favorite)
		campaigns.DELETE("/:id/favorite", h.Unfavorite)
		campaigns.DELETE("/:id", h.Archive)
	}
}
