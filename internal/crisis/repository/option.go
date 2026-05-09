package repository

import "project-srv/internal/model"

// UpsertOptions contains the data needed to create or update a crisis config.
type UpsertOptions struct {
	ProjectID         string
	Status            *model.CrisisStatus
	KeywordsTrigger   *model.KeywordsTrigger
	VolumeTrigger     *model.VolumeTrigger
	SentimentTrigger  *model.SentimentTrigger
	InfluencerTrigger *model.InfluencerTrigger
	ResponsePolicy    *model.CrisisResponsePolicy
}
