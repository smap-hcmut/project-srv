package usecase

import (
	"fmt"
	"strings"
	"time"

	"smap-project/internal/webhook"
)

// TransformDryRunCallback transforms a crawler callback request into a structured JobMessage.
// This function handles the conversion of raw crawler data into the standardized message
// format used for Redis pub/sub communication.
//
// The transformation includes:
//   - Platform string to Platform enum mapping
//   - Status string to Status enum mapping
//   - Content array to BatchData with ContentItems
//   - Error array to Progress.Errors string array
//
// The function uses graceful degradation - if optional transformations fail,
// it continues with partial data rather than failing entirely.
func (uc *usecase) TransformDryRunCallback(req webhook.CallbackRequest) webhook.JobMessage {
	// Transform content to batch format
	var batch *webhook.BatchData
	if len(req.Payload.Content) > 0 {
		batch = &webhook.BatchData{
			Keyword:     uc.extractKeywordFromContent(req.Payload.Content),
			ContentList: uc.transformContentItems(req.Payload.Content),
			CrawledAt:   time.Now().Format(time.RFC3339),
		}
	}

	// Transform errors to progress format
	var progress *webhook.Progress
	if req.Status == "success" || len(req.Payload.Errors) > 0 {
		progress = &webhook.Progress{
			Current:    1, // Single batch for dry run
			Total:      1,
			Percentage: 100.0,
			ETA:        0.0,
			Errors:     uc.transformErrorsToStrings(req.Payload.Errors),
		}
	}

	return webhook.JobMessage{
		Platform: uc.mapPlatform(req.Platform),
		Status:   uc.mapDryRunStatus(req.Status),
		Batch:    batch,
		Progress: progress,
	}
}

// TransformProjectCallback transforms a collector progress callback into a structured ProjectMessage.
// This function handles the conversion of progress data into the standardized message
// format used for Redis pub/sub communication.
//
// Supports both old flat format (deprecated) and new phase-based format.
// Format detection: if Crawl.Total > 0 or Analyze.Total > 0, use new format.
//
// The transformation includes:
//   - Status string to Status enum mapping
//   - Progress calculation (current, total, percentage)
//   - Error count to error message array conversion
func (uc *usecase) TransformProjectCallback(req webhook.ProgressCallbackRequest) webhook.ProjectMessage {
	// Detect format: new format has phase data
	if uc.isNewProgressFormat(req) {
		return uc.transformNewProgressFormat(req)
	}
	return uc.transformOldProgressFormat(req)
}

// isNewProgressFormat detects if the request uses new phase-based format.
// Returns true if Crawl.Total > 0 or Analyze.Total > 0.
func (uc *usecase) isNewProgressFormat(req webhook.ProgressCallbackRequest) bool {
	return req.Crawl.Total > 0 || req.Analyze.Total > 0
}

// transformNewProgressFormat handles the new phase-based progress format.
// Builds a ProjectMessage with phase-specific progress data.
func (uc *usecase) transformNewProgressFormat(req webhook.ProgressCallbackRequest) webhook.ProjectMessage {
	// Calculate overall progress from phases
	crawlProgress := uc.calculatePhaseProgress(req.Crawl)
	analyzeProgress := uc.calculatePhaseProgress(req.Analyze)
	overallProgress := (crawlProgress + analyzeProgress) / 2

	// Use provided overall progress if available, otherwise calculate
	if req.OverallProgressPercent > 0 {
		overallProgress = req.OverallProgressPercent
	}

	// Build progress with phase data
	progress := &webhook.Progress{
		Current:    int(req.Crawl.Done + req.Analyze.Done),
		Total:      int(req.Crawl.Total + req.Analyze.Total),
		Percentage: overallProgress,
		ETA:        0.0,
		Errors:     uc.buildPhaseErrors(req.Crawl.Errors, req.Analyze.Errors),
	}

	return webhook.ProjectMessage{
		Status:   uc.mapProjectStatus(req.Status),
		Progress: progress,
	}
}

// transformOldProgressFormat handles the deprecated flat progress format.
// Logs a deprecation warning and converts to the standard ProjectMessage.
func (uc *usecase) transformOldProgressFormat(req webhook.ProgressCallbackRequest) webhook.ProjectMessage {
	progress := &webhook.Progress{
		Current:    int(req.Done),
		Total:      int(req.Total),
		Percentage: uc.calculatePercentage(req.Done, req.Total),
		ETA:        0.0,
		Errors:     uc.transformErrorCount(req.Errors),
	}

	return webhook.ProjectMessage{
		Status:   uc.mapProjectStatus(req.Status),
		Progress: progress,
	}
}

// calculatePhaseProgress calculates progress percentage for a single phase.
func (uc *usecase) calculatePhaseProgress(phase webhook.PhaseProgress) float64 {
	if phase.Total <= 0 {
		return 0.0
	}
	// Use provided progress_percent if available
	if phase.ProgressPercent > 0 {
		return phase.ProgressPercent
	}
	// Calculate from done + errors
	return float64(phase.Done+phase.Errors) / float64(phase.Total) * 100.0
}

