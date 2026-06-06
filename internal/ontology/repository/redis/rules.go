package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"project-srv/internal/model"
	"project-srv/internal/ontology/repository"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyPrefix      = "smap:project:ontology-rules:"
	defaultRuleTTL = 7 * 24 * time.Hour

	// ontologyEventStream is a Redis Stream that consumers (analysis-srv)
	// tail to invalidate their in-memory ontology cache the moment a rule
	// changes. Publishing here removes the up-to-60s staleness window the
	// previous HTTP cache had to live with.
	ontologyEventStream = "smap:ontology:events"

	// ontologyEventStreamMaxLen approximates the cap on stream length so
	// long-lived deployments do not accumulate every historical change.
	ontologyEventStreamMaxLen int64 = 5000
)

type storedConfig struct {
	ProjectID string                     `json:"project_id"`
	Enabled   bool                       `json:"enabled"`
	Rules     []model.OntologySignalRule `json:"rules"`
	UpdatedBy string                     `json:"updated_by,omitempty"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

// Upsert merges the incoming rule into the existing record under WATCH so
// concurrent edits do not last-write-wins each other. After the write
// succeeds a `ontology:updated` event is XAdded so analysis-srv can drop its
// 60s HTTP cache for the project immediately instead of waiting for the
// next refresh tick.
func (r *implRepository) Upsert(ctx context.Context, opt repository.UpsertOptions) (model.ProjectOntologyRules, error) {
	client := r.redis.GetClient()
	cacheKey := key(opt.ProjectID)

	var stored storedConfig
	txErr := client.Watch(ctx, func(tx *goredis.Tx) error {
		now := time.Now().UTC()
		createdAt := now
		raw, getErr := tx.Get(ctx, cacheKey).Result()
		if getErr == nil {
			var existing storedConfig
			if json.Unmarshal([]byte(raw), &existing) == nil && !existing.CreatedAt.IsZero() {
				createdAt = existing.CreatedAt
			}
		} else if !errors.Is(getErr, goredis.Nil) {
			return getErr
		}

		stored = storedConfig{
			ProjectID: opt.ProjectID,
			Enabled:   opt.Enabled,
			Rules:     opt.Rules,
			UpdatedBy: opt.UpdatedBy,
			CreatedAt: createdAt,
			UpdatedAt: now,
		}
		payload, mErr := json.Marshal(stored)
		if mErr != nil {
			return mErr
		}

		_, execErr := tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, cacheKey, payload, defaultRuleTTL)
			return nil
		})
		return execErr
	}, cacheKey)

	if txErr != nil {
		r.l.Errorf(ctx, "ontology.redis.Upsert: project_id=%s err=%v", opt.ProjectID, txErr)
		return model.ProjectOntologyRules{}, repository.ErrFailedToInsert
	}

	r.publishOntologyEvent(ctx, opt.ProjectID, "ontology:updated")
	return stored.toModel(), nil
}

func (r *implRepository) Detail(ctx context.Context, projectID string) (model.ProjectOntologyRules, error) {
	raw, err := r.redis.Get(ctx, key(projectID))
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return model.ProjectOntologyRules{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "ontology.redis.Detail.Get: project_id=%s err=%v", projectID, err)
		return model.ProjectOntologyRules{}, repository.ErrFailedToGet
	}
	var stored storedConfig
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		r.l.Errorf(ctx, "ontology.redis.Detail.Unmarshal: project_id=%s err=%v", projectID, err)
		return model.ProjectOntologyRules{}, repository.ErrFailedToGet
	}
	return stored.toModel(), nil
}

func (r *implRepository) Delete(ctx context.Context, projectID string) error {
	exists, err := r.redis.Exists(ctx, key(projectID))
	if err != nil {
		return repository.ErrFailedToDelete
	}
	if !exists {
		return repository.ErrFailedToGet
	}
	if err := r.redis.Delete(ctx, key(projectID)); err != nil {
		return repository.ErrFailedToDelete
	}
	r.publishOntologyEvent(ctx, projectID, "ontology:deleted")
	return nil
}

// publishOntologyEvent pushes a Redis Stream entry so cache holders can
// invalidate proactively. Best-effort — failures here do not roll back the
// write that already committed because the cache TTL is the safety net.
func (r *implRepository) publishOntologyEvent(ctx context.Context, projectID, eventType string) {
	args := &goredis.XAddArgs{
		Stream: ontologyEventStream,
		MaxLen: ontologyEventStreamMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"event":      eventType,
			"project_id": projectID,
			"emitted_at": time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	if err := r.redis.GetClient().XAdd(ctx, args).Err(); err != nil {
		r.l.Warnf(ctx, "ontology.redis.publishOntologyEvent: project_id=%s event=%s err=%v", projectID, eventType, err)
	}
}

func (stored storedConfig) toModel() model.ProjectOntologyRules {
	rules := stored.Rules
	if rules == nil {
		rules = []model.OntologySignalRule{}
	}
	return model.ProjectOntologyRules{
		ProjectID: stored.ProjectID,
		Enabled:   stored.Enabled,
		Rules:     rules,
		CreatedAt: stored.CreatedAt,
		UpdatedAt: stored.UpdatedAt,
	}
}

func key(projectID string) string {
	return keyPrefix + projectID
}
