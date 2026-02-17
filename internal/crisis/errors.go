package crisis

import "errors"

// Domain errors — returned by UseCase layer.
var (
	ErrNotFound       = errors.New("crisis config not found")
	ErrProjectInvalid = errors.New("invalid project id")
	ErrUpsertFailed   = errors.New("failed to upsert crisis config")
	ErrDeleteFailed   = errors.New("failed to delete crisis config")
)
