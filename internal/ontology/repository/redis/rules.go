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
	keyPrefix     = "smap:project:ontology-rules:"
	defaultRuleTTL = 7 * 24 * time.Hour
)

type storedConfig struct {
	ProjectID string                     `json:"project_id"`
	Enabled   bool                       `json:"enabled"`
	Rules     []model.OntologySignalRule `json:"rules"`
	UpdatedBy string                     `json:"updated_by,omitempty"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

func (r *implRepository) Upsert(ctx context.Context, opt repository.UpsertOptions) (model.ProjectOntologyRules, error) {
	now := time.Now().UTC()
	existing, err := r.Detail(ctx, opt.ProjectID)
	createdAt := now
	if err == nil && !existing.CreatedAt.IsZero() {
		createdAt = existing.CreatedAt
	}

	stored := storedConfig{
		ProjectID: opt.ProjectID,
		Enabled:   opt.Enabled,
		Rules:     opt.Rules,
		UpdatedBy: opt.UpdatedBy,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
	payload, err := json.Marshal(stored)
	if err != nil {
		r.l.Errorf(ctx, "ontology.redis.Upsert.Marshal: project_id=%s err=%v", opt.ProjectID, err)
		return model.ProjectOntologyRules{}, repository.ErrFailedToInsert
	}
	if err := r.redis.Set(ctx, key(opt.ProjectID), payload, defaultRuleTTL); err != nil {
		r.l.Errorf(ctx, "ontology.redis.Upsert.Set: project_id=%s err=%v", opt.ProjectID, err)
		return model.ProjectOntologyRules{}, repository.ErrFailedToInsert
	}
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
	return nil
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
