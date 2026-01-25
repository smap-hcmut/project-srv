package model

import (
	"encoding/json"
	"slices"
	"time"

	"smap-project/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/types"
)

// CompetitorKeyword represents a competitor with its keywords
type CompetitorKeyword struct {
	CompetitorName string   `json:"competitor_name"`
	Keywords       []string `json:"keywords"`
}

// Project represents a project entity in the domain model
type Project struct {
	ID                 string
	Name               string
	Description        *string
	Status             string
	FromDate           time.Time
	ToDate             time.Time
	BrandName          string
	CompetitorNames    []string
	BrandKeywords      []string
	CompetitorKeywords []CompetitorKeyword // Array of competitor keywords
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

// NewProjectFromDB converts SQLBoiler Project to domain Project
func NewProjectFromDB(p *sqlboiler.Project) *Project {
	if p == nil {
		return nil
	}

	project := &Project{
		ID:              p.ID,
		Name:            p.Name,
		Status:          p.Status,
		FromDate:        p.FromDate,
		ToDate:          p.ToDate,
		BrandName:       p.BrandName,
		CompetitorNames: []string(p.CompetitorNames),
		BrandKeywords:   []string(p.BrandKeywords),
		CreatedBy:       p.CreatedBy,
	}

	if p.Description.Valid {
		desc := p.Description.String
		project.Description = &desc
	}

	if p.CompetitorKeywordsMap.Valid {
		// Try to unmarshal as array first (new format)
		var kwArray []CompetitorKeyword
		if err := json.Unmarshal(p.CompetitorKeywordsMap.JSON, &kwArray); err == nil {
			project.CompetitorKeywords = kwArray
		} else {
			// Fallback: try to unmarshal as map (old format) and convert to array
			var kwMap map[string][]string
			if err := json.Unmarshal(p.CompetitorKeywordsMap.JSON, &kwMap); err == nil {
				kwArray = make([]CompetitorKeyword, 0, len(kwMap))
				for name, keywords := range kwMap {
					kwArray = append(kwArray, CompetitorKeyword{
						CompetitorName: name,
						Keywords:       keywords,
					})
				}
				project.CompetitorKeywords = kwArray
			}
		}
	}

	if p.CreatedAt.Valid {
		project.CreatedAt = p.CreatedAt.Time
	}

	if p.UpdatedAt.Valid {
		project.UpdatedAt = p.UpdatedAt.Time
	}

	if p.DeletedAt.Valid {
		deletedAt := p.DeletedAt.Time
		project.DeletedAt = &deletedAt
	}

	return project
}

// ToDBProject converts domain Project to SQLBoiler Project
func (p *Project) ToDBProject() *sqlboiler.Project {
	dbProject := &sqlboiler.Project{
		ID:              p.ID,
		Name:            p.Name,
		Status:          p.Status,
		FromDate:        p.FromDate,
		ToDate:          p.ToDate,
		BrandName:       p.BrandName,
		CompetitorNames: types.StringArray(p.CompetitorNames),
		BrandKeywords:   types.StringArray(p.BrandKeywords),
		CreatedBy:       p.CreatedBy,
	}

	if p.Description != nil {
		dbProject.Description = null.StringFrom(*p.Description)
	}

	if len(p.CompetitorKeywords) > 0 {
		if kwArrayJSON, err := json.Marshal(p.CompetitorKeywords); err == nil {
			dbProject.CompetitorKeywordsMap = null.JSONFrom(kwArrayJSON)
		}
	}

	if !p.CreatedAt.IsZero() {
		dbProject.CreatedAt = null.TimeFrom(p.CreatedAt)
	}

	if !p.UpdatedAt.IsZero() {
		dbProject.UpdatedAt = null.TimeFrom(p.UpdatedAt)
	}

	if p.DeletedAt != nil {
		dbProject.DeletedAt = null.TimeFrom(*p.DeletedAt)
	}

	return dbProject
}

// ProjectStatus constants
const (
	ProjectStatusDraft     = "draft"
	ProjectStatusProcess   = "process"
	ProjectStatusCompleted = "completed"
)

// IsValidProjectStatus checks if the given status is valid
func IsValidProjectStatus(status string) bool {
	validStatuses := []string{
		ProjectStatusDraft,
		ProjectStatusProcess,
		ProjectStatusCompleted,
	}

	return slices.Contains(validStatuses, status)
}
