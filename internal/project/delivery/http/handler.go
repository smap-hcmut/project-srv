package http

import (
	"slices"

	"smap-project/pkg/response"

	"github.com/gin-gonic/gin"
)

// @Summary Get project detail
// @Description Get a single project by ID
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Project ID"
// @Success 200 {object} ProjectResp
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects/{id} [get]
func (h handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processDetailReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Detail.processDetailReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Detail(ctx, sc, req)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Detail.Detail: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Detail.Detail: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, h.newProjectResp(o))
}

// @Summary Get projects with pagination
// @Description Get projects for the authenticated user with pagination
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param ids query []string false "Filter by project IDs"
// @Param statuses query []string false "Filter by statuses"
// @Param search_name query string false "Search by project name"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} GetResp
// @Failure 400 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects [get]
func (h handler) Get(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Get.processGetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	ip := req.toInput()
	o, err := h.uc.Get(ctx, sc, ip)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Get.Get: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Get.Get: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, h.newGetResp(o))
}

// @Summary Create a new project
// @Description Create a new project for the authenticated user
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body CreateReq true "Project creation request"
// @Success 201 {object} ProjectResp
// @Failure 400 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects [post]
func (h handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processCreateReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Create.processCreateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	ip := req.toInput()
	o, err := h.uc.Create(ctx, sc, ip)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Create.Create: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Create.Create: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, h.newProjectResp(o))
}

// @Summary Patch a project
// @Description Patch an existing project
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Project ID"
// @Param request body PatchReq true "Project patch request"
// @Success 200 {object} ProjectResp
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects/{id} [patch]
func (h handler) Patch(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processPatchReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Patch.processPatchReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	ip := req.toInput()
	_, err = h.uc.Patch(ctx, sc, ip)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Patch.Patch: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Patch.Patch: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, nil)
}

// @Summary Delete a project
// @Description Soft delete a project (sets deleted_at timestamp)
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body DeleteReq true "Project deletion request"
// @Success 204 "No Content"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects [delete]
func (h handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processDeleteReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Delete.processDeleteReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	ip := req.toInput()
	err = h.uc.Delete(ctx, sc, ip)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Delete.Delete: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Delete.Delete: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, nil)
}

// @Summary Get project progress (legacy)
// @Description Get real-time progress of a project's processing status (legacy flat format)
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Project ID"
// @Success 200 {object} ProgressResp
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects/{id}/progress [get]
func (h handler) GetProgress(c *gin.Context) {
	ctx := c.Request.Context()

	projectID, sc, err := h.processProgressReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.GetProgress.processProgressReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetProgress(ctx, sc, projectID)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.GetProgress.GetProgress: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.GetProgress.GetProgress: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	// Set Cache-Control header to prevent caching
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	response.OK(c, h.newProgressResp(o))
}

// @Summary Get project phase progress
// @Description Get real-time phase-based progress of a project's processing status (new format with crawl/analyze phases)
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Project ID"
// @Success 200 {object} ProjectProgressResp
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects/{id}/phase-progress [get]
func (h handler) GetPhaseProgress(c *gin.Context) {
	ctx := c.Request.Context()

	projectID, sc, err := h.processProgressReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.GetPhaseProgress.processProgressReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetPhaseProgress(ctx, sc, projectID)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.GetPhaseProgress.GetPhaseProgress: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.GetPhaseProgress.GetPhaseProgress: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	// Set Cache-Control header to prevent caching
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	response.OK(c, h.newProjectProgressResp(o))
}

// @Summary Execute a project
// @Description Start processing for an existing project (init Redis state + publish event)
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Project ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError "Project already executing"
// @Failure 500 {object} errors.HTTPError
// @Router /projects/{id}/execute [post]
func (h handler) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	projectID, sc, err := h.processExecuteReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.Execute.processExecuteReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	err = h.uc.Execute(ctx, sc, projectID)
	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.Execute.Execute: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.Execute.Execute: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, nil)
}

// @Summary Dry run keywords
// @Description Perform a dry run for the provided keywords
// @Tags Projects
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body DryRunKeywordsReq true "Dry run keywords request"
// @Success 200 {object} DryRunJobResp
// @Failure 400 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /projects/dryrun [post]
func (h handler) DryRunKeywords(c *gin.Context) {
	ctx := c.Request.Context()
	req, sc, err := h.processDryRunReq(c)
	if err != nil {
		h.l.Errorf(ctx, "project.http.DryRunKeywords.processDryRunReq: %v", err)
		response.Error(c, err, h.discord)
		return

	}

	output, err := h.uc.DryRunKeywords(ctx, sc, req.toInput())

	if err != nil {
		err = h.mapErrorCode(err)
		if !slices.Contains(NotFound, err) {
			h.l.Errorf(ctx, "project.http.DryRunKeywords.DryRunKeywords: %v", err)
		} else {
			h.l.Warnf(ctx, "project.http.DryRunKeywords.DryRunKeywords: %v", err)
		}
		response.Error(c, err, h.discord)
		return
	}

	response.OK(c, toDryRunJobResp(output))
}
