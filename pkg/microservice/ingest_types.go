package microservice

import (
	"time"

	"github.com/smap-hcmut/shared-libs/go/contracts"
)

// IngestConfig is runtime config for the ingest microservice client.
type IngestConfig struct {
	BaseURL     string
	Timeout     time.Duration
	InternalKey string
}

// Type aliases for shared contract types used by the ingest HTTP client.
type ActivationReadinessError = contracts.ActivationReadinessError
type ActivationReadinessCommand = contracts.ActivationReadinessCommand
type ActivationReadinessInput = contracts.ActivationReadinessInput
type ActivationReadiness = contracts.ActivationReadiness

const (
	ActivationReadinessCommandActivate = contracts.ActivationReadinessCommandActivate
	ActivationReadinessCommandResume   = contracts.ActivationReadinessCommandResume
)
