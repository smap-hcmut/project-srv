package redis

import (
	"context"
	"strconv"
	"time"

	"smap-project/internal/model"
)

const (
	keyPrefix       = "smap:proj:"
	fieldStatus     = "status"
	stateTTLSeconds = 7 * 24 * 60 * 60 // 604800 seconds

	// Crawl phase fields
	fieldCrawlTotal  = "crawl_total"
	fieldCrawlDone   = "crawl_done"
	fieldCrawlErrors = "crawl_errors"

	// Analyze phase fields
	fieldAnalyzeTotal  = "analyze_total"
	fieldAnalyzeDone   = "analyze_done"
	fieldAnalyzeErrors = "analyze_errors"

	// Old flat format fields (deprecated, kept for backward compatibility)
	fieldTotal  = "total"
	fieldDone   = "done"
	fieldErrors = "errors"
)

// buildKey constructs the Redis key for a project's state.
func buildKey(projectID string) string {
	return keyPrefix + projectID
}

func (r *redisStateRepository) InitState(ctx context.Context, projectID string, state model.ProjectState) error {
	key := buildKey(projectID)

	// Use pipeline for atomic operation
	pipe := r.client.Pipeline()

	// Set status
	pipe.HSet(ctx, key, fieldStatus, string(state.Status))

	// Set crawl phase fields
	pipe.HSet(ctx, key, fieldCrawlTotal, state.CrawlTotal)
	pipe.HSet(ctx, key, fieldCrawlDone, state.CrawlDone)
	pipe.HSet(ctx, key, fieldCrawlErrors, state.CrawlErrors)

	// Set analyze phase fields
	pipe.HSet(ctx, key, fieldAnalyzeTotal, state.AnalyzeTotal)
	pipe.HSet(ctx, key, fieldAnalyzeDone, state.AnalyzeDone)
	pipe.HSet(ctx, key, fieldAnalyzeErrors, state.AnalyzeErrors)

	// Set old flat format fields for backward compatibility
	pipe.HSet(ctx, key, fieldTotal, state.Total)
	pipe.HSet(ctx, key, fieldDone, state.Done)
	pipe.HSet(ctx, key, fieldErrors, state.Errors)

	// Set TTL
	pipe.Expire(ctx, key, time.Duration(stateTTLSeconds)*time.Second)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.InitState: failed for project %s: %v", projectID, err)
		return err
	}

	r.logger.Infof(ctx, "state.repository.redis.InitState: initialized state for project %s", projectID)
	return nil
}

// GetState retrieves the current state of a project.
func (r *redisStateRepository) GetState(ctx context.Context, projectID string) (*model.ProjectState, error) {
	key := buildKey(projectID)

	data, err := r.client.HGetAll(ctx, key)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.GetState: failed for project %s: %v", projectID, err)
		return nil, err
	}

	// If no data, return nil (key doesn't exist)
	if len(data) == 0 {
		return nil, nil
	}

	// Parse fields
	s := &model.ProjectState{
		Status: model.ProjectStatus(data[fieldStatus]),
	}

	// Parse crawl phase fields
	if v, ok := data[fieldCrawlTotal]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.CrawlTotal = val
		}
	}
	if v, ok := data[fieldCrawlDone]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.CrawlDone = val
		}
	}
	if v, ok := data[fieldCrawlErrors]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.CrawlErrors = val
		}
	}

	// Parse analyze phase fields
	if v, ok := data[fieldAnalyzeTotal]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.AnalyzeTotal = val
		}
	}
	if v, ok := data[fieldAnalyzeDone]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.AnalyzeDone = val
		}
	}
	if v, ok := data[fieldAnalyzeErrors]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.AnalyzeErrors = val
		}
	}

	// Parse old flat format fields for backward compatibility
	if v, ok := data[fieldTotal]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.Total = val
		}
	}
	if v, ok := data[fieldDone]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.Done = val
		}
	}
	if v, ok := data[fieldErrors]; ok {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			s.Errors = val
		}
	}

	return s, nil
}

// SetStatus updates the status field.
func (r *redisStateRepository) SetStatus(ctx context.Context, projectID string, status model.ProjectStatus) error {
	key := buildKey(projectID)

	// Use pipeline for atomic status update + TTL refresh
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fieldStatus, string(status))
	pipe.Expire(ctx, key, time.Duration(stateTTLSeconds)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.SetStatus: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// SetTotal sets the total number of items.
func (r *redisStateRepository) SetTotal(ctx context.Context, projectID string, total int64) error {
	key := buildKey(projectID)

	// Use pipeline for atomic total update + TTL refresh
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fieldTotal, total)
	pipe.Expire(ctx, key, time.Duration(stateTTLSeconds)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.SetTotal: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// IncrementDone atomically increments the done counter.
// Returns the new done count after increment.
func (r *redisStateRepository) IncrementDone(ctx context.Context, projectID string) (int64, error) {
	key := buildKey(projectID)

	// Atomically increment done counter
	newDone, err := r.client.HIncrBy(ctx, key, fieldDone, 1)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.IncrementDone: failed for project %s: %v", projectID, err)
		return 0, err
	}

	return newDone, nil
}

// IncrementErrors atomically increments the errors counter.
func (r *redisStateRepository) IncrementErrors(ctx context.Context, projectID string) error {
	key := buildKey(projectID)

	_, err := r.client.HIncrBy(ctx, key, fieldErrors, 1)
	if err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.IncrementErrors: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// Delete removes the state for a project.
func (r *redisStateRepository) Delete(ctx context.Context, projectID string) error {
	key := buildKey(projectID)

	if err := r.client.Del(ctx, key); err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.Delete: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// RefreshTTL refreshes the TTL to 7 days.
func (r *redisStateRepository) RefreshTTL(ctx context.Context, projectID string) error {
	key := buildKey(projectID)

	if err := r.client.Expire(ctx, key, stateTTLSeconds); err != nil {
		r.logger.Errorf(ctx, "state.repository.redis.RefreshTTL: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}
