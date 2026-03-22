package http

import (
	"net/http"

	"project-srv/internal/crisis"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors.
var (
	errNotFound                = &pkgErrors.HTTPError{Code: 160001, Message: "Crisis config not found", StatusCode: http.StatusBadRequest}
	errProjectInvalid          = &pkgErrors.HTTPError{Code: 160002, Message: "Invalid project ID", StatusCode: http.StatusBadRequest}
	errUpsertFailed            = &pkgErrors.HTTPError{Code: 160003, Message: "Failed to upsert crisis config", StatusCode: http.StatusInternalServerError}
	errDeleteFailed            = &pkgErrors.HTTPError{Code: 160004, Message: "Failed to delete crisis config", StatusCode: http.StatusInternalServerError}
	errWrongBody               = &pkgErrors.HTTPError{Code: 160005, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
	errNoTrigger               = &pkgErrors.HTTPError{Code: 160006, Message: "At least one trigger must be provided", StatusCode: http.StatusBadRequest}
	errInvalidKeywordGroup     = &pkgErrors.HTTPError{Code: 160007, Message: "Keyword group requires non-empty name, at least 1 keyword, and weight > 0", StatusCode: http.StatusBadRequest}
	errInvalidVolumeRule       = &pkgErrors.HTTPError{Code: 160008, Message: "Volume rule requires level (WARNING/CRITICAL), threshold > 0, and window > 0", StatusCode: http.StatusBadRequest}
	errInvalidSentimentRule    = &pkgErrors.HTTPError{Code: 160009, Message: "Sentiment rule requires non-empty type", StatusCode: http.StatusBadRequest}
	errInvalidInfluencerRule   = &pkgErrors.HTTPError{Code: 160010, Message: "Influencer rule requires non-empty type", StatusCode: http.StatusBadRequest}
	errKeywordGroupsRequired   = &pkgErrors.HTTPError{Code: 160011, Message: "Enabled keywords trigger requires at least 1 group", StatusCode: http.StatusBadRequest}
	errVolumeRulesRequired     = &pkgErrors.HTTPError{Code: 160012, Message: "Enabled volume trigger requires at least 1 rule", StatusCode: http.StatusBadRequest}
	errSentimentRulesRequired  = &pkgErrors.HTTPError{Code: 160013, Message: "Enabled sentiment trigger requires at least 1 rule", StatusCode: http.StatusBadRequest}
	errInfluencerRulesRequired = &pkgErrors.HTTPError{Code: 160014, Message: "Enabled influencer trigger requires at least 1 rule", StatusCode: http.StatusBadRequest}
)

// mapError maps UseCase domain errors to delivery HTTP errors.
func (h *handler) mapError(err error) error {
	switch err {
	case crisis.ErrNotFound:
		return errNotFound
	case crisis.ErrProjectInvalid:
		return errProjectInvalid
	case crisis.ErrUpsertFailed:
		return errUpsertFailed
	case crisis.ErrDeleteFailed:
		return errDeleteFailed
	default:
		panic(err)
	}
}
