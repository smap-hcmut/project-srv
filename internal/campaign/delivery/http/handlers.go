package http

import (
	"project-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// @Summary Create a campaign
// @Description Create a new campaign
// @Tags Campaign
// @Accept json
// @Produce json
// @Param body body createReq true "Create campaign request"
// @Success 200 {object} createResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns [post]
func (h *handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "campaign.delivery.Create.processCreateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Create(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "campaign.delivery.Create.uc.Create: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateResp(o))
}

// @Summary Get campaign detail
// @Description Return campaign info by ID
// @Tags Campaign
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} detailResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns/{id} [get]
func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	o, err := h.uc.Detail(ctx, id)
	if err != nil {
		h.l.Errorf(ctx, "campaign.delivery.Detail.uc.Detail: id=%s err=%v", id, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(o))
}

// @Summary List campaigns
// @Description Paginate campaigns with optional status filter
// @Tags Campaign
// @Produce json
// @Param status query string false "Filter by status (ACTIVE, INACTIVE, ARCHIVED)"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Number of records per page (default 15)"
// @Success 200 {object} listResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns [get]
func (h *handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processListReq(c)
	if err != nil {
		h.l.Warnf(ctx, "campaign.delivery.List.processListReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.List(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "campaign.delivery.List.uc.List: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListResp(o))
}

// @Summary Update a campaign
// @Description Update campaign fields by ID
// @Tags Campaign
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body updateReq true "Update campaign request"
// @Success 200 {object} updateResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns/{id} [put]
func (h *handler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "campaign.delivery.Update.processUpdateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Update(ctx, req.toInput(c.Param("id")))
	if err != nil {
		h.l.Errorf(ctx, "campaign.delivery.Update.uc.Update: id=%s err=%v", c.Param("id"), err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateResp(o))
}

// @Summary Archive a campaign
// @Description Soft-delete a campaign by ID
// @Tags Campaign
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns/{id} [delete]
func (h *handler) Archive(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := h.uc.Archive(ctx, id); err != nil {
		h.l.Errorf(ctx, "campaign.delivery.Archive.uc.Archive: id=%s err=%v", id, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
