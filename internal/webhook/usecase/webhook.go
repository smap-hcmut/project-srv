// Package usecase implements the business logic for webhook callback processing.
// It handles transformation of crawler/collector callbacks into structured messages
// and publishes them to Redis for WebSocket delivery.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"smap-project/internal/webhook"
)

// HandleDryRunCallback processes dry-run job callbacks from the crawler service.
// It looks up the job mapping from Redis, transforms the callback into a JobMessage,
// and publishes it to the job-specific Redis topic.
//
// Topic pattern: job:{jobID}:{userID}
//
// The function performs the following steps:
//  1. Look up userID and projectID from Redis job mapping
//  2. Transform the callback request to a structured JobMessage
//  3. Publish the message to Redis for WebSocket delivery
//
// Returns an error if job mapping lookup fails, message marshaling fails,
// or Redis publish fails.
func (uc *usecase) HandleDryRunCallback(ctx context.Context, req webhook.CallbackRequest) error {
	// Look up UserID and ProjectID from Redis job mapping
	userID, projectID, err := uc.getJobMapping(ctx, req.JobID)
	if err != nil {
		uc.l.Errorf(ctx, "webhook.HandleDryRunCallback.getJobMapping failed: job_id=%s, platform=%s, status=%s, error=%v",
			req.JobID, req.Platform, req.Status, err)
		return err
	}

	// Log successful job mapping lookup with context
	uc.l.Infof(ctx, "webhook.HandleDryRunCallback: job mapping found: job_id=%s, user_id=%s, project_id=%s",
		req.JobID, userID, projectID)

	// Build job-specific topic pattern for Redis pub/sub
	channel := fmt.Sprintf("job:%s:%s", req.JobID, userID)

	// Transform callback to structured JobMessage
	message := uc.TransformDryRunCallback(req)

	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		uc.l.Errorf(ctx, "webhook.HandleDryRunCallback.Marshal failed: job_id=%s, message_type=JobMessage, error=%v",
			req.JobID, err)
		return fmt.Errorf("failed to marshal JobMessage: %w", err)
	}

	// Publish to Redis with structured logging
	if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
		uc.l.Errorf(ctx, "webhook.HandleDryRunCallback.Publish failed: topic=%s, job_id=%s, message_size=%d, error=%v",
			channel, req.JobID, len(body), err)
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	// Log successful publish with metrics for observability
	contentCount := 0
	if message.Batch != nil {
		contentCount = len(message.Batch.ContentList)
	}
	uc.l.Infof(ctx, "webhook.HandleDryRunCallback: published to Redis: topic=%s, job_id=%s, platform=%s, status=%s, content_count=%d, message_size=%d",
		channel, req.JobID, message.Platform, message.Status, contentCount, len(body))

	return nil
}

// HandleProgressCallback processes progress updates from the collector service.
// It validates the request, transforms it into a ProjectMessage, and publishes
// to the project-specific Redis topic.
//
// Supports both old flat format (deprecated) and new phase-based format.
// Format detection: if Crawl.Total > 0 or Analyze.Total > 0, use new format.
//
// Topic pattern: project:{projectID}:{userID}
//
// The function performs the following steps:
//  1. Validate required fields (projectID, userID)
//  2. Detect format (new phase-based vs old flat)
//  3. Transform the progress callback to a structured ProjectMessage
//  4. Build WebSocket message with phase structure
//  5. Publish the message to Redis for WebSocket delivery
//
// Returns an error if validation fails, message marshaling fails,
// or Redis publish fails.
func (uc *usecase) HandleProgressCallback(ctx context.Context, req webhook.ProgressCallbackRequest) error {
	// Validate required fields before processing
	if req.ProjectID == "" || req.UserID == "" {
		uc.l.Errorf(ctx, "webhook.HandleProgressCallback: validation failed: project_id=%s, user_id=%s",
			req.ProjectID, req.UserID)
		return fmt.Errorf("missing required fields: project_id or user_id")
	}

	// Detect format and log deprecation warning for old format
	isNewFormat := uc.isNewProgressFormat(req)
	if !isNewFormat {
		uc.l.Warnf(ctx, "webhook.HandleProgressCallback: DEPRECATED old format detected for project_id=%s, please migrate to phase-based format",
			req.ProjectID)
	}

	// Build project-specific topic pattern for Redis pub/sub
	channel := fmt.Sprintf("project:%s:%s", req.ProjectID, req.UserID)

	// Determine message type based on status
	messageType := webhook.MessageTypeProjectProgress
	if req.Status == "DONE" || req.Status == "FAILED" {
		messageType = webhook.MessageTypeProjectCompleted
	}

	// Build WebSocket message with phase structure
	wsMessage := uc.buildProgressWebSocketMessage(req, messageType)

	// Marshal message to JSON
	body, err := json.Marshal(wsMessage)
	if err != nil {
		uc.l.Errorf(ctx, "webhook.HandleProgressCallback.Marshal failed: project_id=%s, message_type=%s, error=%v",
			req.ProjectID, messageType, err)
		return fmt.Errorf("failed to marshal WebSocket message: %w", err)
	}

	// Publish to Redis with structured logging
	if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
		uc.l.Errorf(ctx, "webhook.HandleProgressCallback.Publish failed: topic=%s, project_id=%s, message_size=%d, error=%v",
			channel, req.ProjectID, len(body), err)
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	// Log successful publish with metrics for observability
	uc.l.Infof(ctx, "webhook.HandleProgressCallback: published to Redis: topic=%s, project_id=%s, status=%s, format=%s, overall_progress=%.1f%%, message_size=%d",
		channel, req.ProjectID, req.Status, uc.formatType(isNewFormat), uc.getOverallProgress(req), len(body))

	return nil
}

// buildProgressWebSocketMessage builds the WebSocket message with phase structure.
// This is the message format that will be sent to connected clients.
func (uc *usecase) buildProgressWebSocketMessage(req webhook.ProgressCallbackRequest, messageType string) map[string]interface{} {
	// Build phase progress objects
	crawl := map[string]interface{}{
		"total":            req.Crawl.Total,
		"done":             req.Crawl.Done,
		"errors":           req.Crawl.Errors,
		"progress_percent": req.Crawl.ProgressPercent,
	}

	analyze := map[string]interface{}{
		"total":            req.Analyze.Total,
		"done":             req.Analyze.Done,
		"errors":           req.Analyze.Errors,
		"progress_percent": req.Analyze.ProgressPercent,
	}

	// Calculate overall progress
	overallProgress := req.OverallProgressPercent
	if overallProgress == 0 {
		overallProgress = uc.getOverallProgress(req)
	}

	return map[string]interface{}{
		"type": messageType,
		"payload": map[string]interface{}{
			"project_id":               req.ProjectID,
			"status":                   req.Status,
			"crawl":                    crawl,
			"analyze":                  analyze,
			"overall_progress_percent": overallProgress,
		},
	}
}

// getOverallProgress calculates overall progress from request data.
func (uc *usecase) getOverallProgress(req webhook.ProgressCallbackRequest) float64 {
	if req.OverallProgressPercent > 0 {
		return req.OverallProgressPercent
	}
	// Calculate from phases
	crawlProgress := uc.calculatePhaseProgress(req.Crawl)
	analyzeProgress := uc.calculatePhaseProgress(req.Analyze)
	return (crawlProgress + analyzeProgress) / 2
}

// formatType returns format type string for logging.
func (uc *usecase) formatType(isNewFormat bool) string {
	if isNewFormat {
		return "phase-based"
	}
	return "flat(deprecated)"
}
