package usecase

import (
	"context"
	"fmt"

	"smap-project/internal/keyword"
	"smap-project/internal/model"
	"smap-project/internal/project"
	"smap-project/internal/project/delivery/rabbitmq"
	"smap-project/internal/project/repository"

	"github.com/google/uuid"
)

func (uc *usecase) Detail(ctx context.Context, sc model.Scope, id string) (project.ProjectOutput, error) {
	p, err := uc.repo.Detail(ctx, sc, id)
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.Detail: %v", err)
			return project.ProjectOutput{}, project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.Detail: %v", err)
		return project.ProjectOutput{}, err
	}

	// Check if user owns this project
	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.Detail: user %s does not own project %s", sc.UserID, id)
		return project.ProjectOutput{}, project.ErrUnauthorized
	}

	return project.ProjectOutput{Project: p}, nil
}

func (uc *usecase) List(ctx context.Context, sc model.Scope, ip project.ListInput) ([]model.Project, error) {
	// Users can only see their own projects
	userID := sc.UserID

	opts := repository.ListOptions{
		IDs:        ip.Filter.IDs,
		Statuses:   ip.Filter.Statuses,
		CreatedBy:  &userID,
		SearchName: ip.Filter.SearchName,
	}

	projects, err := uc.repo.List(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.List: %v", err)
		return nil, err
	}

	return projects, nil
}

func (uc *usecase) Get(ctx context.Context, sc model.Scope, ip project.GetInput) (project.GetProjectOutput, error) {
	// Users can only see their own projects
	userID := sc.UserID

	opts := repository.GetOptions{
		IDs:           ip.Filter.IDs,
		Statuses:      ip.Filter.Statuses,
		CreatedBy:     &userID,
		SearchName:    ip.Filter.SearchName,
		PaginateQuery: ip.PaginateQuery,
	}

	projects, pag, err := uc.repo.Get(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Get: %v", err)
		return project.GetProjectOutput{}, err
	}

	return project.GetProjectOutput{
		Projects:  projects,
		Paginator: pag,
	}, nil
}

func (uc *usecase) Create(ctx context.Context, sc model.Scope, ip project.CreateInput) (project.ProjectOutput, error) {
	// Validate date range
	if ip.ToDate.Before(ip.FromDate) || ip.ToDate.Equal(ip.FromDate) {
		uc.l.Warnf(ctx, "internal.project.usecase.Create: invalid date range %s - %s", ip.FromDate, ip.ToDate)
		return project.ProjectOutput{}, project.ErrInvalidDateRange
	}

	// Validate and normalize brand keywords
	brandKeywords := ip.BrandKeywords
	if len(brandKeywords) > 0 {
		validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: brandKeywords})
		if err != nil {
			uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
			return project.ProjectOutput{}, err
		}
		brandKeywords = validateOut.ValidKeywords
	}

	// Validate and normalize competitor keywords
	competitorKeywords := make([]model.CompetitorKeyword, 0, len(ip.CompetitorKeywords))
	for _, ck := range ip.CompetitorKeywords {
		if len(ck.Keywords) > 0 {
			validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: ck.Keywords})
			if err != nil {
				uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
				return project.ProjectOutput{}, err
			}
			competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
				CompetitorName: ck.Name,
				Keywords:       validateOut.ValidKeywords,
			})
		}
	}

	// Extract competitor names from competitor keywords
	competitorNames := make([]string, 0, len(competitorKeywords))
	for _, ck := range competitorKeywords {
		competitorNames = append(competitorNames, ck.CompetitorName)
	}

	// Save project to PostgreSQL only (no Redis state, no event publishing)
	p, err := uc.repo.Create(ctx, sc, repository.CreateOptions{
		Name:               ip.Name,
		Description:        ip.Description,
		FromDate:           ip.FromDate,
		ToDate:             ip.ToDate,
		BrandName:          ip.BrandName,
		CompetitorNames:    competitorNames,
		BrandKeywords:      brandKeywords,
		CompetitorKeywords: competitorKeywords,
		CreatedBy:          sc.UserID,
	})
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
		return project.ProjectOutput{}, err
	}

	return project.ProjectOutput{
		Project: p,
	}, nil
}

