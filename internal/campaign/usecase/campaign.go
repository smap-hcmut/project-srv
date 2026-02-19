package usecase

import (
	"context"
	"time"

	"project-srv/internal/campaign"
	repo "project-srv/internal/campaign/repository"
	"project-srv/pkg/scope"
)

// Create validates input and creates a new campaign.
func (uc *implUseCase) Create(ctx context.Context, input campaign.CreateInput) (campaign.CreateOutput, error) {
	// Business validation
	if input.Name == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Create: name is required")
		return campaign.CreateOutput{}, campaign.ErrNameRequired
	}

	// Parse dates
	var startDate, endDate *time.Time
	if input.StartDate != "" {
		t, err := time.Parse(time.RFC3339, input.StartDate)
		if err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.Create.parseStartDate: %v", err)
			return campaign.CreateOutput{}, campaign.ErrInvalidDateRange
		}
		startDate = &t
	}
	if input.EndDate != "" {
		t, err := time.Parse(time.RFC3339, input.EndDate)
		if err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.Create.parseEndDate: %v", err)
			return campaign.CreateOutput{}, campaign.ErrInvalidDateRange
		}
		endDate = &t
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		uc.l.Warnf(ctx, "campaign.usecase.Create: start_date after end_date")
		return campaign.CreateOutput{}, campaign.ErrInvalidDateRange
	}

	// Get user from context
	userID, _ := scope.GetUserIDFromContext(ctx)

	// Convert Input → Options
	opt := repo.CreateOptions{
		Name:        input.Name,
		Description: input.Description,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedBy:   userID,
	}

	result, err := uc.repo.Create(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Create.repo.Create: %v", err)
		return campaign.CreateOutput{}, campaign.ErrCreateFailed
	}

	return campaign.CreateOutput{Campaign: result}, nil
}

// Detail fetches a single campaign by ID.
func (uc *implUseCase) Detail(ctx context.Context, id string) (campaign.DetailOutput, error) {
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Detail: empty id")
		return campaign.DetailOutput{}, campaign.ErrNotFound
	}

	result, err := uc.repo.Detail(ctx, id)
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Detail.repo.Detail: id=%s err=%v", id, err)
		return campaign.DetailOutput{}, campaign.ErrNotFound
	}

	return campaign.DetailOutput{Campaign: result}, nil
}

// List fetches campaigns with pagination and filters.
func (uc *implUseCase) List(ctx context.Context, input campaign.ListInput) (campaign.ListOutput, error) {
	// Validate status if provided
	if input.Status != "" {
		if err := validateStatus(input.Status); err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.List.validateStatus: invalid status=%s", input.Status)
			return campaign.ListOutput{}, err
		}
	}

	// Convert Input → Options
	opt := repo.GetOptions{
		Status:    input.Status,
		Name:      input.Name,
		Paginator: input.Paginator,
	}

	campaigns, pag, err := uc.repo.Get(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.List.repo.Get: %v", err)
		return campaign.ListOutput{}, campaign.ErrListFailed
	}

	return campaign.ListOutput{
		Campaigns: campaigns,
		Paginator: pag,
	}, nil
}

// Update validates input and updates a campaign.
func (uc *implUseCase) Update(ctx context.Context, input campaign.UpdateInput) (campaign.UpdateOutput, error) {
	if input.ID == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Update: empty id")
		return campaign.UpdateOutput{}, campaign.ErrNotFound
	}

	// Validate status if provided
	if input.Status != "" {
		if err := validateStatus(input.Status); err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.Update.validateStatus: invalid status=%s", input.Status)
			return campaign.UpdateOutput{}, err
		}
	}

	// Parse dates
	var startDate, endDate *time.Time
	if input.StartDate != "" {
		t, err := time.Parse(time.RFC3339, input.StartDate)
		if err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.Update.parseStartDate: %v", err)
			return campaign.UpdateOutput{}, campaign.ErrInvalidDateRange
		}
		startDate = &t
	}
	if input.EndDate != "" {
		t, err := time.Parse(time.RFC3339, input.EndDate)
		if err != nil {
			uc.l.Warnf(ctx, "campaign.usecase.Update.parseEndDate: %v", err)
			return campaign.UpdateOutput{}, campaign.ErrInvalidDateRange
		}
		endDate = &t
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		uc.l.Warnf(ctx, "campaign.usecase.Update: start_date after end_date")
		return campaign.UpdateOutput{}, campaign.ErrInvalidDateRange
	}

	// Convert Input → Options
	opt := repo.UpdateOptions{
		ID:          input.ID,
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	result, err := uc.repo.Update(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Update.repo.Update: id=%s err=%v", input.ID, err)
		// Map repo not-found to UC not-found
		if err == repo.ErrFailedToGet {
			return campaign.UpdateOutput{}, campaign.ErrNotFound
		}
		return campaign.UpdateOutput{}, campaign.ErrUpdateFailed
	}

	return campaign.UpdateOutput{Campaign: result}, nil
}

// Archive soft-deletes a campaign by ID.
func (uc *implUseCase) Archive(ctx context.Context, id string) error {
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Archive: empty id")
		return campaign.ErrNotFound
	}

	if err := uc.repo.Archive(ctx, id); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Archive.repo.Archive: id=%s err=%v", id, err)
		// Map repo not-found to UC not-found
		if err == repo.ErrFailedToGet {
			return campaign.ErrNotFound
		}
		return campaign.ErrDeleteFailed
	}

	return nil
}
