package microservice

import "time"

// IngestConfig is runtime config for the ingest microservice client.
type IngestConfig struct {
	BaseURL     string
	Timeout     time.Duration
	InternalKey string
}

// ActivationReadinessError describes one readiness blocker from ingest internal API.
type ActivationReadinessError struct {
	Code         string
	Message      string
	DataSourceID string
	TargetID     string
}

// ActivationReadiness is the readiness payload returned from ingest internal API.
type ActivationReadiness struct {
	ProjectID                string
	DataSourceCount          int
	HasDatasource            bool
	PassiveUnconfirmedCount  int
	MissingTargetDryrunCount int
	FailedTargetDryrunCount  int
	CanProceed               bool
	Errors                   []ActivationReadinessError
}
