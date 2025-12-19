package http

import (
	"smap-project/internal/webhook"
)

// CallbackReq represents the HTTP request for webhook callback
type CallbackReq struct {
	JobID    string                  `json:"job_id" binding:"required"`
	Status   string                  `json:"status" binding:"required"`
	Platform string                  `json:"platform" binding:"required"`
	Payload  webhook.CallbackPayload `json:"payload"`
	UserID   string                  `json:"user_id,omitempty"`
}

func (r CallbackReq) toInput() webhook.CallbackRequest {
	return webhook.CallbackRequest{
		JobID:    r.JobID,
		Status:   r.Status,
		Platform: r.Platform,
		Payload:  r.Payload,
		UserID:   r.UserID,
	}
}

// CallbackResp represents the HTTP response for webhook callback
