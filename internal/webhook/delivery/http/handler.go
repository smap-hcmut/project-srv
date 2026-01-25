package http

import (
	"smap-project/pkg/response"

	"github.com/gin-gonic/gin"
)

// DryRunCallback handles dry-run webhook callbacks from collector
func (h handler) DryRunCallback(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCallbackReq(c)
	if err != nil {
		h.l.Errorf(ctx, "webhook.http.DryRunCallback.processCallbackReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	// Handle callback
	if err := h.uc.HandleDryRunCallback(ctx, req); err != nil {
		err = h.mapErrorCode(err)
		h.l.Errorf(ctx, "webhook.http.DryRunCallback.HandleDryRunCallback: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	h.l.Infof(ctx, "Webhook callback processed successfully: job_id=%s, platform=%s, status=%s",
		req.JobID, req.Platform, req.Status)

	response.OK(c, nil)
}

// ProgressCallback handles progress webhook callbacks from collector
// @Summary Progress webhook callback
// @Description Receive progress updates from collector service and publish to WebSocket
// @Tags Internal
// @Accept json
// @Produce json
// @Param X-Internal-Key header string true "Internal API Key"
// @Param request body webhook.ProgressCallbackRequest true "Progress callback request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /internal/collector/progress/callback [post]
func (h handler) ProgressCallback(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processProgressCallbackReq(c)
	if err != nil {
		h.l.Errorf(ctx, "webhook.http.ProgressCallback.processProgressCallbackReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	// Handle callback
	if err := h.uc.HandleProgressCallback(ctx, req); err != nil {
		err = h.mapErrorCode(err)
		h.l.Errorf(ctx, "webhook.http.ProgressCallback.HandleProgressCallback: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	h.l.Infof(ctx, "Progress callback processed: project_id=%s, status=%s, done=%d/%d",
		req.ProjectID, req.Status, req.Done, req.Total)

	response.OK(c, map[string]string{"status": "ok"})
}
