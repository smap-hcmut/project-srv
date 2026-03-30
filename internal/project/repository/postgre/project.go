package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"project-srv/internal/model"
	"project-srv/internal/project/repository"
	"project-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/lib/pq"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

type projectRow struct {
	ID              string
	CampaignID      string
	Name            string
	Description     sql.NullString
	Brand           sql.NullString
	EntityType      string
	EntityName      string
	Status          string
	ConfigStatus    sql.NullString
	FavoriteUserIDs pq.StringArray
	CreatedBy       string
	CreatedAt       sql.NullTime
	UpdatedAt       sql.NullTime
}

func toProjectModel(row projectRow) model.Project {
	result := model.Project{
		ID:              row.ID,
		CampaignID:      row.CampaignID,
		Name:            row.Name,
		EntityType:      model.EntityType(row.EntityType),
		EntityName:      row.EntityName,
		Status:          model.ProjectStatus(row.Status),
		FavoriteUserIDs: append([]string(nil), row.FavoriteUserIDs...),
		CreatedBy:       row.CreatedBy,
	}

	if row.Description.Valid {
		result.Description = row.Description.String
	}
	if row.Brand.Valid {
		result.Brand = row.Brand.String
	}
	if row.ConfigStatus.Valid {
		result.ConfigStatus = model.ProjectConfigStatus(row.ConfigStatus.String)
	}
	if row.CreatedAt.Valid {
		result.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		result.UpdatedAt = row.UpdatedAt.Time
	}

	return result
}

func (r *implRepository) fetchProjectByID(ctx context.Context, id string) (model.Project, error) {
	query := `
		SELECT id, campaign_id, name, description, brand, entity_type, entity_name, status, config_status, favorite_user_ids, created_by, created_at, updated_at
		FROM schema_project.projects
		WHERE id = $1 AND deleted_at IS NULL
	`

	var row projectRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.CampaignID,
		&row.Name,
		&row.Description,
		&row.Brand,
		&row.EntityType,
		&row.EntityName,
		&row.Status,
		&row.ConfigStatus,
		&row.FavoriteUserIDs,
		&row.CreatedBy,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "project.repository.fetchProjectByID.QueryRowContext: not found id=%s", id)
			return model.Project{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "project.repository.fetchProjectByID.QueryRowContext: %v", err)
		return model.Project{}, repository.ErrFailedToGet
	}

	return toProjectModel(row), nil
}

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
	return r.fetchProjectByID(ctx, id)
}

// Get fetches projects with pagination and filters.
func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]model.Project, paginator.Paginator, error) {
	whereClause, args := r.buildProjectFilters(opt)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM schema_project.projects WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		r.l.Errorf(ctx, "project.repository.Get.QueryRowContext.count: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	pqg := opt.Paginator
	orderBy := r.buildProjectOrderBy(opt, &args)
	args = append(args, pqg.Limit, pqg.Offset())

	query := fmt.Sprintf(`
		SELECT id, campaign_id, name, description, brand, entity_type, entity_name, status, config_status, favorite_user_ids, created_by, created_at, updated_at
		FROM schema_project.projects
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Get.QueryContext.list: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		var row projectRow
		if err := rows.Scan(
			&row.ID,
			&row.CampaignID,
			&row.Name,
			&row.Description,
			&row.Brand,
			&row.EntityType,
			&row.EntityName,
			&row.Status,
			&row.ConfigStatus,
			&row.FavoriteUserIDs,
			&row.CreatedBy,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			r.l.Errorf(ctx, "project.repository.Get.rows.Scan: %v", err)
			return nil, paginator.Paginator{}, repository.ErrFailedToList
		}
		projects = append(projects, toProjectModel(row))
	}
	if err := rows.Err(); err != nil {
		r.l.Errorf(ctx, "project.repository.Get.rows.Err: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(projects)),
		PerPage:     pqg.Limit,
		CurrentPage: pqg.Page,
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

	result, err := r.fetchProjectByID(ctx, opt.ID)
	if err != nil {
		return model.Project{}, err
	}
	return result, nil
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

		return r.fetchProjectByID(ctx, opt.ID)
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

	return r.fetchProjectByID(ctx, opt.ID)
}

// Favorite adds a user to the favorite array idempotently.
func (r *implRepository) Favorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE schema_project.projects
		SET favorite_user_ids = CASE
			WHEN NOT favorite_user_ids @> $2::uuid[] THEN array_append(favorite_user_ids, $3::uuid)
			ELSE favorite_user_ids
		END,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, pq.Array([]string{userID}), userID)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Favorite.ExecContext: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}

	affected, err := result.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Favorite.RowsAffected: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}
	if affected == 0 {
		r.l.Warnf(ctx, "project.repository.Favorite: not found id=%s", id)
		return repository.ErrNotFound
	}

	return nil
}

// Unfavorite removes a user from the favorite array idempotently.
func (r *implRepository) Unfavorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE schema_project.projects
		SET favorite_user_ids = array_remove(favorite_user_ids, $2::uuid),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Unfavorite.ExecContext: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}

	affected, err := result.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Unfavorite.RowsAffected: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}
	if affected == 0 {
		r.l.Warnf(ctx, "project.repository.Unfavorite: not found id=%s", id)
		return repository.ErrNotFound
	}

	return nil
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
