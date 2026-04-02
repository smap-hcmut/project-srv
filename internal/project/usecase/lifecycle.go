package usecase

import (
	"context"

	"project-srv/internal/model"
	"project-srv/internal/project"
	repo "project-srv/internal/project/repository"
	"project-srv/pkg/microservice"
)

func (uc *implUseCase) Activate(ctx context.Context, id string) (project.ActivateOutput, error) {
	detail, err := uc.Detail(ctx, id)
	if err != nil {
		return project.ActivateOutput{}, err
	}
	current := detail.Project
	if !model.CanActivateProjectStatus(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Activate: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ActivateOutput{}, project.ErrActivateNotAllowed
	}

	readiness, err := uc.GetActivationReadiness(ctx, project.ActivationReadinessInput{
		ProjectID: current.ID,
		Command:   project.ActivationReadinessCommandActivate,
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.GetActivationReadiness: id=%s err=%v", current.ID, err)
		return project.ActivateOutput{}, err
	}
	if !readiness.CanProceed {
		uc.l.Warnf(ctx, "project.usecase.Activate: id=%s readiness blocked", current.ID)
		return project.ActivateOutput{}, uc.mapReadinessBlockedError(readiness)
	}

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "project.usecase.Activate: ingest client is nil")
		return project.ActivateOutput{}, project.ErrLifecycleManagerFailed
	}
	if err := uc.ingest.Activate(ctx, current.ID); err != nil {
		mappedErr := project.MapLifecycleClientError(err)
		uc.l.Errorf(ctx, "project.usecase.Activate.ingest.Activate: id=%s err=%v mapped=%v", current.ID, err, mappedErr)
		return project.ActivateOutput{}, mappedErr
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:               current.ID,
		Status:           string(model.ProjectStatusActive),
		ExpectedStatuses: []string{string(model.ProjectStatusPending)},
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			uc.l.Warnf(ctx, "project.usecase.Activate: id=%s not found during update", current.ID)
			return project.ActivateOutput{}, project.ErrNotFound
		}
		if err == repo.ErrStatusConflict {
			uc.l.Warnf(ctx, "project.usecase.Activate: id=%s lost lifecycle race", current.ID)
			return project.ActivateOutput{}, project.ErrActivateNotAllowed
		}
		return project.ActivateOutput{}, project.ErrUpdateFailed
	}

	if err := uc.publishLifecycleEvent(ctx, project.ProjectLifecycleEventActivated, updated); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.publishLifecycleEvent: id=%s event=%s err=%v", updated.ID, project.ProjectLifecycleEventActivated, err)
	}

	return project.ActivateOutput{Project: updated}, nil
}

func (uc *implUseCase) Pause(ctx context.Context, id string) (project.PauseOutput, error) {
	detail, err := uc.Detail(ctx, id)
	if err != nil {
		return project.PauseOutput{}, err
	}
	current := detail.Project
	if !model.CanPauseProjectStatus(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Pause: id=%s status=%s not eligible", current.ID, current.Status)
		return project.PauseOutput{}, project.ErrPauseNotAllowed
	}

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "project.usecase.Pause: ingest client is nil")
		return project.PauseOutput{}, project.ErrLifecycleManagerFailed
	}
	if err := uc.ingest.Pause(ctx, current.ID); err != nil {
		mappedErr := project.MapLifecycleClientError(err)
		uc.l.Errorf(ctx, "project.usecase.Pause.ingest.Pause: id=%s err=%v mapped=%v", current.ID, err, mappedErr)
		return project.PauseOutput{}, mappedErr
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:               current.ID,
		Status:           string(model.ProjectStatusPaused),
		ExpectedStatuses: []string{string(model.ProjectStatusActive)},
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Pause.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.PauseOutput{}, project.ErrNotFound
		}
		if err == repo.ErrStatusConflict {
			uc.l.Warnf(ctx, "project.usecase.Pause: id=%s lost lifecycle race", current.ID)
			return project.PauseOutput{}, project.ErrPauseNotAllowed
		}
		return project.PauseOutput{}, project.ErrUpdateFailed
	}

	if err := uc.publishLifecycleEvent(ctx, project.ProjectLifecycleEventPaused, updated); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Pause.publishLifecycleEvent: id=%s event=%s err=%v", updated.ID, project.ProjectLifecycleEventPaused, err)
	}

	return project.PauseOutput{Project: updated}, nil
}

func (uc *implUseCase) Resume(ctx context.Context, id string) (project.ResumeOutput, error) {
	detail, err := uc.Detail(ctx, id)
	if err != nil {
		return project.ResumeOutput{}, err
	}
	current := detail.Project
	if !model.CanResumeProjectStatus(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Resume: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ResumeOutput{}, project.ErrResumeNotAllowed
	}

	readiness, err := uc.GetActivationReadiness(ctx, project.ActivationReadinessInput{
		ProjectID: current.ID,
		Command:   project.ActivationReadinessCommandResume,
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.GetActivationReadiness: id=%s err=%v", current.ID, err)
		return project.ResumeOutput{}, err
	}
	if !readiness.CanProceed {
		uc.l.Warnf(ctx, "project.usecase.Resume: id=%s readiness blocked", current.ID)
		return project.ResumeOutput{}, uc.mapReadinessBlockedError(readiness)
	}

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "project.usecase.Resume: ingest client is nil")
		return project.ResumeOutput{}, project.ErrLifecycleManagerFailed
	}
	if err := uc.ingest.Resume(ctx, current.ID); err != nil {
		mappedErr := project.MapLifecycleClientError(err)
		uc.l.Errorf(ctx, "project.usecase.Resume.ingest.Resume: id=%s err=%v mapped=%v", current.ID, err, mappedErr)
		return project.ResumeOutput{}, mappedErr
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:               current.ID,
		Status:           string(model.ProjectStatusActive),
		ExpectedStatuses: []string{string(model.ProjectStatusPaused)},
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.ResumeOutput{}, project.ErrNotFound
		}
		if err == repo.ErrStatusConflict {
			uc.l.Warnf(ctx, "project.usecase.Resume: id=%s lost lifecycle race", current.ID)
			return project.ResumeOutput{}, project.ErrResumeNotAllowed
		}
		return project.ResumeOutput{}, project.ErrUpdateFailed
	}

	if err := uc.publishLifecycleEvent(ctx, project.ProjectLifecycleEventResumed, updated); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.publishLifecycleEvent: id=%s event=%s err=%v", updated.ID, project.ProjectLifecycleEventResumed, err)
	}

	return project.ResumeOutput{Project: updated}, nil
}

