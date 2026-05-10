package ontology

import (
	"context"

	"project-srv/internal/model"
)

type UpsertInput struct {
	ProjectID string
	Enabled   bool
	Rules     []model.OntologySignalRule
	UpdatedBy string
}

type UpsertOutput struct {
	Config model.ProjectOntologyRules
}

type DetailOutput struct {
	Config model.ProjectOntologyRules
}

type RuntimeOutput struct {
	ProjectID string
	Config    model.ProjectOntologyRules
}

type TestInput struct {
	ProjectID string
	Enabled   bool
	Rules     []model.OntologySignalRule
	Text      string
}

type TestMatch struct {
	RuleID     string   `json:"rule_id"`
	Label      string   `json:"label"`
	TargetKind string   `json:"target_kind"`
	TargetKey  string   `json:"target_key"`
	Matched    bool     `json:"matched"`
	Evidence   []string `json:"evidence"`
}

type TestOutput struct {
	Matches []TestMatch
}

type UseCase interface {
	Upsert(ctx context.Context, input UpsertInput) (UpsertOutput, error)
	Detail(ctx context.Context, projectID string) (DetailOutput, error)
	Runtime(ctx context.Context, projectID string) (RuntimeOutput, error)
	Test(ctx context.Context, input TestInput) (TestOutput, error)
	Delete(ctx context.Context, projectID string) error
}
