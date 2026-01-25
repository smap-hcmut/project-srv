package postgres

import (
	"context"

	"smap-project/internal/model"
	"smap-project/internal/project/repository"
)

// buildModelFromCreateOptions builds a model.Project from CreateOptions
func (r *implRepository) buildModelFromCreateOptions(ctx context.Context, sc model.Scope, opts repository.CreateOptions) (model.Project, error) {
	now := r.clock()

	// Extract competitor names from competitor keywords
	competitorNames := make([]string, 0, len(opts.CompetitorKeywords))
	for _, ck := range opts.CompetitorKeywords {
		competitorNames = append(competitorNames, ck.CompetitorName)
	}

	project := model.Project{
		Name:               opts.Name,
		Description:        opts.Description,
		Status:             opts.Status,
		FromDate:           opts.FromDate,
		ToDate:             opts.ToDate,
		BrandName:          opts.BrandName,
		CompetitorNames:    competitorNames,
		BrandKeywords:      opts.BrandKeywords,
		CompetitorKeywords: opts.CompetitorKeywords,
		CreatedBy:          opts.CreatedBy,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return project, nil
}

// buildModelFromUpdateOptions builds an updated model.Project by merging UpdateOptions with existing project
func (r *implRepository) buildModelFromUpdateOptions(ctx context.Context, existing model.Project, opts repository.UpdateOptions) (model.Project, error) {
	now := r.clock()

	// Start with existing project
	updated := existing

	// Update fields if provided
	if opts.Description != nil {
		updated.Description = opts.Description
	}

	if opts.Status != nil {
		updated.Status = *opts.Status
	}

	if opts.FromDate != nil {
		updated.FromDate = *opts.FromDate
	}

	if opts.ToDate != nil {
		updated.ToDate = *opts.ToDate
	}

	if opts.BrandKeywords != nil {
		updated.BrandKeywords = opts.BrandKeywords
	}

	if opts.CompetitorKeywords != nil {
		updated.CompetitorKeywords = opts.CompetitorKeywords
		// Update competitor names from competitor keywords
		competitorNames := make([]string, 0, len(opts.CompetitorKeywords))
		for _, ck := range opts.CompetitorKeywords {
			competitorNames = append(competitorNames, ck.CompetitorName)
		}
		updated.CompetitorNames = competitorNames
	}

	// Always update UpdatedAt
	updated.UpdatedAt = now

	return updated, nil
}
