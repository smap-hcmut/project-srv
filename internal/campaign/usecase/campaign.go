package usecase

import (
	"context"
	"strings"
	"time"

	"project-srv/internal/campaign"
	repo "project-srv/internal/campaign/repository"
	"project-srv/internal/model"
	projectrepo "project-srv/internal/project/repository"

	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

const campaignDefaultProjectPageLimit = paginator.MaxLimit

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
	userID, _ := auth.GetUserIDFromContext(ctx)

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
	result = favoriteCampaignForUser(result, userID)

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
	userID, _ := auth.GetUserIDFromContext(ctx)
	result = favoriteCampaignForUser(result, userID)

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
	if err := validateSort(input.Sort); err != nil {
		uc.l.Warnf(ctx, "campaign.usecase.List.validateSort: invalid sort=%s", input.Sort)
		return campaign.ListOutput{}, err
	}
	userID, _ := auth.GetUserIDFromContext(ctx)

	// Convert Input → Options
	opt := repo.GetOptions{
		Status:        input.Status,
		Name:          input.Name,
		FavoriteOnly:  input.FavoriteOnly,
		Sort:          input.Sort,
		CurrentUserID: userID,
		Paginator:     input.Paginator,
	}

	campaigns, pag, err := uc.repo.Get(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.List.repo.Get: %v", err)
		return campaign.ListOutput{}, campaign.ErrListFailed
	}
	for i := range campaigns {
		campaigns[i] = favoriteCampaignForUser(campaigns[i], userID)
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
	userID, _ := auth.GetUserIDFromContext(ctx)
	result = favoriteCampaignForUser(result, userID)

	return campaign.UpdateOutput{Campaign: result}, nil
}

// Favorite marks a campaign as favorite for the current user.
func (uc *implUseCase) Favorite(ctx context.Context, id string) error {
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Favorite: empty id")
		return campaign.ErrNotFound
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Favorite: missing user id for id=%s", id)
		return campaign.ErrUpdateFailed
	}
	if err := uc.repo.Favorite(ctx, id, userID); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Favorite.repo.Favorite: id=%s user_id=%s err=%v", id, userID, err)
		if err == repo.ErrFailedToGet {
			return campaign.ErrNotFound
		}
		return campaign.ErrUpdateFailed
	}

	return nil
}

// Unfavorite removes a campaign favorite for the current user.
func (uc *implUseCase) Unfavorite(ctx context.Context, id string) error {
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Unfavorite: empty id")
		return campaign.ErrNotFound
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Unfavorite: missing user id for id=%s", id)
		return campaign.ErrUpdateFailed
	}
	if err := uc.repo.Unfavorite(ctx, id, userID); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Unfavorite.repo.Unfavorite: id=%s user_id=%s err=%v", id, userID, err)
		if err == repo.ErrFailedToGet {
			return campaign.ErrNotFound
		}
		return campaign.ErrUpdateFailed
	}

	return nil
}

// Archive soft-deletes a campaign by ID.
func (uc *implUseCase) Archive(ctx context.Context, id string) error {
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Archive: empty id")
		return campaign.ErrNotFound
	}

	if err := uc.shutdownCampaignRuntime(ctx, id); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Archive.shutdownCampaignRuntime: id=%s err=%v", id, err)
		return err
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

// shutdownCampaignRuntime pauses all active projects in the campaign to ensure
// no crawl jobs continue and no new scheduled jobs are produced.
// It hard-stops both ACTIVE and PENDING projects by:
// - cancelling runtime for ACTIVE/PENDING projects via ingest pause,
// - marking non-archived projects as PAUSED so periodic scheduling is suppressed.
func (uc *implUseCase) shutdownCampaignRuntime(ctx context.Context, campaignID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaign.ErrArchiveFailed
	}

	projects, err := uc.listCampaignProjects(ctx, campaignID, "")
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.shutdownCampaignRuntime.listCampaignProjects: id=%s err=%v", campaignID, err)
		return campaign.ErrArchiveFailed
	}

	for _, project := range projects {
		if err := uc.shutdownProjectRuntime(ctx, project); err != nil {
			uc.l.Errorf(
				ctx,
				"campaign.usecase.shutdownCampaignRuntime.shutdownProjectRuntime: campaign_id=%s project_id=%s status=%s err=%v",
				campaignID,
				project.ID,
				project.Status,
				err,
			)
			return campaign.ErrArchiveFailed
		}
	}

	return nil
}

func (uc *implUseCase) shutdownProjectRuntime(ctx context.Context, project model.Project) error {
	if project.Status == model.ProjectStatusArchived {
		return nil
	}

	if project.Status == model.ProjectStatusActive || project.Status == model.ProjectStatusPending {
		if uc.ingest == nil {
			return campaign.ErrPauseFailed
		}
		if err := uc.ingest.Pause(ctx, project.ID); err != nil {
			uc.l.Errorf(ctx, "campaign.usecase.shutdownProjectRuntime.ingest.Pause: project_id=%s status=%s err=%v", project.ID, project.Status, err)
			return campaign.ErrPauseFailed
		}
	}

	if project.Status == model.ProjectStatusPaused || project.Status == model.ProjectStatusArchived {
		return nil
	}

	if uc.projectRepo == nil {
		return campaign.ErrPauseFailed
	}

	_, err := uc.projectRepo.UpdateStatus(ctx, projectrepo.UpdateStatusOptions{
		ID:               project.ID,
		Status:           string(model.ProjectStatusPaused),
		ExpectedStatuses: []string{string(project.Status)},
	})
	if err != nil {
		if err == projectrepo.ErrNotFound || err == projectrepo.ErrStatusConflict {
			return nil
		}
		uc.l.Errorf(ctx, "campaign.usecase.shutdownProjectRuntime.projectRepo.UpdateStatus: project_id=%s err=%v", project.ID, err)
		return campaign.ErrPauseFailed
	}

	return nil
}

