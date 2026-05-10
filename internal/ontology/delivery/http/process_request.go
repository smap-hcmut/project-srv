package http

import "github.com/gin-gonic/gin"

func (h *handler) processUpsertReq(c *gin.Context) (upsertReq, error) {
	var req upsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "ontology.delivery.processUpsertReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	return req, nil
}

func (h *handler) processTestReq(c *gin.Context) (testReq, error) {
	var req testReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "ontology.delivery.processTestReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	return req, nil
}
