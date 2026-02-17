package crisis

import (
	"project-srv/internal/model"
)

// UpsertInput is the input for creating or updating a crisis config.
type UpsertInput struct {
	ProjectID         string
	KeywordsTrigger   *model.KeywordsTrigger
	VolumeTrigger     *model.VolumeTrigger
	SentimentTrigger  *model.SentimentTrigger
	InfluencerTrigger *model.InfluencerTrigger
}

// UpsertOutput is the output after upserting a crisis config.
type UpsertOutput struct {
	CrisisConfig model.CrisisConfig
}

// DetailOutput is the output for getting crisis config detail.
type DetailOutput struct {
	CrisisConfig model.CrisisConfig
}
