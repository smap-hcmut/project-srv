package postgres

import (
	"context"
	"database/sql"

	"smap-project/internal/model"
	"smap-project/internal/project/repository"
	"smap-project/internal/sqlboiler"
	"smap-project/pkg/paginator"
	postgresPkg "smap-project/pkg/postgre"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

func (r *implRepository) Detail(ctx context.Context, sc model.Scope, id string) (model.Project, error) {
	mods, err := r.buildDetailQuery(ctx, id)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Detail.buildDetailQuery: %v", err)
		return model.Project{}, err
	}

	project, err := sqlboiler.Projects(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Project{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.project.repository.postgres.Detail.One: %v", err)
		return model.Project{}, err
	}

	return *model.NewProjectFromDB(project), nil
}

func (r *implRepository) List(ctx context.Context, sc model.Scope, opts repository.ListOptions) ([]model.Project, error) {
	mods, err := r.buildListQuery(ctx, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.List.buildListQuery: %v", err)
		return nil, err
	}

	projects, err := sqlboiler.Projects(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.List.All: %v", err)
		return nil, err
	}

	res := make([]model.Project, len(projects))
	for i, p := range projects {
		res[i] = *model.NewProjectFromDB(p)
	}

	return res, nil
}

func (r *implRepository) Create(ctx context.Context, sc model.Scope, opts repository.CreateOptions) (model.Project, error) {
	// Build model from options
	p, err := r.buildModelFromCreateOptions(ctx, sc, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Create.buildModelFromCreateOptions: %v", err)
		return model.Project{}, err
	}

	// Convert to DB model
	dbProject := p.ToDBProject()
	if err := dbProject.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Create.Insert: %v", err)
		return model.Project{}, err
	}

	// Reload from database to get generated ID and timestamps
	project, err := sqlboiler.Projects(
		sqlboiler.ProjectWhere.ID.EQ(dbProject.ID),
	).One(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Create.Reload: %v", err)
		return model.Project{}, err
	}

	return *model.NewProjectFromDB(project), nil
}

func (r *implRepository) Update(ctx context.Context, sc model.Scope, opts repository.UpdateOptions) (model.Project, error) {
	// Validate ID
	if err := postgresPkg.IsUUID(opts.ID); err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Update.IsUUID: %v", err)
		return model.Project{}, err
	}

	// Get existing project
	existing, err := sqlboiler.Projects(
		sqlboiler.ProjectWhere.ID.EQ(opts.ID),
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Project{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.project.repository.postgres.Update.Find: %v", err)
		return model.Project{}, err
	}

	// Convert to domain model
	existingProject := *model.NewProjectFromDB(existing)

	// Build updated model from options
	updated, err := r.buildModelFromUpdateOptions(ctx, existingProject, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Update.buildModelFromUpdateOptions: %v", err)
		return model.Project{}, err
	}

	// Convert to DB model
	dbProject := updated.ToDBProject()
	rows, err := dbProject.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Update.Update: %v", err)
		return model.Project{}, err
	}

	if rows == 0 {
		return model.Project{}, repository.ErrNotFound
	}

	// Reload from database
	project, err := sqlboiler.Projects(
		sqlboiler.ProjectWhere.ID.EQ(opts.ID),
	).One(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Update.Reload: %v", err)
		return model.Project{}, err
	}

	return *model.NewProjectFromDB(project), nil
}

func (r *implRepository) GetOne(ctx context.Context, sc model.Scope, opts repository.GetOneOptions) (model.Project, error) {
	mods, err := r.buildGetOneQuery(ctx, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.GetOne.buildGetOneQuery: %v", err)
		return model.Project{}, err
	}

	project, err := sqlboiler.Projects(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Project{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.project.repository.postgres.GetOne.One: %v", err)
		return model.Project{}, err
	}

	return *model.NewProjectFromDB(project), nil
}

func (r *implRepository) Get(ctx context.Context, sc model.Scope, opts repository.GetOptions) ([]model.Project, paginator.Paginator, error) {
	mods, err := r.buildGetQuery(ctx, opts, opts.PaginateQuery)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Get.buildGetQuery: %v", err)
		return nil, paginator.Paginator{}, err
	}

	cntMods := []qm.QueryMod{
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
	}

	// Apply filters
	if len(opts.IDs) > 0 {
		cntMods = append(cntMods, sqlboiler.ProjectWhere.ID.IN(opts.IDs))
	}

	if len(opts.Statuses) > 0 {
		cntMods = append(cntMods, sqlboiler.ProjectWhere.Status.IN(opts.Statuses))
	}

	if opts.CreatedBy != nil && *opts.CreatedBy != "" {
		cntMods = append(cntMods, sqlboiler.ProjectWhere.CreatedBy.EQ(*opts.CreatedBy))
	}

	if opts.SearchName != nil && *opts.SearchName != "" {
		cntMods = append(cntMods, qm.Where("name ILIKE ?", "%"+*opts.SearchName+"%"))
	}

	cnt, err := sqlboiler.Projects(cntMods...).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Get.Count: %v", err)
		return nil, paginator.Paginator{}, err
	}

	projects, err := sqlboiler.Projects(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Get.All: %v", err)
		return nil, paginator.Paginator{}, err
	}

	res := make([]model.Project, len(projects))
	for i, p := range projects {
		res[i] = *model.NewProjectFromDB(p)
	}

	pgn := paginator.NewPaginator(int(cnt), opts.PaginateQuery)

	return res, pgn, nil
}

func (r *implRepository) Delete(ctx context.Context, sc model.Scope, ids []string) error {
	if err := postgresPkg.ValidateUUIDs(ids); err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Delete.ValidateUUIDs: %v", err)
		return err
	}

	rowsAffected, err := sqlboiler.Projects(
		sqlboiler.ProjectWhere.ID.IN(ids),
		sqlboiler.ProjectWhere.DeletedAt.IsNull(),
	).UpdateAll(ctx, r.db, sqlboiler.M{
		sqlboiler.ProjectColumns.DeletedAt: "NOW()",
	})

	if rowsAffected != int64(len(ids)) {
		r.l.Warnf(ctx, "internal.project.repository.postgres.Delete.someProjectsNotFound: %v", ids)
		return repository.ErrNotFound
	}

	if err != nil {
		r.l.Errorf(ctx, "internal.project.repository.postgres.Delete.UpdateAll: %v", err)
		return err
	}

	return nil
}