// Pause hard-stops all non-archived projects in the campaign to stop crawling tasks
// and prevent future schedule generation by setting them to PAUSED.
func (uc *implUseCase) Pause(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Pause: empty id")
		return campaign.ErrNotFound
	}

	projects, err := uc.listCampaignProjects(ctx, id, "")
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Pause.listCampaignProjects: id=%s err=%v", id, err)
		return err
	}

	for _, project := range projects {
		if err := uc.shutdownProjectRuntime(ctx, project); err != nil {
			uc.l.Errorf(ctx, "campaign.usecase.Pause.shutdownProjectRuntime: id=%s project_id=%s status=%s err=%v", id, project.ID, project.Status, err)
			return campaign.ErrPauseFailed
		}
	}

	if err := uc.setCampaignStatus(ctx, id, string(model.CampaignStatusPaused)); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Pause.setCampaignStatus: id=%s err=%v", id, err)
		return err
	}

	return nil
}

// Resume resumes all paused projects in the campaign.
func (uc *implUseCase) Resume(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		uc.l.Warnf(ctx, "campaign.usecase.Resume: empty id")
		return campaign.ErrNotFound
	}

	projects, err := uc.listCampaignProjects(ctx, id, string(model.ProjectStatusPaused))
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Resume.listCampaignProjects: id=%s err=%v", id, err)
		return err
	}

	for _, project := range projects {
		if err := uc.resumeProject(ctx, project.ID); err != nil {
			uc.l.Errorf(ctx, "campaign.usecase.Resume.resumeProject: id=%s project_id=%s err=%v", id, project.ID, err)
			return campaign.ErrResumeFailed
		}
	}

	if err := uc.setCampaignStatus(ctx, id, string(model.CampaignStatusActive)); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.Resume.setCampaignStatus: id=%s err=%v", id, err)
		return err
	}

	return nil
}

func (uc *implUseCase) listCampaignProjects(ctx context.Context, campaignID, status string) ([]model.Project, error) {
	if uc.projectRepo == nil {
		uc.l.Warnf(ctx, "campaign.usecase.listCampaignProjects: projectRepo is nil")
		return nil, campaign.ErrListFailed
	}

	all := make([]model.Project, 0)
	for page := 1; ; page++ {
		projectsOut, pagePaginator, err := uc.projectRepo.Get(ctx, projectrepo.GetOptions{
			CampaignID: campaignID,
			Status:     status,
			Paginator: paginator.PaginateQuery{
				Page:  page,
				Limit: campaignDefaultProjectPageLimit,
			},
		})
		if err != nil {
			uc.l.Errorf(ctx, "campaign.usecase.listCampaignProjects.projectRepo.Get: campaign_id=%s status=%s page=%d err=%v", campaignID, status, page, err)
			return nil, campaign.ErrListFailed
		}

		all = append(all, projectsOut...)
		if pagePaginator.Total <= int64(len(all)) {
			break
		}
	}

	return all, nil
}

func (uc *implUseCase) resumeProject(ctx context.Context, projectID string) error {
	if uc.ingest == nil || uc.projectRepo == nil {
		uc.l.Warnf(ctx, "campaign.usecase.resumeProject: dependencies missing for project_id=%s", projectID)
		return campaign.ErrResumeFailed
	}

	if err := uc.ingest.Resume(ctx, strings.TrimSpace(projectID)); err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.resumeProject.ingest.Resume: project_id=%s err=%v", projectID, err)
		return campaign.ErrResumeFailed
	}

	if _, err := uc.projectRepo.UpdateStatus(ctx, projectrepo.UpdateStatusOptions{
		ID:               strings.TrimSpace(projectID),
		Status:           string(model.ProjectStatusActive),
		ExpectedStatuses:  []string{string(model.ProjectStatusPaused)},
	}); err != nil {
		if err == projectrepo.ErrStatusConflict || err == projectrepo.ErrNotFound {
			return nil
		}
		uc.l.Errorf(ctx, "campaign.usecase.resumeProject.projectRepo.UpdateStatus: project_id=%s err=%v", projectID, err)
		return campaign.ErrResumeFailed
	}

	return nil
}

func (uc *implUseCase) setCampaignStatus(ctx context.Context, id, status string) error {
	_, err := uc.repo.Update(ctx, repo.UpdateOptions{ID: id, Status: status})
	if err != nil {
		uc.l.Errorf(ctx, "campaign.usecase.setCampaignStatus.repo.Update: id=%s status=%s err=%v", id, status, err)
		if err == repo.ErrFailedToGet {
			return campaign.ErrNotFound
		}
		return campaign.ErrUpdateFailed
	}

	return nil
}
