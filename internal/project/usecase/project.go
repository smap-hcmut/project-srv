package usecase

import (
	"context"
	"fmt"

	"smap-project/internal/model"
	"smap-project/internal/project"
	"smap-project/internal/project/delivery/rabbitmq"
	"smap-project/internal/project/repository"
	"smap-project/internal/sampling"

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
	// TEMPORARILY DISABLED: LLM validation causes timeout (see validator.go)
	brandKeywords := ip.BrandKeywords
	// if len(brandKeywords) > 0 {
	// 	validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: brandKeywords})
	// 	if err != nil {
	// 		uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
	// 		return project.ProjectOutput{}, err
	// 	}
	// 	brandKeywords = validateOut.ValidKeywords
	// }

	// Validate and normalize competitor keywords
	// TEMPORARILY DISABLED: LLM validation causes timeout (see validator.go)
	competitorKeywords := make([]model.CompetitorKeyword, 0, len(ip.CompetitorKeywords))
	for _, ck := range ip.CompetitorKeywords {
		// if len(ck.Keywords) > 0 {
		// 	validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: ck.Keywords})
		// 	if err != nil {
		// 		uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
		// 		return project.ProjectOutput{}, err
		// 	}
		// 	competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
		// 		CompetitorName: ck.Name,
		// 		Keywords:       validateOut.ValidKeywords,
		// 	})
		// }
		competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
			CompetitorName: ck.Name,
			Keywords:       ck.Keywords,
		})
	}

	// Extract competitor names from competitor keywords
	competitorNames := make([]string, 0, len(competitorKeywords))
	for _, ck := range competitorKeywords {
		competitorNames = append(competitorNames, ck.CompetitorName)
	}

	// Save project to PostgreSQL only (no Redis state, no event publishing)
	// Set initial status to "draft" as per Requirements 2.1, 2.4
	p, err := uc.repo.Create(ctx, sc, repository.CreateOptions{
		Name:               ip.Name,
		Description:        ip.Description,
		Status:             model.ProjectStatusDraft,
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
// Flow: Verify ownership → Verify draft status → Check duplicate → Update PostgreSQL status → Init Redis state → Publish event
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 5.1
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

	// Step 2: Verify project is in "draft" status (Requirement 5.1)
	// Completed projects cannot be re-executed (Requirement 4.4)
	// Process projects are already executing (Requirement 3.4)
	if p.Status != model.ProjectStatusDraft {
		uc.l.Warnf(ctx, "internal.project.usecase.Execute: project %s has status %s, expected draft", projectID, p.Status)
		return project.ErrInvalidStatusTransition
	}

	// Step 3: Check if project is already executing (prevent duplicate execution - Requirement 3.4)
	existingState, err := uc.stateUC.GetProjectState(ctx, projectID)
	if err == nil && existingState != nil {
		uc.l.Warnf(ctx, "internal.project.usecase.Execute: project %s already has state (status: %s)", projectID, existingState.Status)
		return project.ErrProjectAlreadyExecuting
	}

	// Step 4: Update PostgreSQL status to "process" (Requirements 3.1, 3.5)
	processStatus := model.ProjectStatusProcess
	_, err = uc.repo.Update(ctx, sc, repository.UpdateOptions{
		ID:     projectID,
		Status: &processStatus,
	})
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: failed to update status to process: %v", err)
		return err
	}
	uc.l.Infof(ctx, "internal.project.usecase.Execute: updated project %s status to process", projectID)

	// Step 5: Initialize Redis state (Requirement 3.2)
	if err := uc.stateUC.InitProjectState(ctx, projectID); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: failed to init Redis state: %v", err)
		// Rollback PostgreSQL status to "draft"
		draftStatus := model.ProjectStatusDraft
		if rollbackErr := uc.rollbackStatus(ctx, sc, projectID, &draftStatus); rollbackErr != nil {
			uc.l.Errorf(ctx, "internal.project.usecase.Execute: failed to rollback status to draft: %v", rollbackErr)
		}
		return err
	}

	// Step 6: Publish project.created event (Requirement 3.3)
	event := rabbitmq.ToProjectCreatedEvent(p)
	if err := uc.producer.PublishProjectCreated(ctx, event); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.Execute: failed to publish event: %v", err)
		// Rollback both Redis and PostgreSQL
		_ = uc.stateUC.DeleteProjectState(ctx, projectID)
		draftStatus := model.ProjectStatusDraft
		if rollbackErr := uc.rollbackStatus(ctx, sc, projectID, &draftStatus); rollbackErr != nil {
			uc.l.Errorf(ctx, "internal.project.usecase.Execute: failed to rollback status to draft: %v", rollbackErr)
		}
		return err
	}

	uc.l.Infof(ctx, "internal.project.usecase.Execute: successfully executed project %s", projectID)
	return nil
}

