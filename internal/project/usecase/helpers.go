package usecase

import (
	"context"
	"strings"

	"project-srv/internal/model"
	"project-srv/internal/project"
	repo "project-srv/internal/project/repository"
)

func (uc *implUseCase) getProjectForLifecycle(ctx context.Context, id string, action string) (model.Project, error) {
	projectID := strings.TrimSpace(id)
	if projectID == "" {
		uc.l.Warnf(ctx, "project.usecase.%s: empty id", action)
		return model.Project{}, project.ErrNotFound
	}

	current, err := uc.repo.Detail(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.%s.repo.Detail: id=%s err=%v", action, projectID, err)
		if err == repo.ErrNotFound {
			return model.Project{}, project.ErrNotFound
		}
		return model.Project{}, project.ErrDetailFailed
	}

	return current, nil
}

// validateStatus checks if the given status is valid for a project.
func validateStatus(status string) error {
	switch model.ProjectStatus(status) {
	case model.ProjectStatusDraft, model.ProjectStatusActive, model.ProjectStatusPaused, model.ProjectStatusArchived:
		return nil
	default:
		return project.ErrInvalidStatus
	}
}

// validateEntityType checks if the given entity type is valid.
func validateEntityType(entityType string) error {
	switch model.EntityType(entityType) {
	case model.EntityTypeProduct, model.EntityTypeCampaign, model.EntityTypeService,
		model.EntityTypeCompetitor, model.EntityTypeTopic:
		return nil
	default:
		return project.ErrInvalidEntity
	}
}

func canActivate(status model.ProjectStatus) bool {
	return status == model.ProjectStatusDraft || status == model.ProjectStatusPaused
}

func canPause(status model.ProjectStatus) bool {
	return status == model.ProjectStatusActive
}

func canResume(status model.ProjectStatus) bool {
	return status == model.ProjectStatusPaused
}

func canArchive(status model.ProjectStatus) bool {
	return status == model.ProjectStatusDraft ||
		status == model.ProjectStatusActive ||
		status == model.ProjectStatusPaused
}

func canUnarchive(status model.ProjectStatus) bool {
	return status == model.ProjectStatusArchived
}
