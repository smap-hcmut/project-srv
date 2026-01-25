package project

import (
	"context"

	"smap-project/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Detail(ctx context.Context, sc model.Scope, id string) (ProjectOutput, error)
	List(ctx context.Context, sc model.Scope, ip ListInput) ([]model.Project, error)
	Get(ctx context.Context, sc model.Scope, ip GetInput) (GetProjectOutput, error)
	Create(ctx context.Context, sc model.Scope, ip CreateInput) (ProjectOutput, error)
	GetOne(ctx context.Context, sc model.Scope, ip GetOneInput) (model.Project, error)
	Patch(ctx context.Context, sc model.Scope, ip PatchInput) (ProjectOutput, error)
	Delete(ctx context.Context, sc model.Scope, ip DeleteInput) error
	// SuggestKeywords(ctx context.Context, sc model.Scope, brandName string) ([]string, []string, error)
	DryRunKeywords(ctx context.Context, sc model.Scope, input DryRunKeywordsInput) (DryRunKeywordsOutput, error)
	GetProgress(ctx context.Context, sc model.Scope, projectID string) (ProgressOutput, error)
	GetPhaseProgress(ctx context.Context, sc model.Scope, projectID string) (ProjectProgressOutput, error)
	Execute(ctx context.Context, sc model.Scope, projectID string) error
}