func (uc *implUseCase) Archive(ctx context.Context, id string) (project.ArchiveOutput, error) {
	detail, err := uc.Detail(ctx, id)
	if err != nil {
		return project.ArchiveOutput{}, err
	}
	current := detail.Project
	if !model.CanArchiveProjectStatus(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Archive: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ArchiveOutput{}, project.ErrInvalidTransition
	}

	if current.Status == model.ProjectStatusActive {
		if uc.ingest == nil {
			uc.l.Errorf(ctx, "project.usecase.Archive: ingest client is nil")
			return project.ArchiveOutput{}, project.ErrLifecycleManagerFailed
		}
		if err := uc.ingest.Pause(ctx, current.ID); err != nil {
			mappedErr := project.MapLifecycleClientError(err)
			uc.l.Errorf(ctx, "project.usecase.Archive.ingest.Pause: id=%s err=%v mapped=%v", current.ID, err, mappedErr)
			return project.ArchiveOutput{}, mappedErr
		}
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:     current.ID,
		Status: string(model.ProjectStatusArchived),
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Archive.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.ArchiveOutput{}, project.ErrNotFound
		}
		return project.ArchiveOutput{}, project.ErrUpdateFailed
	}

	if err := uc.publishLifecycleEvent(ctx, project.ProjectLifecycleEventArchived, updated); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Archive.publishLifecycleEvent: id=%s event=%s err=%v", updated.ID, project.ProjectLifecycleEventArchived, err)
	}

	return project.ArchiveOutput{Project: updated}, nil
}

func (uc *implUseCase) Unarchive(ctx context.Context, id string) (project.UnarchiveOutput, error) {
	detail, err := uc.Detail(ctx, id)
	if err != nil {
		return project.UnarchiveOutput{}, err
	}
	current := detail.Project
	if !model.CanUnarchiveProjectStatus(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Unarchive: id=%s status=%s not eligible", current.ID, current.Status)
		return project.UnarchiveOutput{}, project.ErrUnarchiveNotAllowed
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:     current.ID,
		Status: string(model.ProjectStatusPaused),
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Unarchive.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.UnarchiveOutput{}, project.ErrNotFound
		}
		return project.UnarchiveOutput{}, project.ErrUpdateFailed
	}

	if err := uc.publishLifecycleEvent(ctx, project.ProjectLifecycleEventUnarchived, updated); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Unarchive.publishLifecycleEvent: id=%s event=%s err=%v", updated.ID, project.ProjectLifecycleEventUnarchived, err)
	}

	return project.UnarchiveOutput{Project: updated}, nil
}

// GetActivationReadiness evaluates project readiness by querying ingest internals and local status.
func (uc *implUseCase) GetActivationReadiness(ctx context.Context, input project.ActivationReadinessInput) (project.ActivationReadiness, error) {
	detail, err := uc.Detail(ctx, input.ProjectID)
	if err != nil {
		return project.ActivationReadiness{}, err
	}
	current := detail.Project

	if uc.ingest == nil {
		uc.l.Errorf(ctx, "project.usecase.GetActivationReadiness: ingest client is nil")
		return project.ActivationReadiness{}, project.ErrLifecycleManagerFailed
	}

	readiness, err := uc.ingest.GetActivationReadiness(ctx, microservice.ActivationReadinessInput{
		ProjectID: current.ID,
		Command:   microservice.ActivationReadinessCommand(uc.normalizeActivationReadinessCommand(input.Command)),
	})
	if err != nil {
		mappedErr := project.MapLifecycleClientError(err)
		uc.l.Errorf(ctx, "project.usecase.GetActivationReadiness.ingest.GetActivationReadiness: id=%s err=%v mapped=%v", current.ID, err, mappedErr)
		return project.ActivationReadiness{}, mappedErr
	}

	errorsOut := make([]project.ActivationReadinessError, 0, len(readiness.Errors))
	for _, e := range readiness.Errors {
		errorsOut = append(errorsOut, project.ActivationReadinessError{
			Code:         project.ActivationReadinessCode(e.Code),
			Message:      e.Message,
			DataSourceID: e.DataSourceID,
			TargetID:     e.TargetID,
		})
	}

	return project.ActivationReadiness{
		ProjectID:                current.ID,
		ProjectStatus:            current.Status,
		DataSourceCount:          readiness.DataSourceCount,
		HasDatasource:            readiness.HasDatasource,
		PassiveUnconfirmedCount:  readiness.PassiveUnconfirmedCount,
		MissingTargetDryrunCount: readiness.MissingTargetDryrunCount,
		FailedTargetDryrunCount:  readiness.FailedTargetDryrunCount,
		CanProceed:               readiness.CanProceed,
		Errors:                   errorsOut,
	}, nil
}
