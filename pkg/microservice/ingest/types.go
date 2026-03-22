package ingest

import (
	"encoding/json"
	"project-srv/pkg/microservice"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l           log.Logger
	baseURL     string
	internalKey string
	client      pkghttp.Client
}

var _ microservice.IngestUseCase = (*implUseCase)(nil)

type responseEnvelope struct {
	ErrorCode int             `json:"error_code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
}

type readinessRespDTO struct {
	ProjectID                string                  `json:"project_id"`
	DataSourceCount          int                     `json:"datasource_count"`
	HasDatasource            bool                    `json:"has_datasource"`
	PassiveUnconfirmedCount  int                     `json:"passive_unconfirmed_count"`
	MissingTargetDryrunCount int                     `json:"missing_target_dryrun_count"`
	FailedTargetDryrunCount  int                     `json:"failed_target_dryrun_count"`
	CanProceed               bool                    `json:"can_proceed"`
	Errors                   []readinessErrorRespDTO `json:"errors"`
}

type readinessErrorRespDTO struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	DataSourceID string `json:"datasource_id,omitempty"`
	TargetID     string `json:"target_id,omitempty"`
}