// Execute starts processing for an existing project.
// Flow: Verify ownership → Check duplicate → Init Redis state → Publish event
func (uc *usecase) Execute(ctx context.Context, sc model.Scope, projectID string) error {
	// Step 1: Get project and verify ownership
	p, err := uc.repo.Detail(ctx, sc, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.Execute: project %s not found", projectID)
			return project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: %v", err)
		return err
	}

	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.Execute: user %s does not own project %s", sc.UserID, projectID)
		return project.ErrUnauthorized
	}

	// Step 2: Check if project is already executing (prevent duplicate execution)
	existingState, err := uc.stateUC.GetProjectState(ctx, projectID)
	if err == nil && existingState != nil {
		uc.l.Warnf(ctx, "internal.project.usecase.Execute: project %s already has state (status: %s)", projectID, existingState.Status)
		return project.ErrProjectAlreadyExecuting
	}

	// Step 3: Initialize Redis state
	if err := uc.stateUC.InitProjectState(ctx, projectID); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: %v", err)
		return err
	}

	// Step 4: Publish project.created event
	event := rabbitmq.ToProjectCreatedEvent(p)
	if err := uc.producer.PublishProjectCreated(ctx, event); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: %v", err)
		// Clean up Redis state on failure
		_ = uc.stateUC.DeleteProjectState(ctx, projectID)
		return err
	}

	return nil
}

func (uc *usecase) GetOne(ctx context.Context, sc model.Scope, ip project.GetOneInput) (model.Project, error) {
	p, err := uc.repo.GetOne(ctx, sc, repository.GetOneOptions{
		ID: ip.ID,
	})
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.GetOne: project %s not found", ip.ID)
			return model.Project{}, project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.GetOne: %v", err)
		return model.Project{}, err
	}

	// Check if user owns this project
	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.GetOne: user %s does not own project %s", sc.UserID, ip.ID)
		return model.Project{}, project.ErrUnauthorized
	}

	return p, nil
}

func (uc *usecase) Patch(ctx context.Context, sc model.Scope, ip project.PatchInput) (project.ProjectOutput, error) {
	p, err := uc.repo.Detail(ctx, sc, ip.ID)
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.Patch.Detail: project %s not found", ip.ID)
			return project.ProjectOutput{}, project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.Patch.Detail: %v", err)
		return project.ProjectOutput{}, err
	}

	opts := repository.UpdateOptions{
		ID:          ip.ID,
		Description: ip.Description,
		Status:      ip.Status,
		FromDate:    ip.FromDate,
		ToDate:      ip.ToDate,
	}

	// Check if user owns this project
	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.Patch: user %s does not own project %s", sc.UserID, ip.ID)
		return project.ProjectOutput{}, project.ErrUnauthorized
	}

	// Validate and normalize brand keywords
	brandKeywords := ip.BrandKeywords
	if len(brandKeywords) > 0 {
		validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: brandKeywords})
		if err != nil {
			uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
			return project.ProjectOutput{}, err
		}
		brandKeywords = validateOut.ValidKeywords
	}
	opts.BrandKeywords = brandKeywords

	// Validate and normalize competitor keywords
	competitorKeywords := make([]model.CompetitorKeyword, 0, len(ip.CompetitorKeywords))
	for _, ck := range ip.CompetitorKeywords {
		if len(ck.Keywords) > 0 {
			validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: ck.Keywords})
			if err != nil {
				uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
				return project.ProjectOutput{}, err
			}
			competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
				CompetitorName: ck.Name,
				Keywords:       validateOut.ValidKeywords,
			})
		}
	}
	opts.CompetitorKeywords = competitorKeywords

	up, err := uc.repo.Update(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Update: %v", err)
		return project.ProjectOutput{}, err
	}

	return project.ProjectOutput{Project: up}, nil
}

