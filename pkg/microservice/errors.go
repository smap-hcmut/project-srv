package microservice

import "errors"

var (
	ErrBadRequest    = errors.New("ingest client bad request")
	ErrUnauthorized  = errors.New("ingest client unauthorized")
	ErrForbidden     = errors.New("ingest client forbidden")
	ErrRequestFailed = errors.New("ingest client request failed")
)
