package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"project-srv/internal/model"
	"project-srv/internal/project/repository"
	"project-srv/internal/sqlboiler"
	"strings"
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
		Status:     sqlboiler.ProjectStatusPENDING,
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
			return model.Project{}, repository.ErrNotFound
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
			return model.Project{}, repository.ErrNotFound
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
	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Update.Update: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}

	result := model.NewProjectFromDB(row)
	return *result, nil
}

// UpdateStatus updates only the lifecycle status of a project.
func (r *implRepository) UpdateStatus(ctx context.Context, opt repository.UpdateStatusOptions) (model.Project, error) {
	if len(opt.ExpectedStatuses) > 0 {
		now := time.Now()
		mods := []qm.QueryMod{
			sqlboiler.ProjectWhere.ID.EQ(opt.ID),
		}

		expected := make([]interface{}, 0, len(opt.ExpectedStatuses))
		for _, status := range opt.ExpectedStatuses {
			trimmed := strings.TrimSpace(status)
			if trimmed == "" {
				continue
			}
			expected = append(expected, trimmed)
		}
		if len(expected) > 0 {
			mods = append(mods, qm.WhereIn(fmt.Sprintf("%s IN ?", sqlboiler.ProjectColumns.Status), expected...))
		}

		affected, err := sqlboiler.Projects(mods...).UpdateAll(ctx, r.db, sqlboiler.M{
			sqlboiler.ProjectColumns.Status:    opt.Status,
			sqlboiler.ProjectColumns.UpdatedAt: null.TimeFrom(now),
		})
		if err != nil {
			r.l.Errorf(ctx, "project.repository.UpdateStatus.UpdateAll: %v", err)
			return model.Project{}, repository.ErrFailedToUpdate
		}
		if affected == 0 {
			_, err := sqlboiler.FindProject(ctx, r.db, opt.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					r.l.Warnf(ctx, "project.repository.UpdateStatus.FindProjectAfterConflict: not found id=%s", opt.ID)
					return model.Project{}, repository.ErrNotFound
				}
				r.l.Errorf(ctx, "project.repository.UpdateStatus.FindProjectAfterConflict: %v", err)
				return model.Project{}, repository.ErrFailedToUpdate
			}
			r.l.Warnf(ctx, "project.repository.UpdateStatus: status conflict id=%s target_status=%s expected_statuses=%v", opt.ID, opt.Status, opt.ExpectedStatuses)
			return model.Project{}, repository.ErrStatusConflict
		}

		row, err := sqlboiler.FindProject(ctx, r.db, opt.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				r.l.Warnf(ctx, "project.repository.UpdateStatus.FindProjectAfterUpdate: not found id=%s", opt.ID)
				return model.Project{}, repository.ErrNotFound
			}
			r.l.Errorf(ctx, "project.repository.UpdateStatus.FindProjectAfterUpdate: %v", err)
			return model.Project{}, repository.ErrFailedToUpdate
		}

		result := model.NewProjectFromDB(row)
		return *result, nil
	}

	row, err := sqlboiler.FindProject(ctx, r.db, opt.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "project.repository.UpdateStatus.FindProject: not found id=%s", opt.ID)
			return model.Project{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "project.repository.UpdateStatus.FindProject: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}

	row.Status = sqlboiler.ProjectStatus(opt.Status)
	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.ProjectColumns.Status,
		sqlboiler.ProjectColumns.UpdatedAt,
	))
	if err != nil {
		r.l.Errorf(ctx, "project.repository.UpdateStatus.Update: %v", err)
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
			return repository.ErrNotFound
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
