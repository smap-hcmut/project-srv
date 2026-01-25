package postgres

import (
	"context"

	"smap-project/internal/project/repository"
	"smap-project/internal/sqlboiler"
	"smap-project/pkg/paginator"
	postgresPkg "smap-project/pkg/postgre"

	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

func (r *implRepository) buildDetailQuery(ctx context.Context, id string) ([]qm.QueryMod, error) {
	if err := postgresPkg.IsUUID(id); err != nil {
		return nil, err
	}

	return []qm.QueryMod{
		sqlboiler.ProjectWhere.ID.EQ(id),
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
	}, nil
}

func (r *implRepository) buildListQuery(ctx context.Context, opts repository.ListOptions) ([]qm.QueryMod, error) {
	mods := []qm.QueryMod{
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
		qm.OrderBy(sqlboiler.ProjectColumns.CreatedAt + " DESC"),
	}

	if len(opts.IDs) > 0 {
		mods = append(mods, sqlboiler.ProjectWhere.ID.IN(opts.IDs))
	}

	if len(opts.Statuses) > 0 {
		mods = append(mods, sqlboiler.ProjectWhere.Status.IN(opts.Statuses))
	}

	if opts.CreatedBy != nil && *opts.CreatedBy != "" {
		mods = append(mods, sqlboiler.ProjectWhere.CreatedBy.EQ(*opts.CreatedBy))
	}

	if opts.SearchName != nil && *opts.SearchName != "" {
		mods = append(mods, qm.Where("name ILIKE ?", "%"+*opts.SearchName+"%"))
	}

	return mods, nil
}

func (r *implRepository) buildGetQuery(ctx context.Context, opts repository.GetOptions, pq paginator.PaginateQuery) ([]qm.QueryMod, error) {
	mods := []qm.QueryMod{
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
		qm.OrderBy(sqlboiler.ProjectColumns.CreatedAt + " DESC"),
	}

	if len(opts.IDs) > 0 {
		mods = append(mods, sqlboiler.ProjectWhere.ID.IN(opts.IDs))
	}

	if len(opts.Statuses) > 0 {
		mods = append(mods, sqlboiler.ProjectWhere.Status.IN(opts.Statuses))
	}

	if opts.CreatedBy != nil && *opts.CreatedBy != "" {
		mods = append(mods, sqlboiler.ProjectWhere.CreatedBy.EQ(*opts.CreatedBy))
	}

	if opts.SearchName != nil && *opts.SearchName != "" {
		mods = append(mods, qm.Where("name ILIKE ?", "%"+*opts.SearchName+"%"))
	}

	// Pagination
	mods = append(mods, qm.Limit(int(pq.Limit)))
	mods = append(mods, qm.Offset(int(pq.Offset())))

	return mods, nil
}

func (r *implRepository) buildGetOneQuery(ctx context.Context, opts repository.GetOneOptions) ([]qm.QueryMod, error) {
	if err := postgresPkg.IsUUID(opts.ID); err != nil {
		return nil, err
	}

	return []qm.QueryMod{
		sqlboiler.ProjectWhere.ID.EQ(opts.ID),
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
	}, nil
}
