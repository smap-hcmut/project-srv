package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"smap-project/internal/webhook"
	"time"

	"github.com/redis/go-redis/v9"
)

// storeJobMapping stores a job mapping in Redis with TTL
// Key pattern: job:mapping:{jobID}
// Value: JSON with userID, projectID, platform, createdAt
// Note: projectID can be empty for dry-run jobs
func (uc *usecase) storeJobMapping(ctx context.Context, jobID, userID, projectID string, ttl time.Duration) error {
	if jobID == "" {
		return fmt.Errorf("jobID cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}

	// Create the mapping data
	data := webhook.JobMappingData{
		UserID:    userID,
		ProjectID: projectID,
		Platform:  "", // Platform will be set when available
		CreatedAt: time.Now().UTC(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.storeJobMapping.Marshal: jobID=%s, error=%v", jobID, err)
		return fmt.Errorf("failed to marshal job mapping: %w", err)
	}

	// Store in Redis with key pattern: job:mapping:{jobID}
	key := fmt.Sprintf("job:mapping:%s", jobID)
	expirationSeconds := int(ttl.Seconds())

	if err := uc.redisClient.Set(ctx, key, jsonData, expirationSeconds); err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.storeJobMapping.Set: jobID=%s, error=%v", jobID, err)
		return fmt.Errorf("failed to store job mapping in Redis: %w", err)
	}

	uc.l.Infof(ctx, "Stored job mapping: jobID=%s, userID=%s, projectID=%s, ttl=%v", jobID, userID, projectID, ttl)
	return nil
}

// getJobMapping retrieves a job mapping from Redis
// Returns userID, projectID, and error if not found
func (uc *usecase) getJobMapping(ctx context.Context, jobID string) (userID, projectID string, err error) {
	if jobID == "" {
		return "", "", fmt.Errorf("jobID cannot be empty")
	}

	// Retrieve from Redis with key pattern: job:mapping:{jobID}
	key := fmt.Sprintf("job:mapping:%s", jobID)
	jsonData, err := uc.redisClient.Get(ctx, key)
	if err != nil {
		// Check if it's a "key not found" error
		if err == redis.Nil {
			uc.l.Warnf(ctx, "Job mapping not found: jobID=%s", jobID)
			return "", "", fmt.Errorf("job mapping not found for jobID: %s", jobID)
		}
		uc.l.Errorf(ctx, "internal.webhook.usecase.getJobMapping.Get: jobID=%s, error=%v", jobID, err)
		return "", "", fmt.Errorf("failed to retrieve job mapping from Redis: %w", err)
	}

	// Parse JSON
	var data webhook.JobMappingData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		uc.l.Errorf(ctx, "internal.webhook.usecase.getJobMapping.Unmarshal: jobID=%s, error=%v", jobID, err)
		return "", "", fmt.Errorf("failed to unmarshal job mapping: %w", err)
	}

	uc.l.Infof(ctx, "Retrieved job mapping: jobID=%s, userID=%s, projectID=%s", jobID, data.UserID, data.ProjectID)
	return data.UserID, data.ProjectID, nil
}

// StoreJobMapping is a public method for storing job mappings
// This will be called from project use case when creating jobs
// Uses a default TTL of 7 days
func (uc *usecase) StoreJobMapping(ctx context.Context, jobID, userID, projectID string) error {
	const defaultTTL = 7 * 24 * time.Hour // 7 days
	return uc.storeJobMapping(ctx, jobID, userID, projectID, defaultTTL)
}
