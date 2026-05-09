package crisis

import (
	"project-srv/internal/model"
)

// UpsertInput is the input for creating or updating a crisis config.
type UpsertInput struct {
	ProjectID         string
	Status            *model.CrisisStatus
	KeywordsTrigger   *model.KeywordsTrigger
	VolumeTrigger     *model.VolumeTrigger
	SentimentTrigger  *model.SentimentTrigger
	InfluencerTrigger *model.InfluencerTrigger
	ResponsePolicy    *model.CrisisResponsePolicy
}

// UpsertOutput is the output after upserting a crisis config.
type UpsertOutput struct {
	CrisisConfig model.CrisisConfig
}

// DetailOutput is the output for getting crisis config detail.
type DetailOutput struct {
	CrisisConfig model.CrisisConfig
}

type RuntimeConfigOutput struct {
	ProjectID    string
	ProjectName  string
	CampaignID   string
	OwnerUserID  string
	CrisisConfig model.CrisisConfig
}

// ApplyRuntimeInput applies crisis status to ingest runtime crawl mode.
type ApplyRuntimeInput struct {
	ProjectID   string
	Status      *model.CrisisStatus
	CrisisLevel *model.CrisisRuntimeLevel
	Reason      string
	EventRef    string
}

type ApplyRuntimeOutput struct {
	ProjectID               string
	CrisisStatus            model.CrisisStatus
	CrisisLevel             model.CrisisRuntimeLevel
	AppliedCrawlMode        string
	AffectedDataSourceCount int
	NoopReason              string
}
