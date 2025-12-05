package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"smap-project/internal/model"
	"smap-project/internal/webhook"
)

func (uc *usecase) HandleDryRunCallback(ctx context.Context, req webhook.CallbackRequest) error {
	// Always look up UserID and ProjectID from Redis (no fallback)
	userID, projectID, err := uc.getJobMapping(ctx, req.JobID)
	if err != nil {
		// Log error with JobID, Platform, and Status (Requirement 5.1, 5.3)
		uc.l.Errorf(ctx, "Failed to get job mapping: job_id=%s, platform=%s, status=%s, error=%v", req.JobID, req.Platform, req.Status, err)
		return err
	}

	// Successfully looked up from Redis
	uc.l.Infof(ctx, "Successfully looked up job mapping: job_id=%s, user_id=%s, project_id=%s", req.JobID, userID, projectID)

	// Format Redis channel as user_noti:{user_id}
	channel := fmt.Sprintf("user_noti:%s", userID)

	// Construct message with type="dryrun_result"
	// Note: job_id, platform, status must be inside payload because
	// websocket subscriber only extracts "type" and "payload" fields
	message := map[string]interface{}{
		"type": webhook.MessageTypeDryRunResult,
		"payload": map[string]interface{}{
			"job_id":   req.JobID,
			"platform": req.Platform,
			"status":   req.Status,
			"content":  req.Payload.Content,
			"errors":   req.Payload.Errors,
		},
	}

	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.HandleDryRunCallback.Marshal: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to Redis
	if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.HandleDryRunCallback.Publish: %v", err)
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	uc.l.Infof(ctx, "Published dry-run result to Redis: channel=%s, job_id=%s, platform=%s, status=%s", channel, req.JobID, req.Platform, req.Status)

	return nil
}

// HandleProgressCallback handles progress updates from collector service
// and publishes to WebSocket via Redis Pub/Sub
func (uc *usecase) HandleProgressCallback(ctx context.Context, req webhook.ProgressCallbackRequest) error {
	// Format Redis channel as user_noti:{user_id}
	channel := fmt.Sprintf("user_noti:%s", req.UserID)

	// Calculate progress percentage
	var progressPercent float64
	if req.Total > 0 {
		progressPercent = float64(req.Done) / float64(req.Total) * 100
	}

	// Determine message type based on status
	messageType := webhook.MessageTypeProjectProgress
	status := model.ProjectStatus(req.Status)
	if status == model.ProjectStatusDone || status == model.ProjectStatusFailed {
		messageType = webhook.MessageTypeProjectCompleted
	}

	// Construct message
	message := map[string]interface{}{
		"type": messageType,
		"payload": map[string]interface{}{
			"project_id":       req.ProjectID,
			"status":           req.Status,
			"total":            req.Total,
			"done":             req.Done,
			"errors":           req.Errors,
			"progress_percent": progressPercent,
		},
	}

	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.HandleProgressCallback.Marshal: %v", err)
		return err
	}

	// Publish to Redis
	if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.HandleProgressCallback.Publish: %v", err)
		return err
	}

	return nil
}
