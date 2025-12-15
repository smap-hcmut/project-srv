package webhook

import "time"

// WebSocket message types
const (
	MessageTypeProjectProgress  = "project_progress"
	MessageTypeProjectCompleted = "project_completed"
	MessageTypeDryRunResult     = "dryrun_result"
)

// CallbackRequest represents the webhook callback from collector service
type CallbackRequest struct {
	JobID    string          `json:"job_id" binding:"required"`
	Status   string          `json:"status" binding:"required,oneof=success failed"`
	Platform string          `json:"platform" binding:"required,oneof=youtube tiktok"`
	Payload  CallbackPayload `json:"payload"`
	// UserID is optional for backward compatibility during migration.
	// The project service will look up UserID from JobID using Redis.
	UserID string `json:"user_id,omitempty"`
}

// CallbackPayload contains the crawl results or errors.
// The Content field is named to maintain consistency with the crawler's data structure
// where the payload contains content items rather than posts.
type CallbackPayload struct {
	Content []Content `json:"content,omitempty"` // Array of crawled content items
	Errors  []Error   `json:"errors,omitempty"`
}

// Content represents a single social media content item with full crawler data
type Content struct {
	Meta        ContentMeta        `json:"meta"`
	Content     ContentData        `json:"content"`
	Interaction ContentInteraction `json:"interaction"`
	Author      ContentAuthor      `json:"author"`
	Comments    []Comment          `json:"comments,omitempty"`
}

// ContentMeta contains metadata about the content
type ContentMeta struct {
	ID              string    `json:"id"`
	Platform        string    `json:"platform"`
	JobID           string    `json:"job_id"`
	CrawledAt       time.Time `json:"crawled_at"`
	PublishedAt     time.Time `json:"published_at"`
	Permalink       string    `json:"permalink"`
	KeywordSource   string    `json:"keyword_source"`
	Lang            string    `json:"lang"`
	Region          string    `json:"region"`
	PipelineVersion string    `json:"pipeline_version"`
	FetchStatus     string    `json:"fetch_status"`
	FetchError      *string   `json:"fetch_error"`
}

// ContentData contains the content data
type ContentData struct {
	Text          string        `json:"text"`
	Duration      int           `json:"duration,omitempty"`
	Hashtags      []string      `json:"hashtags,omitempty"`
	SoundName     string        `json:"sound_name,omitempty"`
	Category      *string       `json:"category,omitempty"`
	Title         *string       `json:"title,omitempty"` // YouTube only
	Media         *ContentMedia `json:"media,omitempty"`
	Transcription *string       `json:"transcription,omitempty"`
}

// ContentMedia contains media information
type ContentMedia struct {
	Type         string    `json:"type"`
	VideoPath    string    `json:"video_path,omitempty"`
	AudioPath    string    `json:"audio_path,omitempty"`
	DownloadedAt time.Time `json:"downloaded_at,omitempty"`
}

// ContentInteraction contains engagement metrics
type ContentInteraction struct {
	Views          int       `json:"views"`
	Likes          int       `json:"likes"`
	CommentsCount  int       `json:"comments_count"`
	Shares         int       `json:"shares"`
	Saves          int       `json:"saves,omitempty"`
	EngagementRate float64   `json:"engagement_rate,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ContentAuthor contains author information
type ContentAuthor struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Username       string  `json:"username"`
	Followers      int     `json:"followers"`
	Following      int     `json:"following"`
	Likes          int     `json:"likes"`
	Videos         int     `json:"videos"`
	IsVerified     bool    `json:"is_verified"`
	Bio            string  `json:"bio,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	ProfileURL     string  `json:"profile_url"`
	Country        *string `json:"country,omitempty"`          // YouTube only
	TotalViewCount *int    `json:"total_view_count,omitempty"` // YouTube only
}

// Comment represents a comment on a post
type Comment struct {
	ID           string      `json:"id"`
	ParentID     *string     `json:"parent_id,omitempty"`
	PostID       string      `json:"post_id"`
	User         CommentUser `json:"user"`
	Text         string      `json:"text"`
	Likes        int         `json:"likes"`
	RepliesCount int         `json:"replies_count"`
	PublishedAt  time.Time   `json:"published_at"`
	IsAuthor     bool        `json:"is_author"`
	Media        *string     `json:"media,omitempty"`
	IsFavorited  bool        `json:"is_favorited"` // YouTube only
}

// CommentUser contains comment author information
type CommentUser struct {
	ID        *string `json:"id,omitempty"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// Error represents a crawl error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Keyword string `json:"keyword,omitempty"`
}

// PhaseProgress represents progress data for a single processing phase (crawl or analyze).
// Used in webhook callbacks to track granular progress per phase.
type PhaseProgress struct {
	Total           int64   `json:"total"`            // Total items to process in this phase
	Done            int64   `json:"done"`             // Completed items in this phase
	Errors          int64   `json:"errors"`           // Failed items in this phase
	ProgressPercent float64 `json:"progress_percent"` // Completion percentage (0-100)
}

// ProgressCallbackRequest represents the webhook callback for progress updates from collector.
// Supports both old flat format (deprecated) and new phase-based format.
type ProgressCallbackRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Status    string `json:"status" binding:"required"` // INITIALIZING, PROCESSING, DONE, FAILED

	// New phase-based progress fields
	Crawl                  PhaseProgress `json:"crawl"`                    // Crawl phase progress
	Analyze                PhaseProgress `json:"analyze"`                  // Analyze phase progress
	OverallProgressPercent float64       `json:"overall_progress_percent"` // Overall progress (0-100)

	// Old flat format fields (deprecated, kept for backward compatibility)
	Total  int64 `json:"total,omitempty"`
	Done   int64 `json:"done,omitempty"`
	Errors int64 `json:"errors,omitempty"`
}

// JobMappingData represents the data stored in Redis for job mappings
// This is used to map JobID to UserID and ProjectID for webhook callbacks
type JobMappingData struct {
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
}
