package repository

import (
	"context"

	"smap-project/internal/model"
	"smap-project/pkg/paginator"
)

//go:generate mockery --name Repository
type Repository interface {
	Get(ctx context.Context, sc model.Scope, opts GetOptions) ([]model.Project, paginator.Paginator, error)
	Detail(ctx context.Context, sc model.Scope, id string) (model.Project, error)
	List(ctx context.Context, sc model.Scope, opts ListOptions) ([]model.Project, error)
	Create(ctx context.Context, sc model.Scope, opts CreateOptions) (model.Project, error)
	Update(ctx context.Context, sc model.Scope, opts UpdateOptions) (model.Project, error)
	GetOne(ctx context.Context, sc model.Scope, opts GetOneOptions) (model.Project, error)
	Delete(ctx context.Context, sc model.Scope, ids []string) error
}
