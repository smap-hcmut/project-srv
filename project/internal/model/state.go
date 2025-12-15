package model

// ProjectStatus represents the current state of a project in the processing pipeline.
type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "INITIALIZING"
	ProjectStatusCrawling     ProjectStatus = "CRAWLING" // Deprecated: use PROCESSING with phase data
	ProjectStatusProcessing   ProjectStatus = "PROCESSING"
	ProjectStatusDone         ProjectStatus = "DONE"
	ProjectStatusFailed       ProjectStatus = "FAILED"
)

// ProjectState is stored in Redis as a Hash with key pattern: smap:proj:{projectID}
// Supports both old flat format and new phase-based format for backward compatibility.
type ProjectState struct {
	Status ProjectStatus `json:"status"`

	// Crawl phase counters
	CrawlTotal  int64 `json:"crawl_total"`
	CrawlDone   int64 `json:"crawl_done"`
	CrawlErrors int64 `json:"crawl_errors"`

	// Analyze phase counters
	AnalyzeTotal  int64 `json:"analyze_total"`
	AnalyzeDone   int64 `json:"analyze_done"`
	AnalyzeErrors int64 `json:"analyze_errors"`

	// Old flat format fields (deprecated, kept for backward compatibility)
	Total  int64 `json:"total,omitempty"`
	Done   int64 `json:"done,omitempty"`
	Errors int64 `json:"errors,omitempty"`
}

// CrawlProgressPercent calculates the crawl phase completion percentage.
// Returns 0 if CrawlTotal is zero to avoid division by zero.
// Errors count toward progress (item processed, even if failed).
func (s *ProjectState) CrawlProgressPercent() float64 {
	if s.CrawlTotal <= 0 {
		return 0
	}
	return float64(s.CrawlDone+s.CrawlErrors) / float64(s.CrawlTotal) * 100
}

// AnalyzeProgressPercent calculates the analyze phase completion percentage.
// Returns 0 if AnalyzeTotal is zero to avoid division by zero.
// Errors count toward progress (item processed, even if failed).
func (s *ProjectState) AnalyzeProgressPercent() float64 {
	if s.AnalyzeTotal <= 0 {
		return 0
	}
	return float64(s.AnalyzeDone+s.AnalyzeErrors) / float64(s.AnalyzeTotal) * 100
}

// OverallProgressPercent calculates the overall progress as weighted average of phases.
// Both phases have equal weight (50% each).
func (s *ProjectState) OverallProgressPercent() float64 {
	return s.CrawlProgressPercent()/2 + s.AnalyzeProgressPercent()/2
}
