package usecase

import (
	"context"

	"project-srv/internal/model"
	"project-srv/internal/project"
	repo "project-srv/internal/project/repository"
)

func (uc *implUseCase) Activate(ctx context.Context, id string) (project.ActivateOutput, error) {
	current, err := uc.getProjectForLifecycle(ctx, id, "Activate")
	if err != nil {
		return project.ActivateOutput{}, err
	}
	if !canActivate(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Activate: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ActivateOutput{}, project.ErrActivateNotAllowed
	}

	readiness, err := uc.GetActivationReadiness(ctx, current.ID)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.GetActivationReadiness: id=%s err=%v", current.ID, err)
		return project.ActivateOutput{}, project.ErrLifecycleManagerFailed
	}
	if !readiness.CanActivate {
		uc.l.Warnf(ctx, "project.usecase.Activate: id=%s readiness blocked", current.ID)
		return project.ActivateOutput{}, project.ErrReadinessFailed
	}

	if err := uc.ActivateProject(ctx, current.ID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.ActivateProject: id=%s err=%v", current.ID, err)
		return project.ActivateOutput{}, project.ErrLifecycleManagerFailed
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:     current.ID,
		Status: string(model.ProjectStatusActive),
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Activate.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			uc.l.Warnf(ctx, "project.usecase.Activate: id=%s not found during update", current.ID)
			return project.ActivateOutput{}, project.ErrNotFound
		}
		return project.ActivateOutput{}, project.ErrUpdateFailed
	}

	return project.ActivateOutput{Project: updated}, nil
}

func (uc *implUseCase) Pause(ctx context.Context, id string) (project.PauseOutput, error) {
	current, err := uc.getProjectForLifecycle(ctx, id, "Pause")
	if err != nil {
		return project.PauseOutput{}, err
	}
	if !canPause(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Pause: id=%s status=%s not eligible", current.ID, current.Status)
		return project.PauseOutput{}, project.ErrPauseNotAllowed
	}

	if err := uc.PauseProject(ctx, current.ID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Pause.PauseProject: id=%s err=%v", current.ID, err)
		return project.PauseOutput{}, project.ErrLifecycleManagerFailed
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:     current.ID,
		Status: string(model.ProjectStatusPaused),
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Pause.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.PauseOutput{}, project.ErrNotFound
		}
		return project.PauseOutput{}, project.ErrUpdateFailed
	}

	return project.PauseOutput{Project: updated}, nil
}

func (uc *implUseCase) Resume(ctx context.Context, id string) (project.ResumeOutput, error) {
	current, err := uc.getProjectForLifecycle(ctx, id, "Resume")
	if err != nil {
		return project.ResumeOutput{}, err
	}
	if !canResume(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Resume: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ResumeOutput{}, project.ErrResumeNotAllowed
	}

	readiness, err := uc.GetActivationReadiness(ctx, current.ID)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.GetActivationReadiness: id=%s err=%v", current.ID, err)
		return project.ResumeOutput{}, project.ErrLifecycleManagerFailed
	}
	if !readiness.CanActivate {
		uc.l.Warnf(ctx, "project.usecase.Resume: id=%s readiness blocked", current.ID)
		return project.ResumeOutput{}, project.ErrReadinessFailed
	}

	if err := uc.ResumeProject(ctx, current.ID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.ResumeProject: id=%s err=%v", current.ID, err)
		return project.ResumeOutput{}, project.ErrLifecycleManagerFailed
	}

	updated, err := uc.repo.UpdateStatus(ctx, repo.UpdateStatusOptions{
		ID:     current.ID,
		Status: string(model.ProjectStatusActive),
	})
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Resume.repo.UpdateStatus: id=%s err=%v", current.ID, err)
		if err == repo.ErrNotFound {
			return project.ResumeOutput{}, project.ErrNotFound
		}
		return project.ResumeOutput{}, project.ErrUpdateFailed
	}

	return project.ResumeOutput{Project: updated}, nil
}

func (uc *implUseCase) Archive(ctx context.Context, id string) (project.ArchiveOutput, error) {
	current, err := uc.getProjectForLifecycle(ctx, id, "Archive")
	if err != nil {
		return project.ArchiveOutput{}, err
	}
	if !canArchive(current.Status) {
		uc.l.Warnf(ctx, "project.usecase.Archive: id=%s status=%s not eligible", current.ID, current.Status)
		return project.ArchiveOutput{}, project.ErrInvalidTransition
	}

	if err := uc.PauseProject(ctx, current.ID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Archive.PauseProject: id=%s err=%v", current.ID, err)
		return project.ArchiveOutput{}, project.ErrLifecycleManagerFailed
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

	return project.ArchiveOutput{Project: updated}, nil
}

func (uc *implUseCase) Unarchive(ctx context.Context, id string) (project.UnarchiveOutput, error) {
	current, err := uc.getProjectForLifecycle(ctx, id, "Unarchive")
	if err != nil {
		return project.UnarchiveOutput{}, err
	}
	if !canUnarchive(current.Status) {
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

	return project.UnarchiveOutput{Project: updated}, nil
}

// GetActivationReadiness is a phase-1 noop implementation.
func (uc *implUseCase) GetActivationReadiness(_ context.Context, projectID string) (project.ActivationReadiness, error) {
	return project.ActivationReadiness{
		ProjectID:   projectID,
		CanActivate: true,
	}, nil
}

// ActivateProject is a phase-1 noop implementation.
func (uc *implUseCase) ActivateProject(_ context.Context, _ string) error {
	return nil
}

// PauseProject is a phase-1 noop implementation.
func (uc *implUseCase) PauseProject(_ context.Context, _ string) error {
	return nil
}

// ResumeProject is a phase-1 noop implementation.
func (uc *implUseCase) ResumeProject(_ context.Context, _ string) error {
	return nil
}