// rollbackStatus is a helper function to rollback project status
func (uc *usecase) rollbackStatus(ctx context.Context, sc model.Scope, projectID string, status *string) error {
	_, err := uc.repo.Update(ctx, sc, repository.UpdateOptions{
		ID:     projectID,
		Status: status,
	})
	if err != nil {
		return err
	}
	uc.l.Infof(ctx, "internal.project.usecase.rollbackStatus: rolled back project %s status to %s", projectID, *status)
	return nil
}

// validateStatusTransition validates status transitions to prevent invalid state changes
// Valid transitions:
// - draft -> process (via Execute, not Patch)
// - process -> completed (via system, not user)
// - Any status can stay the same
// Users should not manually change status via Patch - status changes happen through Execute or system events
// Requirements: 5.2, 5.5
func (uc *usecase) validateStatusTransition(from, to string) error {
	// If status is not changing, allow it
	if from == to {
		return nil
	}

	// Users should not manually change status via Patch
	// Status changes happen through Execute (draft -> process) or system events (process -> completed)
	// Attempting to change status manually is an invalid transition
	return project.ErrInvalidStatusTransition
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

	// Check if user owns this project
	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.Patch: user %s does not own project %s", sc.UserID, ip.ID)
		return project.ProjectOutput{}, project.ErrUnauthorized
	}

	// Validate status transition if status is being updated (Requirements 5.2, 5.3, 5.5)
	if ip.Status != nil {
		// Validate status value is one of three allowed values (Requirement 5.3)
		if !model.IsValidProjectStatus(*ip.Status) {
			uc.l.Warnf(ctx, "internal.project.usecase.Patch: invalid status value %s", *ip.Status)
			return project.ProjectOutput{}, project.ErrInvalidStatus
		}

		// Enforce valid status transitions (Requirement 5.2, 5.5)
		if err := uc.validateStatusTransition(p.Status, *ip.Status); err != nil {
			uc.l.Warnf(ctx, "internal.project.usecase.Patch: invalid status transition from %s to %s", p.Status, *ip.Status)
			return project.ProjectOutput{}, err
		}
	}

	opts := repository.UpdateOptions{
		ID:          ip.ID,
		Description: ip.Description,
		Status:      ip.Status,
		FromDate:    ip.FromDate,
		ToDate:      ip.ToDate,
	}

	// Validate and normalize brand keywords
	// TEMPORARILY DISABLED: LLM validation causes timeout (see validator.go)
	brandKeywords := ip.BrandKeywords
	// if len(brandKeywords) > 0 {
	// 	validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: brandKeywords})
	// 	if err != nil {
	// 		uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
	// 		return project.ProjectOutput{}, err
	// 	}
	// 	brandKeywords = validateOut.ValidKeywords
	// }
	opts.BrandKeywords = brandKeywords

	// Validate and normalize competitor keywords
	// TEMPORARILY DISABLED: LLM validation causes timeout (see validator.go)
	competitorKeywords := make([]model.CompetitorKeyword, 0, len(ip.CompetitorKeywords))
	for _, ck := range ip.CompetitorKeywords {
		// if len(ck.Keywords) > 0 {
		// 	validateOut, err := uc.keywordUC.Validate(ctx, keyword.ValidateInput{Keywords: ck.Keywords})
		// 	if err != nil {
		// 		uc.l.Errorf(ctx, "internal.project.usecase.Create: %v", err)
		// 		return project.ProjectOutput{}, err
		// 	}
		// 	competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
		// 		CompetitorName: ck.Name,
		// 		Keywords:       validateOut.ValidKeywords,
		// 	})
		// }
		competitorKeywords = append(competitorKeywords, model.CompetitorKeyword{
			CompetitorName: ck.Name,
			Keywords:       ck.Keywords,
		})
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

// GetProgress returns project progress with status and execution metrics.
// Always returns PostgreSQL status (draft, process, completed) for consistency.
// Includes detailed execution metrics from Redis when available.
// For completed projects, Redis state is retained for historical reference (Requirement 4.2).
// Requirements: 4.2, 6.5
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

	// Step 2: Get execution metrics from Redis (if available)
	state, err := uc.stateUC.GetProjectState(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "internal.project.usecase.GetProgress: failed to get Redis state for project %s: %v", projectID, err)
		// Fall through to PostgreSQL fallback
	}

	// Step 3: Return PostgreSQL status with Redis metrics (if available)
	// Always return one of three valid PostgreSQL status values: "draft", "process", "completed"
	if state != nil {
		// Calculate progress percentage
		var progressPercent float64
		if state.Total > 0 {
			progressPercent = float64(state.Done) / float64(state.Total) * 100
		}

		return project.ProgressOutput{
			Project:         p,
			Status:          p.Status, // PostgreSQL status (draft, process, completed)
			TotalItems:      state.Total,
			ProcessedItems:  state.Done,
			FailedItems:     state.Errors,
			ProgressPercent: progressPercent,
		}, nil
	}

	// Step 4: Fallback to PostgreSQL status with zero metrics when Redis unavailable
	return project.ProgressOutput{
		Project:         p,
		Status:          p.Status, // PostgreSQL status (draft, process, completed)
		TotalItems:      0,
		ProcessedItems:  0,
		FailedItems:     0,
		ProgressPercent: 0,
	}, nil
}

