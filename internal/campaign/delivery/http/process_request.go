package http

import (
	"github.com/gin-gonic/gin"
)

// processCreateReq binds and validates the create request.
func (h *handler) processCreateReq(c *gin.Context) (createReq, error) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processCreateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processCreateReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

// processUpdateReq binds and validates the update request.
func (h *handler) processUpdateReq(c *gin.Context) (updateReq, error) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processUpdateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processUpdateReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

// processListReq binds query params for listing.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processListReq.ShouldBindQuery: %v", err)
		return req, errWrongQuery
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "campaign.delivery.processListReq.validate: %v", err)
		return req, err
	}
	return req, nil
}
