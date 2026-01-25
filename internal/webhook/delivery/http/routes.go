package http

import (
	"smap-project/internal/middleware"

	"github.com/gin-gonic/gin"
)

func MapWebhookRoutes(r *gin.RouterGroup, h handler, mw middleware.Middleware) {
	// Webhook routes require internal authentication
	r.POST("/dryrun/callback", mw.InternalAuth(), h.DryRunCallback)
	r.POST("/progress/callback", mw.InternalAuth(), h.ProgressCallback)
}
