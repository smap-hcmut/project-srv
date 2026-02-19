package postgre

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"project-srv/internal/crisis/repository"
	"project-srv/internal/model"
	"project-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
)

// Upsert creates or updates a crisis config for a project.
func (r *implRepository) Upsert(ctx context.Context, opt repository.UpsertOptions) (model.CrisisConfig, error) {
	// Try to find existing
	existing, err := sqlboiler.FindProjectsCrisisConfig(ctx, r.db, opt.ProjectID)
	if err != nil && err != sql.ErrNoRows {
		r.l.Errorf(ctx, "crisis.repository.Upsert.Find: %v", err)
		return model.CrisisConfig{}, repository.ErrFailedToInsert
	}

	if existing == nil {
		// Create new
		return r.create(ctx, opt)
	}

	// Update existing
	return r.update(ctx, existing, opt)
}

func (r *implRepository) create(ctx context.Context, opt repository.UpsertOptions) (model.CrisisConfig, error) {
	row := &sqlboiler.ProjectsCrisisConfig{
		ProjectID: opt.ProjectID,
	}

	if opt.KeywordsTrigger != nil {
		b, _ := json.Marshal(opt.KeywordsTrigger)
		row.KeywordsRules = null.JSONFrom(b)
	}
	if opt.VolumeTrigger != nil {
		b, _ := json.Marshal(opt.VolumeTrigger)
		row.VolumeRules = null.JSONFrom(b)
	}
	if opt.SentimentTrigger != nil {
		b, _ := json.Marshal(opt.SentimentTrigger)
		row.SentimentRules = null.JSONFrom(b)
	}
	if opt.InfluencerTrigger != nil {
		b, _ := json.Marshal(opt.InfluencerTrigger)
		row.InfluencerRules = null.JSONFrom(b)
	}

	row.CreatedAt = null.TimeFrom(time.Now())
	row.UpdatedAt = null.TimeFrom(time.Now())

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "crisis.repository.create.Insert: %v", err)
		return model.CrisisConfig{}, repository.ErrFailedToInsert
	}

	result := model.NewCrisisConfigFromDB(row)
	return *result, nil
}

func (r *implRepository) update(ctx context.Context, row *sqlboiler.ProjectsCrisisConfig, opt repository.UpsertOptions) (model.CrisisConfig, error) {
	if opt.KeywordsTrigger != nil {
		b, _ := json.Marshal(opt.KeywordsTrigger)
		row.KeywordsRules = null.JSONFrom(b)
	}
	if opt.VolumeTrigger != nil {
		b, _ := json.Marshal(opt.VolumeTrigger)
		row.VolumeRules = null.JSONFrom(b)
	}
	if opt.SentimentTrigger != nil {
		b, _ := json.Marshal(opt.SentimentTrigger)
		row.SentimentRules = null.JSONFrom(b)
	}
	if opt.InfluencerTrigger != nil {
		b, _ := json.Marshal(opt.InfluencerTrigger)
		row.InfluencerRules = null.JSONFrom(b)
	}

	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err := row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "crisis.repository.update.Update: %v", err)
		return model.CrisisConfig{}, repository.ErrFailedToUpdate
	}

	result := model.NewCrisisConfigFromDB(row)
	return *result, nil
}

// Detail fetches a crisis config by project ID.
func (r *implRepository) Detail(ctx context.Context, projectID string) (model.CrisisConfig, error) {
	row, err := sqlboiler.FindProjectsCrisisConfig(ctx, r.db, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "crisis.repository.Detail: not found project_id=%s", projectID)
			return model.CrisisConfig{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "crisis.repository.Detail: %v", err)
		return model.CrisisConfig{}, repository.ErrFailedToGet
	}

	result := model.NewCrisisConfigFromDB(row)
	return *result, nil
}

// Delete removes a crisis config by project ID.
func (r *implRepository) Delete(ctx context.Context, projectID string) error {
	row, err := sqlboiler.FindProjectsCrisisConfig(ctx, r.db, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "crisis.repository.Delete: not found project_id=%s", projectID)
			return repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "crisis.repository.Delete.Find: %v", err)
		return repository.ErrFailedToDelete
	}

	_, err = row.Delete(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "crisis.repository.Delete.Delete: %v", err)
		return repository.ErrFailedToDelete
	}

	return nil
}
