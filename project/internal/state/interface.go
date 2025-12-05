package state

import (
	"context"

	"smap-project/internal/model"
)

type UseCase interface {
	InitProjectState(ctx context.Context, projectID string) error
	GetProjectState(ctx context.Context, projectID string) (*model.ProjectState, error)
	UpdateStatus(ctx context.Context, projectID string, status model.ProjectStatus) error
	SetTotal(ctx context.Context, projectID string, total int64) error
	IncrementDone(ctx context.Context, projectID string) (IncrementResult, error)
	IncrementErrors(ctx context.Context, projectID string) error
	DeleteProjectState(ctx context.Context, projectID string) error
	GetProgressPercent(ctx context.Context, projectID string) (float64, error)
}
