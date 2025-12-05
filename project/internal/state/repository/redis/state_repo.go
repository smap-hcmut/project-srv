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
	fieldTotal      = "total"
	fieldDone       = "done"
	fieldErrors     = "errors"
	stateTTLSeconds = 7 * 24 * 60 * 60 // 604800 seconds
)

// buildKey constructs the Redis key for a project's state.
func buildKey(projectID string) string {
	return keyPrefix + projectID
}

func (r *redisStateRepository) InitState(ctx context.Context, projectID string, state model.ProjectState) error {
	key := buildKey(projectID)

	// Use pipeline for atomic operation
	pipe := r.client.Pipeline()

	// Set all fields using pipeline HSet
	pipe.HSet(ctx, key, fieldStatus, string(state.Status))
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

	if totalStr, ok := data[fieldTotal]; ok {
		if total, err := strconv.ParseInt(totalStr, 10, 64); err == nil {
			s.Total = total
		}
	}

	if doneStr, ok := data[fieldDone]; ok {
		if done, err := strconv.ParseInt(doneStr, 10, 64); err == nil {
			s.Done = done
		}
	}

	if errorsStr, ok := data[fieldErrors]; ok {
		if errors, err := strconv.ParseInt(errorsStr, 10, 64); err == nil {
			s.Errors = errors
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
