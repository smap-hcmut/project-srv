package http

import (
	"smap-project/internal/model"
	"smap-project/pkg/scope"

	"github.com/gin-gonic/gin"
)

func (h handler) processGetReq(c *gin.Context) (GetReq, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processGetReq: unauthorized")
		return GetReq{}, model.Scope{}, errUnauthorized
	}

	var req GetReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processGetReq.ShouldBindQuery: %v", err)
		return GetReq{}, model.Scope{}, errWrongQuery
	}
	req.Paginate.Adjust()

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "project.http.processGetReq.validate: %v", err)
		return GetReq{}, model.Scope{}, errInvalidStatus
	}

	return req, sc, nil
}

func (h handler) processDetailReq(c *gin.Context) (string, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processDetailReq: unauthorized")
		return "", model.Scope{}, errUnauthorized
	}

	id := c.Param("id")
	if id == "" {
		h.l.Errorf(ctx, "project.http.processDetailReq.InvalidID: %v", id)
		return "", model.Scope{}, errWrongBody
	}

	return id, sc, nil
}

func (h handler) processCreateReq(c *gin.Context) (CreateReq, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processCreateReq: unauthorized")
		return CreateReq{}, model.Scope{}, errUnauthorized
	}

	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processCreateReq.ShouldBindJSON: %v", err)
		return CreateReq{}, model.Scope{}, errWrongBody
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "project.http.processCreateReq.validate: %v", err)
		return CreateReq{}, model.Scope{}, errWrongBody
	}

	return req, sc, nil
}

func (h handler) processPatchReq(c *gin.Context) (PatchReq, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processUpdateReq: unauthorized")
		return PatchReq{}, model.Scope{}, errUnauthorized
	}

	var req PatchReq
	if err := c.ShouldBindUri(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processPatchReq.ShouldBindUri: %v", err)
		return PatchReq{}, model.Scope{}, errWrongBody
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processPatchReq.ShouldBindJSON: %v", err)
		return PatchReq{}, model.Scope{}, errWrongBody
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "project.http.processPatchReq.validate: %v", err)
		// Return appropriate error based on validation failure
		if err.Error() == "invalid status" {
			return PatchReq{}, model.Scope{}, errInvalidStatus
		}
		return PatchReq{}, model.Scope{}, errWrongBody
	}

	return req, sc, nil
}

func (h handler) processDeleteReq(c *gin.Context) (DeleteReq, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processDeleteReq: unauthorized")
		return DeleteReq{}, model.Scope{}, errUnauthorized
	}

	var req DeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processDeleteReq.ShouldBindJSON: %v", err)
		return DeleteReq{}, model.Scope{}, errWrongBody
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "project.http.processDeleteReq.validate: %v", err)
		return DeleteReq{}, model.Scope{}, errWrongBody
	}

	return req, sc, nil
}

func (h handler) processDryRunReq(c *gin.Context) (DryRunKeywordsReq, model.Scope, error) {
	ctx := c.Request.Context()
	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processDryRunReq: unauthorized")
		return DryRunKeywordsReq{}, model.Scope{}, errUnauthorized
	}

	var req DryRunKeywordsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "project.http.processDryRunReq.ShouldBindJSON: %v", err)
		return DryRunKeywordsReq{}, model.Scope{}, errWrongBody
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "project.http.processDryRunReq.validate: %v", err)
		return DryRunKeywordsReq{}, model.Scope{}, errWrongBody
	}

	return req, sc, nil
}

func (h handler) processProgressReq(c *gin.Context) (string, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processProgressReq: unauthorized")
		return "", model.Scope{}, errUnauthorized
	}

	id := c.Param("id")
	if id == "" {
		h.l.Errorf(ctx, "project.http.processProgressReq.InvalidID: %v", id)
		return "", model.Scope{}, errWrongBody
	}

	return id, sc, nil
}

func (h handler) processExecuteReq(c *gin.Context) (string, model.Scope, error) {
	ctx := c.Request.Context()

	sc, ok := scope.GetScopeFromContext(ctx)
	if !ok {
		h.l.Errorf(ctx, "project.http.processExecuteReq: unauthorized")
		return "", model.Scope{}, errUnauthorized
	}

	id := c.Param("id")
	if id == "" {
		h.l.Errorf(ctx, "project.http.processExecuteReq.InvalidID: %v", id)
		return "", model.Scope{}, errWrongBody
	}

	return id, sc, nil
}
