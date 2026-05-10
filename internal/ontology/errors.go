package ontology

import "errors"

var (
	ErrNotFound       = errors.New("ontology rules not found")
	ErrProjectInvalid = errors.New("invalid project id")
	ErrInvalidRules   = errors.New("invalid ontology rules")
	ErrUpsertFailed   = errors.New("failed to upsert ontology rules")
	ErrDeleteFailed   = errors.New("failed to delete ontology rules")
)
