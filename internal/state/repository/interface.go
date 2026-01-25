package repository

import (
	"context"

	"smap-project/internal/model"
)

// StateRepository defines how project state is stored (no business logic here).
type StateRepository interface {
	// InitState creates a new state entry with initial values and sets a 7-day TTL.
	InitState(ctx context.Context, projectID string, state model.ProjectState) error

	// GetState retrieves the current state of a project, or nil if it doesn't exist.
	GetState(ctx context.Context, projectID string) (*model.ProjectState, error)

	// SetStatus updates only the status and refreshes TTL.
	SetStatus(ctx context.Context, projectID string, status model.ProjectStatus) error

	// SetTotal sets the total number of items and refreshes TTL.
	SetTotal(ctx context.Context, projectID string, total int64) error

	// IncrementDone atomically increments the done counter and returns new value.
	IncrementDone(ctx context.Context, projectID string) (int64, error)

	// IncrementErrors atomically increments the errors counter.
	IncrementErrors(ctx context.Context, projectID string) error

	// Delete removes the state for a project.
	Delete(ctx context.Context, projectID string) error

	// RefreshTTL sets the TTL to 7 days.
	RefreshTTL(ctx context.Context, projectID string) error
}
