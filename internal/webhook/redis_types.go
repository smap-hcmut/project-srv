// Package webhook provides types and handlers for processing webhook callbacks
// from crawler and collector services, transforming them into structured messages
// for Redis pub/sub communication with the WebSocket service.
//
// # Topic Patterns
//
// The package uses topic-specific routing patterns for Redis pub/sub:
//   - Dry-run jobs: job:{jobID}:{userID}
//   - Project progress: project:{projectID}:{userID}
//
// # Message Types
//
// Two primary message types are supported:
//   - JobMessage: For dry-run job notifications with crawled content
//   - ProjectMessage: For project progress updates
//
// # Usage
//
// Messages are published to Redis using the topic patterns above.
// The WebSocket service subscribes to patterns like "job:*" and "project:*"
// to receive and route messages to connected clients.
package webhook

// Platform represents supported social media platforms for crawling.
// Used to identify the source platform of crawled content.
type Platform string

const (
	// PlatformTikTok represents TikTok social media platform
	PlatformTikTok Platform = "TIKTOK"
	// PlatformYouTube represents YouTube video platform
	PlatformYouTube Platform = "YOUTUBE"
	// PlatformInstagram represents Instagram social media platform
	PlatformInstagram Platform = "INSTAGRAM"
)

// Status represents the processing state of a job or project.
// Used to communicate current progress to connected clients.
type Status string

const (
	// StatusProcessing indicates the job/project is currently being processed
	StatusProcessing Status = "PROCESSING"
	// StatusCompleted indicates the job/project has finished successfully
	StatusCompleted Status = "COMPLETED"
	// StatusFailed indicates the job/project encountered an error
	StatusFailed Status = "FAILED"
	// StatusPaused indicates the job/project is temporarily paused
	StatusPaused Status = "PAUSED"
)

// MediaType represents the type of media content in crawled items.
type MediaType string

const (
	MediaTypeVideo MediaType = "video"
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
)

// JobMessage represents dry run job notifications
// Published to topic: job:{jobID}:{userID}
type JobMessage struct {
	Platform Platform   `json:"platform"`           // Platform where the job ran
	Status   Status     `json:"status"`             // Current job status
	Batch    *BatchData `json:"batch,omitempty"`    // Batch data for completed crawls
	Progress *Progress  `json:"progress,omitempty"` // Job progress information
}

// ProjectMessage represents project progress notifications
// Published to topic: project:{projectID}:{userID}
type ProjectMessage struct {
	Status   Status    `json:"status"`             // Current project status
	Progress *Progress `json:"progress,omitempty"` // Overall project progress
}

// Progress represents progress information with completion metrics and errors
type Progress struct {
	Current    int      `json:"current"`    // Current completed items
	Total      int      `json:"total"`      // Total items to process
	Percentage float64  `json:"percentage"` // Completion percentage (0-100)
	ETA        float64  `json:"eta"`        // Estimated time remaining in minutes
	Errors     []string `json:"errors"`     // Array of error messages
}

// BatchData represents a batch of crawled content from a dry run
type BatchData struct {
	Keyword     string        `json:"keyword"`      // Search keyword used for crawling
	ContentList []ContentItem `json:"content_list"` // Crawled content items
	CrawledAt   string        `json:"crawled_at"`   // ISO timestamp when batch was crawled
}

// ContentItem represents a simplified content item for Redis messaging
type ContentItem struct {
	ID          string      `json:"id"`              // Content unique ID
	Text        string      `json:"text"`            // Content text/caption
	Author      AuthorInfo  `json:"author"`          // Author information
	Metrics     MetricsInfo `json:"metrics"`         // Engagement statistics
	Media       *MediaInfo  `json:"media,omitempty"` // Media information (optional)
	PublishedAt string      `json:"published_at"`    // ISO timestamp when content was published
	Permalink   string      `json:"permalink"`       // Direct link to content
}

// AuthorInfo represents author information for content items
type AuthorInfo struct {
	ID         string `json:"id"`          // Author unique ID
	Username   string `json:"username"`    // Author username/handle
	Name       string `json:"name"`        // Author display name
	Followers  int    `json:"followers"`   // Follower count
	IsVerified bool   `json:"is_verified"` // Verification status
	AvatarURL  string `json:"avatar_url"`  // Profile picture URL
}

// MetricsInfo represents engagement metrics for content items
type MetricsInfo struct {
	Views    int     `json:"views"`    // View count
	Likes    int     `json:"likes"`    // Like count
	Comments int     `json:"comments"` // Comment count
	Shares   int     `json:"shares"`   // Share count
	Rate     float64 `json:"rate"`     // Engagement rate percentage
}

// MediaInfo represents media information for content items
type MediaInfo struct {
	Type      MediaType `json:"type"`               // Media type (video, image, audio)
	Duration  int       `json:"duration,omitempty"` // Duration in seconds (for video/audio)
	Thumbnail string    `json:"thumbnail"`          // Thumbnail/preview URL
	URL       string    `json:"url"`                // Media file URL
}
