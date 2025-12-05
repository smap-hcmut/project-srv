package usecase

import (
	"context"

	"smap-project/internal/model"
	"smap-project/internal/state"
)

// InitProjectState initializes state for a new project with INITIALIZING status.
func (uc *stateUseCase) InitProjectState(ctx context.Context, projectID string) error {
	initialState := model.ProjectState{
		Status: model.ProjectStatusInitializing,
		Total:  0,
		Done:   0,
		Errors: 0,
	}

	if err := uc.repo.InitState(ctx, projectID, initialState); err != nil {
		uc.logger.Errorf(ctx, "state.usecase.InitProjectState: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// GetProjectState retrieves the current state of a project.
func (uc *stateUseCase) GetProjectState(ctx context.Context, projectID string) (*model.ProjectState, error) {
	return uc.repo.GetState(ctx, projectID)
}

// UpdateStatus updates the status of a project.
func (uc *stateUseCase) UpdateStatus(ctx context.Context, projectID string, status model.ProjectStatus) error {
	if err := uc.repo.SetStatus(ctx, projectID, status); err != nil {
		uc.logger.Errorf(ctx, "state.usecase.UpdateStatus: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// SetTotal sets the total number of items to process.
func (uc *stateUseCase) SetTotal(ctx context.Context, projectID string, total int64) error {
	if err := uc.repo.SetTotal(ctx, projectID, total); err != nil {
		uc.logger.Errorf(ctx, "state.usecase.SetTotal: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// IncrementDone increments the done counter and checks for completion.
// Business logic: if done >= total && total > 0, marks status as DONE.
func (uc *stateUseCase) IncrementDone(ctx context.Context, projectID string) (state.IncrementResult, error) {
	// Step 1: Atomically increment done counter (repository layer)
	newDone, err := uc.repo.IncrementDone(ctx, projectID)
	if err != nil {
		return state.IncrementResult{}, err
	}

	// Step 2: Get current state to check completion (business logic)
	currentState, err := uc.repo.GetState(ctx, projectID)
	if err != nil {
		uc.logger.Warnf(ctx, "state.usecase.IncrementDone: failed to get state for completion check: %v", err)
		return state.IncrementResult{NewDoneCount: newDone, IsComplete: false}, nil
	}

	if currentState == nil {
		return state.IncrementResult{NewDoneCount: newDone, IsComplete: false}, nil
	}

	// Step 3: Business logic - check completion
	isComplete := currentState.Total > 0 && newDone >= currentState.Total

	if isComplete {
		// Only mark as DONE if not already DONE (prevents duplicate completion events)
		if currentState.Status != model.ProjectStatusDone {
			if err := uc.repo.SetStatus(ctx, projectID, model.ProjectStatusDone); err != nil {
				uc.logger.Errorf(ctx, "state.usecase.IncrementDone: failed to set DONE status: %v", err)
			} else {
				uc.logger.Infof(ctx, "state.usecase.IncrementDone: project %s completed (%d/%d)", projectID, newDone, currentState.Total)
			}
		} else {
			// Already DONE, don't trigger completion again
			isComplete = false
		}
	}

	// Refresh TTL
	if err := uc.repo.RefreshTTL(ctx, projectID); err != nil {
		uc.logger.Warnf(ctx, "state.usecase.IncrementDone: failed to refresh TTL: %v", err)
	}

	return state.IncrementResult{
		NewDoneCount: newDone,
		Total:        currentState.Total,
		IsComplete:   isComplete,
	}, nil
}

// IncrementErrors increments the errors counter.
func (uc *stateUseCase) IncrementErrors(ctx context.Context, projectID string) error {
	if err := uc.repo.IncrementErrors(ctx, projectID); err != nil {
		uc.logger.Errorf(ctx, "state.usecase.IncrementErrors: failed for project %s: %v", projectID, err)
		return err
	}

	// Refresh TTL
	if err := uc.repo.RefreshTTL(ctx, projectID); err != nil {
		uc.logger.Warnf(ctx, "state.usecase.IncrementErrors: failed to refresh TTL: %v", err)
	}

	return nil
}

// DeleteProjectState removes the state for a project.
func (uc *stateUseCase) DeleteProjectState(ctx context.Context, projectID string) error {
	if err := uc.repo.Delete(ctx, projectID); err != nil {
		uc.logger.Errorf(ctx, "state.usecase.DeleteProjectState: failed for project %s: %v", projectID, err)
		return err
	}

	return nil
}

// GetProgressPercent calculates progress percentage.
func (uc *stateUseCase) GetProgressPercent(ctx context.Context, projectID string) (float64, error) {
	currentState, err := uc.repo.GetState(ctx, projectID)
	if err != nil {
		return 0, err
	}

	if currentState == nil || currentState.Total == 0 {
		return 0, nil
	}

	return float64(currentState.Done) / float64(currentState.Total) * 100, nil
}
