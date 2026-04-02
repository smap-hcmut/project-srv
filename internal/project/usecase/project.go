package usecase

import (
	"context"
	"strings"

	"project-srv/internal/model"
	"project-srv/internal/project"
	repo "project-srv/internal/project/repository"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

// Create validates input, checks campaign exists, and creates a new project.
func (uc *implUseCase) Create(ctx context.Context, input project.CreateInput) (project.CreateOutput, error) {
	if input.CampaignID == "" {
		uc.l.Warnf(ctx, "project.usecase.Create: campaign_id is required")
		return project.CreateOutput{}, project.ErrCampaignRequired
	}
	if input.Name == "" {
		uc.l.Warnf(ctx, "project.usecase.Create: name is required")
		return project.CreateOutput{}, project.ErrNameRequired
	}
	if input.EntityType != "" {
		if !model.IsValidEntityType(input.EntityType) {
			uc.l.Warnf(ctx, "project.usecase.Create.validateEntityType: invalid entity_type=%s", input.EntityType)
			return project.CreateOutput{}, project.ErrInvalidEntity
		}
	}
	if strings.TrimSpace(input.DomainTypeCode) == "" {
		uc.l.Warnf(ctx, "project.usecase.Create: domain_type_code is required")
		return project.CreateOutput{}, project.ErrDomainTypeRequired
	}
	exists, err := uc.repo.DomainTypeExists(ctx, input.DomainTypeCode)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Create.repo.DomainTypeExists: domain_type_code=%s err=%v", input.DomainTypeCode, err)
		return project.CreateOutput{}, project.ErrCreateFailed
	}
	if !exists {
		uc.l.Warnf(ctx, "project.usecase.Create: invalid domain_type_code=%s", input.DomainTypeCode)
		return project.CreateOutput{}, project.ErrInvalidDomainType
	}

	// Validate campaign exists
	_, err = uc.campaignUC.Detail(ctx, input.CampaignID)
	if err != nil {
		uc.l.Warnf(ctx, "project.usecase.Create.campaignUC.Detail: campaign_id=%s err=%v", input.CampaignID, err)
		return project.CreateOutput{}, project.ErrCampaignNotFound
	}

	// Get user from context
	userID, _ := auth.GetUserIDFromContext(ctx)

	// Convert Input → Options
	opt := repo.CreateOptions{
		CampaignID:     input.CampaignID,
		Name:           input.Name,
		Description:    input.Description,
		Brand:          input.Brand,
		EntityType:     input.EntityType,
		EntityName:     input.EntityName,
		DomainTypeCode: input.DomainTypeCode,
		CreatedBy:      userID,
	}

	result, err := uc.repo.Create(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Create.repo.Create: %v", err)
		return project.CreateOutput{}, project.ErrCreateFailed
	}
	result = uc.favoriteProjectForUser(result, userID)

	return project.CreateOutput{Project: result}, nil
}

// Detail fetches a single project by ID.
func (uc *implUseCase) Detail(ctx context.Context, id string) (project.DetailOutput, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "project.usecase.Detail: empty id")
		return project.DetailOutput{}, project.ErrNotFound
	}

	result, err := uc.repo.Detail(ctx, id)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Detail.repo.Detail: id=%s err=%v", id, err)
		if err == repo.ErrNotFound {
			uc.l.Warnf(ctx, "project.usecase.Detail: project not found id=%s", id)
			return project.DetailOutput{}, project.ErrNotFound
		}
		return project.DetailOutput{}, project.ErrDetailFailed
	}
	userID, _ := auth.GetUserIDFromContext(ctx)
	result = uc.favoriteProjectForUser(result, userID)

	return project.DetailOutput{Project: result}, nil
}

// List fetches projects with pagination and filters.
func (uc *implUseCase) List(ctx context.Context, input project.ListInput) (project.ListOutput, error) {
	if input.Status != "" {
		if !model.IsValidProjectStatus(input.Status) {
			uc.l.Warnf(ctx, "project.usecase.List.validateStatus: invalid status=%s", input.Status)
			return project.ListOutput{}, project.ErrInvalidStatus
		}
	}
	if input.EntityType != "" {
		if !model.IsValidEntityType(input.EntityType) {
			uc.l.Warnf(ctx, "project.usecase.List.validateEntityType: invalid entity_type=%s", input.EntityType)
			return project.ListOutput{}, project.ErrInvalidEntity
		}
	}
	if err := uc.validateProjectSort(input.Sort); err != nil {
		uc.l.Warnf(ctx, "project.usecase.List.validateProjectSort: invalid sort=%s", input.Sort)
		return project.ListOutput{}, err
	}
	userID, _ := auth.GetUserIDFromContext(ctx)

	// Convert Input → Options
	opt := repo.GetOptions{
		CampaignID:    input.CampaignID,
		Status:        input.Status,
		Name:          input.Name,
		Brand:         input.Brand,
		EntityType:    input.EntityType,
		FavoriteOnly:  input.FavoriteOnly,
		Sort:          input.Sort,
		CurrentUserID: userID,
		Paginator:     input.Paginator,
	}

	projects, pag, err := uc.repo.Get(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.List.repo.Get: %v", err)
		return project.ListOutput{}, project.ErrListFailed
	}
	for i := range projects {
		projects[i] = uc.favoriteProjectForUser(projects[i], userID)
	}

	return project.ListOutput{
		Projects:  projects,
		Paginator: pag,
	}, nil
}

