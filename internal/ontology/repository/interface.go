package repository

import (
	"context"

	"project-srv/internal/model"
)

type Repository interface {
	Upsert(ctx context.Context, opt UpsertOptions) (model.ProjectOntologyRules, error)
	Detail(ctx context.Context, projectID string) (model.ProjectOntologyRules, error)
	Delete(ctx context.Context, projectID string) error
}