// buildPhaseErrors combines errors from both phases into a string array.
func (uc *usecase) buildPhaseErrors(crawlErrors, analyzeErrors int64) []string {
	var errors []string
	if crawlErrors > 0 {
		errors = append(errors, fmt.Sprintf("Crawl phase encountered %d errors", crawlErrors))
	}
	if analyzeErrors > 0 {
		errors = append(errors, fmt.Sprintf("Analyze phase encountered %d errors", analyzeErrors))
	}
	return errors
}

// Helper methods for content transformation

// transformContentItems converts Content objects to ContentItem objects
func (uc *usecase) transformContentItems(contents []webhook.Content) []webhook.ContentItem {
	items := make([]webhook.ContentItem, len(contents))
	for i, content := range contents {
		items[i] = webhook.ContentItem{
			ID:   content.Meta.ID,
			Text: content.Content.Text,
			Author: webhook.AuthorInfo{
				ID:         content.Author.ID,
				Username:   content.Author.Username,
				Name:       content.Author.Name,
				Followers:  content.Author.Followers,
				IsVerified: content.Author.IsVerified,
				AvatarURL:  uc.getStringValue(content.Author.AvatarURL),
			},
			Metrics: webhook.MetricsInfo{
				Views:    content.Interaction.Views,
				Likes:    content.Interaction.Likes,
				Comments: content.Interaction.CommentsCount,
				Shares:   content.Interaction.Shares,
				Rate:     content.Interaction.EngagementRate,
			},
			Media:       uc.transformMedia(content.Content.Media),
			PublishedAt: content.Meta.PublishedAt.Format(time.RFC3339),
			Permalink:   content.Meta.Permalink,
		}
	}
	return items
}

// transformMedia converts ContentMedia to MediaInfo
func (uc *usecase) transformMedia(media *webhook.ContentMedia) *webhook.MediaInfo {
	if media == nil {
		return nil
	}
	return &webhook.MediaInfo{
		Type:      uc.mapMediaType(media.Type),
		Duration:  0,  // Map from content.Duration if available
		Thumbnail: "", // Map from appropriate field if available
		URL:       media.VideoPath,
	}
}

// transformErrorsToStrings converts Error objects to string messages
func (uc *usecase) transformErrorsToStrings(errors []webhook.Error) []string {
	if len(errors) == 0 {
		return []string{}
	}

	result := make([]string, len(errors))
	for i, err := range errors {
		result[i] = fmt.Sprintf("[%s] %s", err.Code, err.Message)
		if err.Keyword != "" {
			result[i] += fmt.Sprintf(" (keyword: %s)", err.Keyword)
		}
	}
	return result
}

// transformErrorCount converts error count to string messages
func (uc *usecase) transformErrorCount(errorCount int64) []string {
	if errorCount == 0 {
		return []string{}
	}
	return []string{
		fmt.Sprintf("Processing encountered %d errors", errorCount),
	}
}

// Mapping methods for enums

// mapPlatform maps string platform to Platform enum
func (uc *usecase) mapPlatform(platform string) webhook.Platform {
	switch strings.ToUpper(platform) {
	case "TIKTOK":
		return webhook.PlatformTikTok
	case "YOUTUBE":
		return webhook.PlatformYouTube
	case "INSTAGRAM":
		return webhook.PlatformInstagram
	default:
		return webhook.PlatformTikTok // Default fallback
	}
}

// mapDryRunStatus maps dry run callback status to Status enum
func (uc *usecase) mapDryRunStatus(status string) webhook.Status {
	switch status {
	case "success":
		return webhook.StatusCompleted
	case "failed":
		return webhook.StatusFailed
	default:
		return webhook.StatusProcessing
	}
}

// mapProjectStatus maps project callback status to Status enum
func (uc *usecase) mapProjectStatus(status string) webhook.Status {
	// Map from collector status to standard status
	switch strings.ToUpper(status) {
	case "DONE":
		return webhook.StatusCompleted
	case "FAILED":
		return webhook.StatusFailed
	case "INITIALIZING", "CRAWLING", "PROCESSING":
		return webhook.StatusProcessing
	default:
		return webhook.StatusProcessing
	}
}

// mapMediaType maps string media type to MediaType enum
func (uc *usecase) mapMediaType(mediaType string) webhook.MediaType {
	switch strings.ToLower(mediaType) {
	case "video":
		return webhook.MediaTypeVideo
	case "image":
		return webhook.MediaTypeImage
	case "audio":
		return webhook.MediaTypeAudio
	default:
		return webhook.MediaTypeVideo // Default fallback
	}
}

// Utility methods

// calculatePercentage calculates completion percentage
func (uc *usecase) calculatePercentage(done, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(done) / float64(total) * 100.0
}

// extractKeywordFromContent extracts keyword from content items
func (uc *usecase) extractKeywordFromContent(contents []webhook.Content) string {
	if len(contents) > 0 {
		return contents[0].Meta.KeywordSource
	}
	return ""
}

// getStringValue safely gets string value from pointer
func (uc *usecase) getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
