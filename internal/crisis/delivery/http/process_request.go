package http

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"
)

// processUpsertReq binds and validates the upsert request.
func (h *handler) processUpsertReq(c *gin.Context) (upsertReq, error) {
	var req upsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "crisis.delivery.processUpsertReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "crisis.delivery.processUpsertReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

func (h *handler) processApplyRuntimeReq(c *gin.Context) (applyRuntimeReq, error) {
	var req applyRuntimeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			h.l.Warnf(c.Request.Context(), "crisis.delivery.processApplyRuntimeReq.ShouldBindJSON: %v", err)
			return req, errWrongBody
		}
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "crisis.delivery.processApplyRuntimeReq.validate: %v", err)
		return req, err
	}
	return req, nil
}
