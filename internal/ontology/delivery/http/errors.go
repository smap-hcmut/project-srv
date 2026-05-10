package http

import (
	"net/http"

	"project-srv/internal/ontology"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errNotFound       = &pkgErrors.HTTPError{Code: 161001, Message: "Ontology rules not found", StatusCode: http.StatusNotFound}
	errProjectInvalid = &pkgErrors.HTTPError{Code: 161002, Message: "Invalid project ID", StatusCode: http.StatusBadRequest}
	errInvalidRules   = &pkgErrors.HTTPError{Code: 161003, Message: "Invalid ontology rules", StatusCode: http.StatusBadRequest}
	errUpsertFailed   = &pkgErrors.HTTPError{Code: 161004, Message: "Failed to save ontology rules", StatusCode: http.StatusInternalServerError}
	errDeleteFailed   = &pkgErrors.HTTPError{Code: 161005, Message: "Failed to delete ontology rules", StatusCode: http.StatusInternalServerError}
	errWrongBody      = &pkgErrors.HTTPError{Code: 161006, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
)

func (h *handler) mapError(err error) error {
	switch err {
	case ontology.ErrNotFound:
		return errNotFound
	case ontology.ErrProjectInvalid:
		return errProjectInvalid
	case ontology.ErrInvalidRules:
		return errInvalidRules
	case ontology.ErrUpsertFailed:
		return errUpsertFailed
	case ontology.ErrDeleteFailed:
		return errDeleteFailed
	default:
		panic(err)
	}
}
