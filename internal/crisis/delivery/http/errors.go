package http

import (
	"project-srv/internal/crisis"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors — all known errors are 400-level.
var (
	errNotFound                = pkgErrors.NewHTTPError(160001, "Crisis config not found")
	errProjectInvalid          = pkgErrors.NewHTTPError(160002, "Invalid project ID")
	errUpsertFailed            = pkgErrors.NewHTTPError(160003, "Failed to upsert crisis config")
	errDeleteFailed            = pkgErrors.NewHTTPError(160004, "Failed to delete crisis config")
	errWrongBody               = pkgErrors.NewHTTPError(160005, "Wrong request body")
	errNoTrigger               = pkgErrors.NewHTTPError(160006, "At least one trigger must be provided")
	errInvalidKeywordGroup     = pkgErrors.NewHTTPError(160007, "Keyword group requires non-empty name, at least 1 keyword, and weight > 0")
	errInvalidVolumeRule       = pkgErrors.NewHTTPError(160008, "Volume rule requires level (WARNING/CRITICAL), threshold > 0, and window > 0")
	errInvalidSentimentRule    = pkgErrors.NewHTTPError(160009, "Sentiment rule requires non-empty type")
	errInvalidInfluencerRule   = pkgErrors.NewHTTPError(160010, "Influencer rule requires non-empty type")
	errKeywordGroupsRequired   = pkgErrors.NewHTTPError(160011, "Enabled keywords trigger requires at least 1 group")
	errVolumeRulesRequired     = pkgErrors.NewHTTPError(160012, "Enabled volume trigger requires at least 1 rule")
	errSentimentRulesRequired  = pkgErrors.NewHTTPError(160013, "Enabled sentiment trigger requires at least 1 rule")
	errInfluencerRulesRequired = pkgErrors.NewHTTPError(160014, "Enabled influencer trigger requires at least 1 rule")
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
