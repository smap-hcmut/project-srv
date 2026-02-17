package repository

import "project-srv/internal/model"

// UpsertOptions contains the data needed to create or update a crisis config.
type UpsertOptions struct {
	ProjectID         string
	KeywordsTrigger   *model.KeywordsTrigger
	VolumeTrigger     *model.VolumeTrigger
	SentimentTrigger  *model.SentimentTrigger
	InfluencerTrigger *model.InfluencerTrigger
}