// Update validates input and updates a project.
func (uc *implUseCase) Update(ctx context.Context, input project.UpdateInput) (project.UpdateOutput, error) {
	if input.ID == "" {
		uc.l.Warnf(ctx, "project.usecase.Update: empty id")
		return project.UpdateOutput{}, project.ErrNotFound
	}

	if input.EntityType != "" {
		if !model.IsValidEntityType(input.EntityType) {
			uc.l.Warnf(ctx, "project.usecase.Update.validateEntityType: invalid entity_type=%s", input.EntityType)
			return project.UpdateOutput{}, project.ErrInvalidEntity
		}
	}
	if strings.TrimSpace(input.DomainTypeCode) != "" {
		exists, err := uc.repo.DomainTypeExists(ctx, input.DomainTypeCode)
		if err != nil {
			uc.l.Errorf(ctx, "project.usecase.Update.repo.DomainTypeExists: domain_type_code=%s err=%v", input.DomainTypeCode, err)
			return project.UpdateOutput{}, project.ErrUpdateFailed
		}
		if !exists {
			uc.l.Warnf(ctx, "project.usecase.Update: invalid domain_type_code=%s", input.DomainTypeCode)
			return project.UpdateOutput{}, project.ErrInvalidDomainType
		}
	}

	// Convert Input → Options
	opt := repo.UpdateOptions{
		ID:             input.ID,
		Name:           input.Name,
		Description:    input.Description,
		Brand:          input.Brand,
		EntityType:     input.EntityType,
		EntityName:     input.EntityName,
		DomainTypeCode: input.DomainTypeCode,
	}

	result, err := uc.repo.Update(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Update.repo.Update: id=%s err=%v", input.ID, err)
		if err == repo.ErrNotFound {
			return project.UpdateOutput{}, project.ErrNotFound
		}
		return project.UpdateOutput{}, project.ErrUpdateFailed
	}
	userID, _ := auth.GetUserIDFromContext(ctx)
	result = uc.favoriteProjectForUser(result, userID)

	return project.UpdateOutput{Project: result}, nil
}

// Favorite marks a project as favorite for the current user.
func (uc *implUseCase) Favorite(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "project.usecase.Favorite: empty id")
		return project.ErrNotFound
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		uc.l.Warnf(ctx, "project.usecase.Favorite: missing user id for id=%s", id)
		return project.ErrUpdateFailed
	}
	if err := uc.repo.Favorite(ctx, id, userID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Favorite.repo.Favorite: id=%s user_id=%s err=%v", id, userID, err)
		if err == repo.ErrNotFound {
			return project.ErrNotFound
		}
		return project.ErrUpdateFailed
	}

	return nil
}

// Unfavorite removes a project favorite for the current user.
func (uc *implUseCase) Unfavorite(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "project.usecase.Unfavorite: empty id")
		return project.ErrNotFound
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		uc.l.Warnf(ctx, "project.usecase.Unfavorite: missing user id for id=%s", id)
		return project.ErrUpdateFailed
	}
	if err := uc.repo.Unfavorite(ctx, id, userID); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Unfavorite.repo.Unfavorite: id=%s user_id=%s err=%v", id, userID, err)
		if err == repo.ErrNotFound {
			return project.ErrNotFound
		}
		return project.ErrUpdateFailed
	}

	return nil
}

// Delete soft-deletes a project by ID after it has been archived.
func (uc *implUseCase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "project.usecase.Delete: empty id")
		return project.ErrNotFound
	}
	current, err := uc.repo.Detail(ctx, id)
	if err != nil {
		uc.l.Errorf(ctx, "project.usecase.Delete.repo.Detail: id=%s err=%v", id, err)
		if err == repo.ErrNotFound {
			return project.ErrNotFound
		}
		return project.ErrDeleteFailed
	}
	if current.Status != model.ProjectStatusArchived {
		uc.l.Warnf(ctx, "project.usecase.Delete: id=%s status=%s must be archived first", id, current.Status)
		return project.ErrDeleteRequiresArchived
	}

	if err := uc.repo.Archive(ctx, id); err != nil {
		uc.l.Errorf(ctx, "project.usecase.Delete.repo.Archive: id=%s err=%v", id, err)
		if err == repo.ErrNotFound {
			return project.ErrNotFound
		}
		return project.ErrDeleteFailed
	}

	return nil
}
