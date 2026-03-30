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
	req := detailReq{ID: c.Param("project_id")}
	if err := req.validate(); err != nil {
		return req, errWrongBody
	}
	return req, nil
}

// processListReq binds query params and extracts path param for listing.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processListReq.ShouldBindQuery: %v", err)
		return req, errWrongQuery
	}
	req.CampaignID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processListReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

// processFavoriteListReq binds query params for global favorite project listing.
func (h *handler) processFavoriteListReq(c *gin.Context) (favoriteListReq, error) {
	var req favoriteListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processFavoriteListReq.ShouldBindQuery: %v", err)
		return req, errWrongQuery
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processFavoriteListReq.validate: %v", err)
		return req, err
	}
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
	req := archiveReq{ID: c.Param("project_id")}
	if err := req.validate(); err != nil {
		return req, errWrongBody
	}
	return req, nil
}

// processLifecycleReq extracts path param for lifecycle actions.
func (h *handler) processLifecycleReq(c *gin.Context) (archiveReq, error) {
	req := archiveReq{ID: c.Param("project_id")}
	if err := req.validate(); err != nil {
		return req, errWrongBody
	}
	return req, nil
}

func (h *handler) processActivationReadinessReq(c *gin.Context) (activationReadinessReq, error) {
	var req activationReadinessReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processActivationReadinessReq.ShouldBindQuery: %v", err)
		return req, errWrongQuery
	}
	req.ID = c.Param("project_id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "project.delivery.processActivationReadinessReq.validate: %v", err)
		return req, errWrongBody
	}
	return req, nil
}
