package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"project-srv/internal/campaign/repository"
	"project-srv/internal/model"
	"project-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/lib/pq"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

type campaignRow struct {
	ID              string
	Name            string
	Description     sql.NullString
	Status          string
	StartDate       sql.NullTime
	EndDate         sql.NullTime
	FavoriteUserIDs pq.StringArray
	CreatedBy       string
	CreatedAt       sql.NullTime
	UpdatedAt       sql.NullTime
}

func toCampaignModel(row campaignRow) model.Campaign {
	result := model.Campaign{
		ID:              row.ID,
		Name:            row.Name,
		Status:          model.CampaignStatus(row.Status),
		FavoriteUserIDs: append([]string(nil), row.FavoriteUserIDs...),
		CreatedBy:       row.CreatedBy,
	}

	if row.Description.Valid {
		result.Description = row.Description.String
	}
	if row.StartDate.Valid {
		t := row.StartDate.Time
		result.StartDate = &t
	}
	if row.EndDate.Valid {
		t := row.EndDate.Time
		result.EndDate = &t
	}
	if row.CreatedAt.Valid {
		result.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		result.UpdatedAt = row.UpdatedAt.Time
	}

	return result
}

func (r *implRepository) fetchCampaignByID(ctx context.Context, id string) (model.Campaign, error) {
	query := `
		SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at
		FROM schema_project.campaigns
		WHERE id = $1 AND deleted_at IS NULL
	`

	var row campaignRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Description,
		&row.Status,
		&row.StartDate,
		&row.EndDate,
		&row.FavoriteUserIDs,
		&row.CreatedBy,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "campaign.repository.fetchCampaignByID.QueryRowContext: not found id=%s", id)
			return model.Campaign{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "campaign.repository.fetchCampaignByID.QueryRowContext: %v", err)
		return model.Campaign{}, repository.ErrFailedToGet
	}

	return toCampaignModel(row), nil
}

// Create inserts a new campaign into the database.
func (r *implRepository) Create(ctx context.Context, opt repository.CreateOptions) (model.Campaign, error) {
	row := &sqlboiler.Campaign{
		Name:      opt.Name,
		CreatedBy: opt.CreatedBy,
		Status:    sqlboiler.CampaignStatusPENDING,
	}

	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if opt.StartDate != nil {
		row.StartDate = null.TimeFrom(*opt.StartDate)
	}
	if opt.EndDate != nil {
		row.EndDate = null.TimeFrom(*opt.EndDate)
	}

	row.CreatedAt = null.TimeFrom(time.Now())
	row.UpdatedAt = null.TimeFrom(time.Now())

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "campaign.repository.Create.Insert: %v", err)
		return model.Campaign{}, repository.ErrFailedToInsert
	}

	result := model.NewCampaignFromDB(row)
	return *result, nil
}

// Detail fetches a single campaign by ID.
func (r *implRepository) Detail(ctx context.Context, id string) (model.Campaign, error) {
	return r.fetchCampaignByID(ctx, id)
}

// Get fetches campaigns with pagination and filters.
func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]model.Campaign, paginator.Paginator, error) {
	whereClause, args := r.buildCampaignFilters(opt)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM schema_project.campaigns WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		r.l.Errorf(ctx, "campaign.repository.Get.QueryRowContext.count: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	pqg := opt.Paginator
	orderBy := r.buildCampaignOrderBy(opt, &args)
	args = append(args, pqg.Limit, pqg.Offset())

	query := fmt.Sprintf(`
		SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at
		FROM schema_project.campaigns
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Get.QueryContext.list: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}
	defer rows.Close()

	campaigns := make([]model.Campaign, 0)
	for rows.Next() {
		var row campaignRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Description,
			&row.Status,
			&row.StartDate,
			&row.EndDate,
			&row.FavoriteUserIDs,
			&row.CreatedBy,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			r.l.Errorf(ctx, "campaign.repository.Get.rows.Scan: %v", err)
			return nil, paginator.Paginator{}, repository.ErrFailedToList
		}
		campaigns = append(campaigns, toCampaignModel(row))
	}
	if err := rows.Err(); err != nil {
		r.l.Errorf(ctx, "campaign.repository.Get.rows.Err: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(campaigns)),
		PerPage:     pqg.Limit,
		CurrentPage: pqg.Page,
	}

	return campaigns, pag, nil
}

// Update updates a campaign by ID.
func (r *implRepository) Update(ctx context.Context, opt repository.UpdateOptions) (model.Campaign, error) {
	row, err := sqlboiler.FindCampaign(ctx, r.db, opt.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "campaign.repository.Update.FindCampaign: not found id=%s", opt.ID)
			return model.Campaign{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "campaign.repository.Update.FindCampaign: %v", err)
		return model.Campaign{}, repository.ErrFailedToUpdate
	}

	if opt.Name != "" {
		row.Name = opt.Name
	}
	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if opt.Status != "" {
		row.Status = sqlboiler.CampaignStatus(opt.Status)
	}
	if opt.StartDate != nil {
		row.StartDate = null.TimeFrom(*opt.StartDate)
	}
	if opt.EndDate != nil {
		row.EndDate = null.TimeFrom(*opt.EndDate)
	}

	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Update.Update: %v", err)
		return model.Campaign{}, repository.ErrFailedToUpdate
	}

	result, err := r.fetchCampaignByID(ctx, opt.ID)
	if err != nil {
		return model.Campaign{}, err
	}
	return result, nil
}

// Favorite adds a user to the favorite array idempotently.
func (r *implRepository) Favorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE schema_project.campaigns
		SET favorite_user_ids = CASE
			WHEN NOT favorite_user_ids @> $2::uuid[] THEN array_append(favorite_user_ids, $3::uuid)
			ELSE favorite_user_ids
		END,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, pq.Array([]string{userID}), userID)
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Favorite.ExecContext: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}

	affected, err := result.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Favorite.RowsAffected: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}
	if affected == 0 {
		r.l.Warnf(ctx, "campaign.repository.Favorite: not found id=%s", id)
		return repository.ErrFailedToGet
	}

	return nil
}

// Unfavorite removes a user from the favorite array idempotently.
func (r *implRepository) Unfavorite(ctx context.Context, id, userID string) error {
	query := `
		UPDATE schema_project.campaigns
		SET favorite_user_ids = array_remove(favorite_user_ids, $2::uuid),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Unfavorite.ExecContext: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}

	affected, err := result.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Unfavorite.RowsAffected: id=%s user_id=%s err=%v", id, userID, err)
		return repository.ErrFailedToUpdate
	}
	if affected == 0 {
		r.l.Warnf(ctx, "campaign.repository.Unfavorite: not found id=%s", id)
		return repository.ErrFailedToGet
	}

	return nil
}

// Archive soft-deletes a campaign by setting deleted_at.
func (r *implRepository) Archive(ctx context.Context, id string) error {
	row, err := sqlboiler.FindCampaign(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "campaign.repository.Archive.FindCampaign: not found id=%s", id)
			return repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "campaign.repository.Archive.FindCampaign: %v", err)
		return repository.ErrFailedToDelete
	}

	row.DeletedAt = null.TimeFrom(time.Now())
	row.UpdatedAt = null.TimeFrom(time.Now())

	_, err = row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.CampaignColumns.DeletedAt,
		sqlboiler.CampaignColumns.UpdatedAt,
	))
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Archive.Update: %v", err)
		return repository.ErrFailedToDelete
	}

	return nil
}
