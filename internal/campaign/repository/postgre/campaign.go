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
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// Create inserts a new campaign into the database.
func (r *implRepository) Create(ctx context.Context, opt repository.CreateOptions) (model.Campaign, error) {
	row := &sqlboiler.Campaign{
		Name:      opt.Name,
		CreatedBy: opt.CreatedBy,
		Status:    sqlboiler.CampaignStatusACTIVE,
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
	row, err := sqlboiler.FindCampaign(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.l.Warnf(ctx, "campaign.repository.Detail.FindCampaign: not found id=%s", id)
			return model.Campaign{}, repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "campaign.repository.Detail.FindCampaign: %v", err)
		return model.Campaign{}, repository.ErrFailedToGet
	}

	result := model.NewCampaignFromDB(row)
	return *result, nil
}

// Get fetches campaigns with pagination and filters.
func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]model.Campaign, paginator.Paginator, error) {
	mods := r.buildGetQuery(opt)

	// Count total
	total, err := sqlboiler.Campaigns(mods...).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Get.Count: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	// Apply pagination from PaginateQuery
	pq := opt.Paginator
	mods = append(mods,
		qm.Limit(int(pq.Limit)),
		qm.Offset(int(pq.Offset())),
		qm.OrderBy(fmt.Sprintf("%s DESC", sqlboiler.CampaignColumns.CreatedAt)),
	)

	rows, err := sqlboiler.Campaigns(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "campaign.repository.Get.All: %v", err)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	campaigns := make([]model.Campaign, 0, len(rows))
	for _, row := range rows {
		campaigns = append(campaigns, *model.NewCampaignFromDB(row))
	}

	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(campaigns)),
		PerPage:     pq.Limit,
		CurrentPage: pq.Page,
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

	result := model.NewCampaignFromDB(row)
	return *result, nil
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