// GetPhaseProgress returns phase-based project progress with separate crawl and analyze metrics.
// This is the new format that provides granular visibility into each processing phase.
// Requirements: 4.2, 6.5
func (uc *usecase) GetPhaseProgress(ctx context.Context, sc model.Scope, projectID string) (project.ProjectProgressOutput, error) {
	// Step 1: Verify user owns this project (authorization check)
	p, err := uc.repo.Detail(ctx, sc, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			uc.l.Warnf(ctx, "internal.project.usecase.GetPhaseProgress: project %s not found", projectID)
			return project.ProjectProgressOutput{}, project.ErrProjectNotFound
		}
		uc.l.Errorf(ctx, "internal.project.usecase.GetPhaseProgress: %v", err)
		return project.ProjectProgressOutput{}, err
	}

	if p.CreatedBy != sc.UserID {
		uc.l.Warnf(ctx, "internal.project.usecase.GetPhaseProgress: user %s does not own project %s", sc.UserID, projectID)
		return project.ProjectProgressOutput{}, project.ErrUnauthorized
	}

	// Step 2: Get execution metrics from Redis (if available)
	state, err := uc.stateUC.GetProjectState(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "internal.project.usecase.GetPhaseProgress: failed to get Redis state for project %s: %v", projectID, err)
		// Fall through to return zero metrics
	}

	// Step 3: Build phase-based progress output
	output := project.ProjectProgressOutput{
		ProjectID: projectID,
		Status:    p.Status,
	}

	if state != nil {
		output.Crawl = project.PhaseProgressOutput{
			Total:           state.CrawlTotal,
			Done:            state.CrawlDone,
			Errors:          state.CrawlErrors,
			ProgressPercent: state.CrawlProgressPercent(),
		}
		output.Analyze = project.PhaseProgressOutput{
			Total:           state.AnalyzeTotal,
			Done:            state.AnalyzeDone,
			Errors:          state.AnalyzeErrors,
			ProgressPercent: state.AnalyzeProgressPercent(),
		}
		output.OverallProgressPercent = state.OverallProgressPercent()
	}

	return output, nil
}

func (uc *usecase) DryRunKeywords(ctx context.Context, sc model.Scope, input project.DryRunKeywordsInput) (project.DryRunKeywordsOutput, error) {
	// Apply keyword sampling using the new sampling module
	samplingInput := sampling.SampleInput{
		Keywords: input.Keywords,
		UserID:   sc.UserID,
	}

	samplingResult, err := uc.sampler.Sample(ctx, samplingInput)
	if err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords: sampling failed: %v", err)
		return project.DryRunKeywordsOutput{}, err
	}

	// Generate UUID for job_id
	jobID := uuid.New().String()

	// Store job mapping in Redis before publishing to RabbitMQ
	// For dry-run jobs, projectID is empty since they're not associated with a specific project
	if err := uc.webhookUC.StoreJobMapping(ctx, jobID, sc.UserID, ""); err != nil {
		uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords.StoreJobMapping: jobID=%s, userID=%s, error=%v", jobID, sc.UserID, err)
		return project.DryRunKeywordsOutput{}, fmt.Errorf("failed to store job mapping: %w", err)
	}
	uc.l.Infof(ctx, "Stored job mapping: jobID=%s, userID=%s", jobID, sc.UserID)

	// Build DryRunCrawlRequest message with SAMPLED keywords
	payload := map[string]any{
		"keywords":          samplingResult.SelectedKeywords, // Use sampled keywords
		"limit_per_keyword": 3,                               // Limit items per keyword
		"include_comments":  true,                            // Include comments
		"max_comments":      5,                               // Limit comments per item
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
		return project.DryRunKeywordsOutput{}, err
	}

	uc.l.Infof(ctx, "Created dry-run job for user %s: %d/%d keywords selected (%.1f%%), method=%s, estimated_time=%v, job_id=%s",
		sc.UserID, len(samplingResult.SelectedKeywords), samplingResult.TotalKeywords,
		samplingResult.SamplingRatio*100, samplingResult.SelectionMethod, samplingResult.EstimatedTime, jobID)

	return project.DryRunKeywordsOutput{
		JobID:             jobID,
		Status:            "processing",
		SampledKeywords:   samplingResult.SelectedKeywords,
		TotalKeywords:     samplingResult.TotalKeywords,
		SamplingRatio:     samplingResult.SamplingRatio,
		EstimatedDuration: samplingResult.EstimatedTime,
	}, nil
}
