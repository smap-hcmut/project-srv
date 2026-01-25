package http

import (
	"smap-project/internal/middleware"

	"github.com/gin-gonic/gin"
)

func MapProjectRoutes(r *gin.RouterGroup, h handler, mw middleware.Middleware) {
	// All routes require authentication
	r.GET("", mw.Auth(), h.Get)
	r.GET("/:id", mw.Auth(), h.Detail)
	r.GET("/:id/progress", mw.Auth(), h.GetProgress)
	r.GET("/:id/phase-progress", mw.Auth(), h.GetPhaseProgress)
	r.POST("", mw.Auth(), h.Create)
	r.POST("/:id/execute", mw.Auth(), h.Execute)
	r.PATCH("/:id", mw.Auth(), h.Patch)
	r.DELETE("", mw.Auth(), h.Delete)
	r.POST("/dryrun", mw.Auth(), h.DryRunKeywords)
}
