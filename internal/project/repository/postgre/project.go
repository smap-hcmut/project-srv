package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"project-srv/internal/model"
	"project-srv/internal/project/repository"
	"project-srv/internal/sqlboiler"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// Create inserts a new project into the database.
func (r *implRepository) Create(ctx context.Context, opt repository.CreateOptions) (model.Project, error) {
	row := &sqlboiler.Project{
		CampaignID: opt.CampaignID,
		Name:       opt.Name,
		EntityType: sqlboiler.EntityType(opt.EntityType),
		EntityName: opt.EntityName,
		Status:     sqlboiler.ProjectStatusACTIVE,
		CreatedBy:  opt.CreatedBy,
	}

	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if opt.Brand != "" {
		row.Brand = null.StringFrom(opt.Brand)
	}

	row.ConfigStatus = sqlboiler.NullProjectConfigStatusFrom(sqlboiler.ProjectConfigStatusDRAFT)
	row.CreatedAt = null.TimeFrom(time.Now())
	row.UpdatedAt = null.TimeFrom(time.Now())

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "project.repository.Create.Insert: %v", err)
		return model.Project{}, repository.ErrFailedToInsert
	}

	result := model.NewProjectFromDB(row)
	return *result, nil
}

// Detail fetches a single project by ID.
func (r *implRepository) Detail(ctx context.Context, id string) (model.Project, error) {
	row, err := sqlboiler.FindProject(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "project.repository.Detail.FindProject: not found id=%s", id)
			return model.Project{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "project.repository.Detail.FindProject: %v", err)
		return model.Project{}, repository.ErrFailedToGet
	}

	result := model.NewProjectFromDB(row)
	return *result, nil
}

// Get fetches projects with pagination and filters.
func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]model.Project, paginator.Paginator, error) {
	mods := r.buildGetQuery(opt)

	// Count total
	total, err := sqlboiler.Projects(mods...).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Get.Count: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	// Apply pagination
	pq := opt.Paginator
	mods = append(mods,
		qm.Limit(int(pq.Limit)),
		qm.Offset(int(pq.Offset())),
		qm.OrderBy(fmt.Sprintf("%s DESC", sqlboiler.ProjectColumns.CreatedAt)),
	)

	rows, err := sqlboiler.Projects(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Get.All: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	projects := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, *model.NewProjectFromDB(row))
	}

	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(projects)),
		PerPage:     pq.Limit,
		CurrentPage: pq.Page,
	}

	return projects, pag, nil
}

// Update updates a project by ID.
func (r *implRepository) Update(ctx context.Context, opt repository.UpdateOptions) (model.Project, error) {
	row, err := sqlboiler.FindProject(ctx, r.db, opt.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "project.repository.Update.FindProject: not found id=%s", opt.ID)
			return model.Project{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "project.repository.Update.FindProject: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}

	if opt.Name != "" {
		row.Name = opt.Name
	}
	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if opt.Brand != "" {
		row.Brand = null.StringFrom(opt.Brand)
	}
	if opt.EntityType != "" {
		row.EntityType = sqlboiler.EntityType(opt.EntityType)
	}
	if opt.EntityName != "" {
		row.EntityName = opt.EntityName
	}
	if opt.Status != "" {
		row.Status = sqlboiler.ProjectStatus(opt.Status)
	}

	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Update.Update: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}

	result := model.NewProjectFromDB(row)
	return *result, nil
}

// Archive soft-deletes a project by setting deleted_at.
func (r *implRepository) Archive(ctx context.Context, id string) error {
	row, err := sqlboiler.FindProject(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "project.repository.Archive.FindProject: not found id=%s", id)
			return repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "project.repository.Archive.FindProject: %v", err)
		return repository.ErrFailedToDelete
	}

	row.DeletedAt = null.TimeFrom(time.Now())
	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.ProjectColumns.DeletedAt,
		sqlboiler.ProjectColumns.UpdatedAt,
	))
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Archive.Update: %v", err)
		return repository.ErrFailedToDelete
	}

	return nil
}
