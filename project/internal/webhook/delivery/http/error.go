package http

import (
	pkgErrors "smap-project/pkg/errors"
)

// HTTP error codes for webhook endpoints (31xxx range)
var (
	// errWrongBody is returned when request body parsing fails
	errWrongBody = pkgErrors.NewHTTPError(31001, "Wrong body")
)

// mapErrorCode maps domain errors to HTTP errors
// Currently passes through errors as-is since the usecase layer
// returns descriptive errors that are suitable for HTTP responses
func (h handler) mapErrorCode(err error) error {
	// Map domain errors to HTTP errors if needed
	// The usecase layer already returns well-formatted errors
	return err
}
