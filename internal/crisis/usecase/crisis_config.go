package usecase

import (
	"context"

	"project-srv/internal/crisis"
	repo "project-srv/internal/crisis/repository"
)

// Upsert validates the project exists, then creates or updates the crisis config.
func (uc *implUseCase) Upsert(ctx context.Context, input crisis.UpsertInput) (crisis.UpsertOutput, error) {
	if input.ProjectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.Upsert: empty project_id")
		return crisis.UpsertOutput{}, crisis.ErrProjectInvalid
	}

	// Validate project exists
	_, err := uc.projectUC.Detail(ctx, input.ProjectID)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.Upsert.validateProject: project_id=%s err=%v", input.ProjectID, err)
		return crisis.UpsertOutput{}, crisis.ErrProjectInvalid
	}

	// Convert Input → Options
	opt := repo.UpsertOptions{
		ProjectID:         input.ProjectID,
		KeywordsTrigger:   input.KeywordsTrigger,
		VolumeTrigger:     input.VolumeTrigger,
		SentimentTrigger:  input.SentimentTrigger,
		InfluencerTrigger: input.InfluencerTrigger,
	}

	result, err := uc.repo.Upsert(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "crisis.usecase.Upsert.repo.Upsert: %v", err)
		return crisis.UpsertOutput{}, crisis.ErrUpsertFailed
	}

	return crisis.UpsertOutput{CrisisConfig: result}, nil
}

// Detail fetches a crisis config by project ID.
func (uc *implUseCase) Detail(ctx context.Context, projectID string) (crisis.DetailOutput, error) {
	if projectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.Detail: empty project_id")
		return crisis.DetailOutput{}, crisis.ErrNotFound
	}

	result, err := uc.repo.Detail(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "crisis.usecase.Detail.repo.Detail: project_id=%s err=%v", projectID, err)
		return crisis.DetailOutput{}, crisis.ErrNotFound
	}

	return crisis.DetailOutput{CrisisConfig: result}, nil
}

// Delete removes a crisis config by project ID.
func (uc *implUseCase) Delete(ctx context.Context, projectID string) error {
	if projectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.Delete: empty project_id")
		return crisis.ErrNotFound
	}

	if err := uc.repo.Delete(ctx, projectID); err != nil {
		uc.l.Errorf(ctx, "crisis.usecase.Delete.repo.Delete: project_id=%s err=%v", projectID, err)
		if err == repo.ErrFailedToGet {
			return crisis.ErrNotFound
		}
		return crisis.ErrDeleteFailed
	}

	return nil
}
