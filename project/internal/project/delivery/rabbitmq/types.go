package rabbitmq

import "time"

// ProjectCreatedEvent is published when a new project is created.
type ProjectCreatedEvent struct {
	EventID   string                `json:"event_id"`
	Timestamp time.Time             `json:"timestamp"`
	Payload   ProjectCreatedPayload `json:"payload"`
}

// ProjectCreatedPayload includes details for the created project.
type ProjectCreatedPayload struct {
	ProjectID             string              `json:"project_id"`
	UserID                string              `json:"user_id"` // For progress notifications via WebSocket
	BrandName             string              `json:"brand_name"`
	BrandKeywords         []string            `json:"brand_keywords"`
	CompetitorNames       []string            `json:"competitor_names"`
	CompetitorKeywordsMap map[string][]string `json:"competitor_keywords_map"`
	DateRange             DateRange           `json:"date_range"`
}

// DateRange specifies a data collection period.
type DateRange struct {
	From string `json:"from"` // Format: YYYY-MM-DD
	To   string `json:"to"`   // Format: YYYY-MM-DD
}