func (uc *usecase) Delete(ctx context.Context, sc model.Scope, ip project.DeleteInput) error {
	// Check if project exists and user owns it
	p, err := uc.repo.List(ctx, sc, repository.ListOptions{
		IDs: ip.IDs,
	})
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Delete.repo.List: %v", err)
		return err
	}

	if len(p) != len(ip.IDs) {
		uc.l.Warnf(ctx, "internal.project.usecase.Delete.someProjectsNotFound: %v", ip.IDs)
		return project.ErrProjectNotFound
	}

	for _, proj := range p {
		if proj.CreatedBy != sc.UserID {
			uc.l.Warnf(ctx, "internal.project.usecase.Delete: user %s does not own project %s", sc.UserID, proj.ID)
			return project.ErrUnauthorized
		}
	}

	if err := uc.repo.Delete(ctx, sc, ip.IDs); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Delete.repo.Delete: %v", err)
		return err
	}

	return nil
}

func (uc *usecase) GetProgress(ctx context.Context, sc model.Scope, projectID string) (project.ProgressOutput, error) {
	// Step 1: Verify user owns this project (authorization check)
	p, err := uc.repo.Detail(ctx, sc, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.GetProgress: project %s not found", projectID)
			return project.ProgressOutput{}, project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.GetProgress: %v", err)
		return project.ProgressOutput{}, err
	}

	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.GetProgress: user %s does not own project %s", sc.UserID, projectID)
		return project.ProgressOutput{}, project.ErrUnauthorized
	}

	// Step 2: Get state from Redis
	state, err := uc.stateUC.GetProjectState(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "internal.project.usecase.GetProgress: failed to get Redis state for project %s: %v", projectID, err)
		// Fall through to PostgreSQL fallback
	} else if state != nil {
		// Calculate progress percentage
		var progressPercent float64
		if state.Total > 0 {
			progressPercent = float64(state.Done) / float64(state.Total) * 100
		}

		return project.ProgressOutput{
			Project:         p,
			Status:          string(state.Status),
			TotalItems:      state.Total,
			ProcessedItems:  state.Done,
			FailedItems:     state.Errors,
			ProgressPercent: progressPercent,
		}, nil
	}

	// Step 3: Fallback to PostgreSQL status with zero progress
	return project.ProgressOutput{
		Project:         p,
		Status:          p.Status,
		TotalItems:      0,
		ProcessedItems:  0,
		FailedItems:     0,
		ProgressPercent: 0,
	}, nil
}

func (uc *usecase) DryRunKeywords(ctx context.Context, sc model.Scope, keywords []string) (string, error) {
	// Validate keywords using existing keyword validation
	// validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: keywords})
	// if err != nil {
	// 	uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords.Validate: %v", err)
	// 	return "", err
	// }

	// // Use validated keywords
	// validKeywords := validateOut.ValidKeywords
	// if len(validKeywords) == 0 {
	// 	uc.l.Warnf(ctx, "internal.project.usecase.DryRunKeywords: no valid keywords after validation")
	// 	return "", project.ErrInvalidKeywords
	// }

	// Generate UUID for job_id
	jobID := uuid.New().String()

	// Store job mapping in Redis before publishing to RabbitMQ
	// For dry-run jobs, projectID is empty since they're not associated with a specific project
	if err := uc.webhookUC.StoreJobMapping(ctx, jobID, sc.UserID, ""); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords.StoreJobMapping: jobID=%s, userID=%s, error=%v", jobID, sc.UserID, err)
		return "", fmt.Errorf("failed to store job mapping: %w", err)
	}
	uc.l.Infof(ctx, "Stored job mapping: jobID=%s, userID=%s", jobID, sc.UserID)

	// Build DryRunCrawlRequest message
	payload := map[string]any{
		"keywords":          keywords, // validKeywords
		"limit_per_keyword": 3,
		"include_comments":  true,
		"max_comments":      5,
	}

	msg := rabbitmq.DryRunCrawlRequest{
		JobID:       jobID,
		TaskType:    "dryrun_keyword",
		Payload:     payload,
		TimeRange:   0,
		Attempt:     1,
		MaxAttempts: 3,
		EmittedAt:   uc.clock(),
	}

	// Publish to RabbitMQ
	if err := uc.producer.PublishDryRunTask(ctx, msg); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords.PublishDryRunTask: %v", err)
		return "", err
	}

	uc.l.Infof(ctx, "Created dry-run job for user %s with %d keywords, job_id=%s", sc.UserID, len(keywords), jobID)
	return jobID, nil
}
