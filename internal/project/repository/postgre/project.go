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
	DomainTypeCode  string
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
		DomainTypeCode:  row.DomainTypeCode,
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
		SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at
		FROM project.projects
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
		&row.DomainTypeCode,
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

func (r *implRepository) Create(ctx context.Context, opt repository.CreateOptions) (model.Project, error) {
	now := time.Now().UTC()
	query := `
		INSERT INTO project.projects (
			campaign_id,
			name,
			description,
			brand,
			entity_type,
			entity_name,
			domain_type_code,
			status,
			config_status,
			created_by,
			created_at,
			updated_at
		) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id string
	if err := r.db.QueryRowContext(
		ctx,
		query,
		opt.CampaignID,
		opt.Name,
		opt.Description,
		opt.Brand,
		opt.EntityType,
		opt.EntityName,
		opt.DomainTypeCode,
		string(model.ProjectStatusPending),
		string(model.ConfigStatusDraft),
		opt.CreatedBy,
		now,
		now,
	).Scan(&id); err != nil {
		r.l.Errorf(ctx, "project.repository.Create.QueryRowContext: %v", err)
		return model.Project{}, repository.ErrFailedToInsert
	}

	return r.fetchProjectByID(ctx, id)
}

func (r *implRepository) Detail(ctx context.Context, id string) (model.Project, error) {
	return r.fetchProjectByID(ctx, id)
}

func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]model.Project, paginator.Paginator, error) {
	whereClause, args := r.buildProjectFilters(opt)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM project.projects WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		r.l.Errorf(ctx, "project.repository.Get.QueryRowContext.count: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	pqg := opt.Paginator
	orderBy := r.buildProjectOrderBy(opt, &args)
	args = append(args, pqg.Limit, pqg.Offset())

	query := fmt.Sprintf(`
		SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at
		FROM project.projects
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
			&row.DomainTypeCode,
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

func (r *implRepository) Update(ctx context.Context, opt repository.UpdateOptions) (model.Project, error) {
	assignments := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argPos := 2

	if opt.Name != "" {
		assignments = append(assignments, fmt.Sprintf("name = $%d", argPos))
		args = append(args, opt.Name)
		argPos++
	}
	if opt.Description != "" {
		assignments = append(assignments, fmt.Sprintf("description = $%d", argPos))
		args = append(args, opt.Description)
		argPos++
	}
	if opt.Brand != "" {
		assignments = append(assignments, fmt.Sprintf("brand = $%d", argPos))
		args = append(args, opt.Brand)
		argPos++
	}
	if opt.EntityType != "" {
		assignments = append(assignments, fmt.Sprintf("entity_type = $%d", argPos))
		args = append(args, opt.EntityType)
		argPos++
	}
	if opt.EntityName != "" {
		assignments = append(assignments, fmt.Sprintf("entity_name = $%d", argPos))
		args = append(args, opt.EntityName)
		argPos++
	}
	if opt.DomainTypeCode != "" {
		assignments = append(assignments, fmt.Sprintf("domain_type_code = $%d", argPos))
		args = append(args, opt.DomainTypeCode)
		argPos++
	}

	args = append(args, opt.ID)
	query := fmt.Sprintf(`
		UPDATE project.projects
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
	`, strings.Join(assignments, ", "), argPos)

	resultExec, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Update.ExecContext: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}
	rowsAffected, err := resultExec.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "project.repository.Update.RowsAffected: %v", err)
		return model.Project{}, repository.ErrFailedToUpdate
	}
	if rowsAffected == 0 {
		r.l.Warnf(ctx, "project.repository.Update: not found id=%s", opt.ID)
		return model.Project{}, repository.ErrNotFound
	}

	return r.fetchProjectByID(ctx, opt.ID)
}

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

func (r *implRepository) Favorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE project.projects
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

func (r *implRepository) Unfavorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE project.projects
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

	now := time.Now()
	row.DeletedAt = null.TimeFrom(now)
	row.UpdatedAt = null.TimeFrom(now)

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
