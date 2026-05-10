package repository

import "project-srv/internal/model"

type UpsertOptions struct {
	ProjectID string
	Enabled   bool
	Rules     []model.OntologySignalRule
	UpdatedBy string
}
