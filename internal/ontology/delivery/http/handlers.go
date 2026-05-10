package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
)

func (h *handler) Upsert(c *gin.Context) {
	ctx := c.Request.Context()
	req, err := h.processUpsertReq(c)
	if err != nil {
		response.Error(c, err, h.discord)
		return
	}

	projectID := c.Param("project_id")
	o, err := h.uc.Upsert(ctx, req.toInput(projectID))
	if err != nil {
		h.l.Errorf(ctx, "ontology.delivery.Upsert.uc.Upsert: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpsertResp(o))
}

func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := c.Param("project_id")

	o, err := h.uc.Detail(ctx, projectID)
	if err != nil {
		h.l.Errorf(ctx, "ontology.delivery.Detail.uc.Detail: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(o))
}

func (h *handler) Runtime(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := c.Param("project_id")

	o, err := h.uc.Runtime(ctx, projectID)
	if err != nil {
		h.l.Errorf(ctx, "ontology.delivery.Runtime.uc.Runtime: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newRuntimeResp(o))
}

func (h *handler) Test(c *gin.Context) {
	ctx := c.Request.Context()
	req, err := h.processTestReq(c)
	if err != nil {
		response.Error(c, err, h.discord)
		return
	}

	projectID := c.Param("project_id")
	o, err := h.uc.Test(ctx, req.toInput(projectID))
	if err != nil {
		h.l.Warnf(ctx, "ontology.delivery.Test.uc.Test: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newTestResp(o))
}

func (h *handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := c.Param("project_id")
	if err := h.uc.Delete(ctx, projectID); err != nil {
		h.l.Errorf(ctx, "ontology.delivery.Delete.uc.Delete: project_id=%s err=%v", projectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}
	response.OK(c, nil)
}
