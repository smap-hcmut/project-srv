package http

import (
	"smap-project/internal/webhook"

	"github.com/gin-gonic/gin"
)

func (h handler) processCallbackReq(c *gin.Context) (webhook.CallbackRequest, error) {
	ctx := c.Request.Context()
	// Bind request body
	var req CallbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "webhook.http.processCallbackReq.ShouldBindJSON: %v", err)
		return webhook.CallbackRequest{}, errWrongBody
	}

	return req.toInput(), nil
}

func (h handler) processProgressCallbackReq(c *gin.Context) (webhook.ProgressCallbackRequest, error) {
	ctx := c.Request.Context()
	// Bind request body
	var req webhook.ProgressCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "webhook.http.processProgressCallbackReq.ShouldBindJSON: %v", err)
		return webhook.ProgressCallbackRequest{}, errWrongBody
	}

	return req, nil
}
