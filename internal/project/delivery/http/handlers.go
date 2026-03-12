package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// @Summary Create a project
// @Description Create a new project under a campaign
// @Tags Project
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body createReq true "Create project request"
// @Success 200 {object} createResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns/{id}/projects [post]
func (h *handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "project.delivery.Create.processCreateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Create(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "project.delivery.Create.uc.Create: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateResp(o))
}

// @Summary Get project detail
// @Description Return project info by ID
// @Tags Project
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {object} detailResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{projectId} [get]
func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDetailReq(c)
	if err != nil {
		h.l.Warnf(ctx, "project.delivery.Detail.processDetailReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Detail(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "project.delivery.Detail.uc.Detail: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(o))
}

// @Summary List projects
// @Description Paginate projects under a campaign with filters
// @Tags Project
// @Produce json
// @Param id path string true "Campaign ID"
// @Param status query string false "Filter by status (ACTIVE, PAUSED, ARCHIVED)"
// @Param name query string false "Filter by name (ILIKE)"
// @Param brand query string false "Filter by brand (ILIKE)"
// @Param entity_type query string false "Filter by entity type"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Number of records per page (default 15)"
// @Success 200 {object} listResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /campaigns/{id}/projects [get]
func (h *handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processListReq(c)
	if err != nil {
		h.l.Warnf(ctx, "project.delivery.List.processListReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.List(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "project.delivery.List.uc.List: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListResp(o))
}

// @Summary Update a project
// @Description Update project fields by ID
// @Tags Project
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param body body updateReq true "Update project request"
// @Success 200 {object} updateResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{projectId} [put]
func (h *handler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "project.delivery.Update.processUpdateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Update(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "project.delivery.Update.uc.Update: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateResp(o))
}

// @Summary Archive a project
// @Description Soft-delete a project by ID
// @Tags Project
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{projectId} [delete]
func (h *handler) Archive(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processArchiveReq(c)
	if err != nil {
		h.l.Warnf(ctx, "project.delivery.Archive.processArchiveReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	if err := h.uc.Archive(ctx, req.toInput()); err != nil {
		h.l.Errorf(ctx, "project.delivery.Archive.uc.Archive: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
