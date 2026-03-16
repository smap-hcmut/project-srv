package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// @Summary Upsert crisis config
// @Description Create or update crisis detection config for a project
// @Tags CrisisConfig
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID"
// @Param body body upsertReq true "Crisis config data"
// @Success 200 {object} upsertResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{project_id}/crisis-config [put]
func (h *handler) Upsert(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpsertReq(c)
	if err != nil {
		h.l.Warnf(ctx, "crisis.delivery.Upsert.processUpsertReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	projectID := c.Param("project_id")
	o, err := h.uc.Upsert(ctx, req.toInput(projectID))
	if err != nil {
		h.l.Errorf(ctx, "crisis.delivery.Upsert.uc.Upsert: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpsertResp(o))
}

// @Summary Get crisis config
// @Description Return crisis detection config for a project
// @Tags CrisisConfig
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} detailResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{project_id}/crisis-config [get]
func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := c.Param("project_id")

	o, err := h.uc.Detail(ctx, projectID)
	if err != nil {
		h.l.Errorf(ctx, "crisis.delivery.Detail.uc.Detail: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(o))
}

// @Summary Delete crisis config
// @Description Remove crisis detection config for a project
// @Tags CrisisConfig
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /projects/{project_id}/crisis-config [delete]
func (h *handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := c.Param("project_id")

	if err := h.uc.Delete(ctx, projectID); err != nil {
		h.l.Errorf(ctx, "crisis.delivery.Delete.uc.Delete: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
