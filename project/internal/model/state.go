package model

// ProjectStatus represents the current state of a project in the processing pipeline.
type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "INITIALIZING"
	ProjectStatusCrawling     ProjectStatus = "CRAWLING"
	ProjectStatusProcessing   ProjectStatus = "PROCESSING"
	ProjectStatusDone         ProjectStatus = "DONE"
	ProjectStatusFailed       ProjectStatus = "FAILED"
)

// This is stored in Redis as a Hash with key pattern: smap:proj:{projectID}
type ProjectState struct {
	Status ProjectStatus `json:"status"`
	Total  int64         `json:"total"`
	Done   int64         `json:"done"`
	Errors int64         `json:"errors"`
}
