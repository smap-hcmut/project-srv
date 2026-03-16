package http

import (
	"github.com/gin-gonic/gin"
)

// processCreateReq binds, validates, and extracts path params for create.
func (h *handler) processCreateReq(c *gin.Context) (createReq, error) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processCreateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody	
	}
	req.CampaignID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processCreateReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

// processDetailReq extracts path param for detail.
func (h *handler) processDetailReq(c *gin.Context) (detailReq, error) {
	return detailReq{ID: c.Param("project_id")}, nil
}

// processListReq binds query params and extracts path param for listing.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processListReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
	}
	req.CampaignID = c.Param("id")
	return req, nil
}

// processUpdateReq binds, validates, and extracts path param for update.
func (h *handler) processUpdateReq(c *gin.Context) (updateReq, error) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processUpdateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.ID = c.Param("project_id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processUpdateReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

// processArchiveReq extracts path param for archive.
func (h *handler) processArchiveReq(c *gin.Context) (archiveReq, error) {
	return archiveReq{ID: c.Param("project_id")}, nil
}
