package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"project-srv/internal/crisis"
	repo "project-srv/internal/crisis/repository"
	"project-srv/internal/model"
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
		ResponsePolicy:    input.ResponsePolicy,
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

func (uc *implUseCase) RuntimeConfig(ctx context.Context, projectID string) (crisis.RuntimeConfigOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		uc.l.Warnf(ctx, "crisis.usecase.RuntimeConfig: empty project_id")
		return crisis.RuntimeConfigOutput{}, crisis.ErrNotFound
	}

	projectDetail, err := uc.projectUC.Detail(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.RuntimeConfig.projectUC.Detail: project_id=%s err=%v", projectID, err)
		return crisis.RuntimeConfigOutput{}, crisis.ErrProjectInvalid
	}

	detail, err := uc.Detail(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.RuntimeConfig.Detail: project_id=%s err=%v", projectID, err)
		return crisis.RuntimeConfigOutput{}, err
	}

	p := projectDetail.Project
	return crisis.RuntimeConfigOutput{
		ProjectID:    projectID,
		ProjectName:  p.Name,
		CampaignID:   p.CampaignID,
		OwnerUserID:  p.CreatedBy,
		CrisisConfig: detail.CrisisConfig,
	}, nil
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

	level := model.CrisisRuntimeLevel(detail.CrisisConfig.Status)
	if input.Status != nil {
		level = model.CrisisRuntimeLevel(*input.Status)
	}
	if input.CrisisLevel != nil {
		level = *input.CrisisLevel
	}
	normalizedLevel, err := uc.normalizeRuntimeLevel(level)
	if err != nil {
		uc.l.Warnf(ctx, "crisis.usecase.ApplyRuntime: invalid level=%s", level)
		return crisis.ApplyRuntimeOutput{}, crisis.ErrInvalidStatus
	}

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "crisis.usecase.ApplyRuntime: ingest client is nil")
		return crisis.ApplyRuntimeOutput{}, crisis.ErrApplyFailed
	}

	crawlMode := uc.mapLevelToCrawlMode(normalizedLevel, detail.CrisisConfig.ResponsePolicy)
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = fmt.Sprintf("crisis runtime apply level=%s", normalizedLevel)
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
		CrisisStatus:            uc.mapLevelToStatus(normalizedLevel),
		CrisisLevel:             normalizedLevel,
		AppliedCrawlMode:        crawlMode,
		AffectedDataSourceCount: ingestOut.AffectedDataSourceCount,
		NoopReason:              ingestOut.NoopReason,
	}, nil
}

func (uc *implUseCase) normalizeStatus(status model.CrisisStatus) (model.CrisisStatus, error) {
	s := model.CrisisStatus(strings.ToUpper(strings.TrimSpace(string(status))))
	switch s {
	case model.CrisisStatusNormal, model.CrisisStatusWatch, model.CrisisStatusWarning, model.CrisisStatusCritical:
		return s, nil
	default:
		return "", crisis.ErrInvalidStatus
	}
}

func (uc *implUseCase) normalizeRuntimeLevel(level model.CrisisRuntimeLevel) (model.CrisisRuntimeLevel, error) {
	l := model.CrisisRuntimeLevel(strings.ToUpper(strings.TrimSpace(string(level))))
	switch l {
	case model.CrisisRuntimeLevelNone, model.CrisisRuntimeLevelNormal, model.CrisisRuntimeLevelWatch, model.CrisisRuntimeLevelWarning, model.CrisisRuntimeLevelCritical:
		return l, nil
	default:
		return "", crisis.ErrInvalidStatus
	}
}

func (uc *implUseCase) mapLevelToStatus(level model.CrisisRuntimeLevel) model.CrisisStatus {
	switch level {
	case model.CrisisRuntimeLevelWatch:
		return model.CrisisStatusWatch
	case model.CrisisRuntimeLevelWarning:
		return model.CrisisStatusWarning
	case model.CrisisRuntimeLevelCritical:
		return model.CrisisStatusCritical
	default:
		return model.CrisisStatusNormal
	}
}

func (uc *implUseCase) mapLevelToCrawlMode(level model.CrisisRuntimeLevel, policy model.CrisisResponsePolicy) string {
	policy = policy.WithDefaults()
	if !policy.AdaptiveCrawl.Enabled {
		return "NORMAL"
	}
	if crisisLevelRank(level) >= crisisLevelRank(model.CrisisRuntimeLevel(policy.AdaptiveCrawl.TriggerLevel)) {
		return "CRISIS"
	}
	return "NORMAL"
}

func crisisLevelRank(level model.CrisisRuntimeLevel) int {
	switch model.CrisisRuntimeLevel(strings.ToUpper(strings.TrimSpace(string(level)))) {
	case model.CrisisRuntimeLevelCritical:
		return 4
	case model.CrisisRuntimeLevelWarning:
		return 3
	case model.CrisisRuntimeLevelWatch:
		return 2
	case model.CrisisRuntimeLevelNormal:
		return 1
	default:
		return 0
	}
}
