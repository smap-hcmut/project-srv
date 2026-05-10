package model

import "time"

type OntologyTargetKind string

const (
	OntologyTargetKindAspect OntologyTargetKind = "ASPECT"
	OntologyTargetKindIssue  OntologyTargetKind = "ISSUE"
	OntologyTargetKindTopic  OntologyTargetKind = "TOPIC"
)

type OntologyMatchMode string

const (
	OntologyMatchModeAny   OntologyMatchMode = "ANY"
	OntologyMatchModeAll   OntologyMatchMode = "ALL"
	OntologyMatchModeRegex OntologyMatchMode = "REGEX"
)

type OntologySignalRule struct {
	ID              string             `json:"id"`
	Label           string             `json:"label"`
	Description     string             `json:"description,omitempty"`
	TargetKind      OntologyTargetKind `json:"target_kind"`
	TargetKey       string             `json:"target_key"`
	MatchMode       OntologyMatchMode  `json:"match_mode"`
	Phrases         []string           `json:"phrases"`
	Patterns        []string           `json:"patterns"`
	NegativePhrases []string           `json:"negative_phrases,omitempty"`
	Enabled         bool               `json:"enabled"`
	Weight          int                `json:"weight"`
	SampleText      string             `json:"sample_text,omitempty"`
}

type ProjectOntologyRules struct {
	ProjectID string               `json:"project_id"`
	Enabled   bool                 `json:"enabled"`
	Rules     []OntologySignalRule `json:"rules"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

func DefaultProjectOntologyRules(projectID string) ProjectOntologyRules {
	return ProjectOntologyRules{
		ProjectID: projectID,
		Enabled:   true,
		Rules:     []OntologySignalRule{},
	}
}
