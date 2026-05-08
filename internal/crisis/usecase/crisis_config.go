package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"project-srv/internal/crisis"
	"project-srv/internal/model"
	repo "project-srv/internal/crisis/repository"
	"project-srv/pkg/microservice"
)

// Upsert validates the project exists, then creates or updates the crisis config.
func (uc *implUseCase) Upsert(ctx context.Context, input crisis.UpsertInput) (crisis.UpsertOutput, error) {
	if input.ProjectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.Upsert: empty project_id")
		return crisis.UpsertOutput{}, crisis.ErrProjectInvalid
	}

	if input.Status != nil {
		if _, err := uc.normalizeStatus(*input.Status); err != nil {
			uc.l.Warnf(ctx, "crisis.usecase.Upsert: invalid status=%s", *input.Status)
			return crisis.UpsertOutput{}, crisis.ErrInvalidStatus
		}
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
		Status:            input.Status,
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

func (uc *implUseCase) ApplyRuntime(ctx context.Context, input crisis.ApplyRuntimeInput) (crisis.ApplyRuntimeOutput, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime: empty project_id")
		return crisis.ApplyRuntimeOutput{}, crisis.ErrProjectInvalid
	}

	if _, err := uc.projectUC.Detail(ctx, projectID); err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime.validateProject: project_id=%s err=%v", projectID, err)
		return crisis.ApplyRuntimeOutput{}, crisis.ErrProjectInvalid
	}

	detail, err := uc.Detail(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime.Detail: project_id=%s err=%v", projectID, err)
		return crisis.ApplyRuntimeOutput{}, err
	}

	status := detail.CrisisConfig.Status
	if input.Status != nil {
		status = *input.Status
	}
	normalizedStatus, err := uc.normalizeStatus(status)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime: invalid status=%s", status)
		return crisis.ApplyRuntimeOutput{}, crisis.ErrInvalidStatus
	}

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "crisis.usecase.ApplyRuntime: ingest client is nil")
		return crisis.ApplyRuntimeOutput{}, crisis.ErrApplyFailed
	}

	crawlMode := uc.mapStatusToCrawlMode(normalizedStatus)
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = fmt.Sprintf("crisis runtime apply status=%s", normalizedStatus)
	}

	ingestOut, err := uc.ingest.UpdateProjectCrawlMode(ctx, microservice.UpdateProjectCrawlModeInput{
		ProjectID:   projectID,
		CrawlMode:   crawlMode,
		TriggerType: "CRISIS_EVENT",
		Reason:      reason,
		EventRef:    strings.TrimSpace(input.EventRef),
	})
	if err != nil {
		if errors.Is(err, microservice.ErrBadRequest) {
			uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime.ingest.UpdateProjectCrawlMode.badRequest: project_id=%s err=%v", projectID, err)
			return crisis.ApplyRuntimeOutput{}, crisis.ErrInvalidStatus
		}
		uc.l.Errorf(ctx, "crisis.usecase.ApplyRuntime.ingest.UpdateProjectCrawlMode: project_id=%s err=%v", projectID, err)
		return crisis.ApplyRuntimeOutput{}, crisis.ErrApplyFailed
	}

	return crisis.ApplyRuntimeOutput{
		ProjectID:               projectID,
		CrisisStatus:            normalizedStatus,
		AppliedCrawlMode:        crawlMode,
		AffectedDataSourceCount: ingestOut.AffectedDataSourceCount,
	}, nil
}

func (uc *implUseCase) normalizeStatus(status model.CrisisStatus) (model.CrisisStatus, error) {
	s := model.CrisisStatus(strings.ToUpper(strings.TrimSpace(string(status))))
	switch s {
	case model.CrisisStatusNormal, model.CrisisStatusWarning, model.CrisisStatusCritical:
		return s, nil
	default:
		return "", crisis.ErrInvalidStatus
	}
}

func (uc *implUseCase) mapStatusToCrawlMode(status model.CrisisStatus) string {
	switch status {
	case model.CrisisStatusWarning, model.CrisisStatusCritical:
		return "CRISIS"
	default:
		return "NORMAL"
	}
}
