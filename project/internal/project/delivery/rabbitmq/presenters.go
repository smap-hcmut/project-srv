package rabbitmq

import (
	"time"

	"smap-project/internal/model"

	"github.com/google/uuid"
)

// DryRunCrawlRequest represents the message sent to collector for dry-run keyword crawling
type DryRunCrawlRequest struct {
	JobID       string         `json:"job_id"`
	TaskType    string         `json:"task_type"` // "dryrun_keyword"
	Payload     map[string]any `json:"payload"`
	TimeRange   int            `json:"time_range,omitempty"`
	Attempt     int            `json:"attempt,omitempty"`
	MaxAttempts int            `json:"max_attempts,omitempty"`
	EmittedAt   time.Time      `json:"emitted_at"`
}

// DryRunPayload represents the payload for dry-run keyword crawling
type DryRunPayload struct {
	Keywords        []string `json:"keywords"`
	LimitPerKeyword int      `json:"limit_per_keyword"`
	IncludeComments bool     `json:"include_comments"`
	MaxComments     int      `json:"max_comments"`
}

// ToProjectCreatedEvent converts a domain Project to a ProjectCreatedEvent.
func ToProjectCreatedEvent(project model.Project) ProjectCreatedEvent {
	// Build competitor keywords map
	competitorKeywordsMap := make(map[string][]string)
	for _, ck := range project.CompetitorKeywords {
		competitorKeywordsMap[ck.CompetitorName] = ck.Keywords
	}

	return ProjectCreatedEvent{
		EventID:   uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Payload: ProjectCreatedPayload{
			ProjectID:             project.ID,
			UserID:                project.CreatedBy, // For progress notifications via WebSocket
			BrandName:             project.BrandName,
			BrandKeywords:         project.BrandKeywords,
			CompetitorNames:       project.CompetitorNames,
			CompetitorKeywordsMap: competitorKeywordsMap,
			DateRange: DateRange{
				From: project.FromDate.Format("2006-01-02"),
				To:   project.ToDate.Format("2006-01-02"),
			},
		},
	}
}
