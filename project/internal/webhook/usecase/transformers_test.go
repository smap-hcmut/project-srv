package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"smap-project/internal/webhook"
	"smap-project/pkg/log"
)

// createTestUsecase creates a usecase instance for testing
func createTestUsecase() *usecase {
	// Use existing mocks from the integration test file
	return &usecase{
		l:           log.NewNopLogger(),
		redisClient: newMockRedisClient(),
	}
}

func TestTransformDryRunCallback(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		name     string
		req      webhook.CallbackRequest
		expected webhook.JobMessage
	}{
		{
			name: "successful dry run with content",
			req: webhook.CallbackRequest{
				JobID:    "job_123",
				Status:   "success",
				Platform: "tiktok",
				Payload: webhook.CallbackPayload{
					Content: []webhook.Content{
						{
							Meta: webhook.ContentMeta{
								ID:            "content_456",
								KeywordSource: "test_keyword",
								PublishedAt:   time.Date(2024, 12, 10, 15, 30, 0, 0, time.UTC),
								Permalink:     "https://tiktok.com/test",
							},
							Content: webhook.ContentData{
								Text: "Test content text",
								Media: &webhook.ContentMedia{
									Type:      "video",
									VideoPath: "https://example.com/video.mp4",
								},
							},
							Author: webhook.ContentAuthor{
								ID:         "author_123",
								Username:   "testuser",
								Name:       "Test User",
								Followers:  1000,
								IsVerified: true,
								AvatarURL:  stringPtr("https://example.com/avatar.jpg"),
							},
							Interaction: webhook.ContentInteraction{
								Views:          50000,
								Likes:          1500,
								CommentsCount:  100,
								Shares:         50,
								EngagementRate: 3.2,
								UpdatedAt:      time.Date(2024, 12, 10, 15, 30, 0, 0, time.UTC),
							},
						},
					},
					Errors: []webhook.Error{},
				},
			},
			expected: webhook.JobMessage{
				Platform: webhook.PlatformTikTok,
				Status:   webhook.StatusCompleted,
				Batch: &webhook.BatchData{
					Keyword: "test_keyword",
					ContentList: []webhook.ContentItem{
						{
							ID:   "content_456",
							Text: "Test content text",
							Author: webhook.AuthorInfo{
								ID:         "author_123",
								Username:   "testuser",
								Name:       "Test User",
								Followers:  1000,
								IsVerified: true,
								AvatarURL:  "https://example.com/avatar.jpg",
							},
							Metrics: webhook.MetricsInfo{
								Views:    50000,
								Likes:    1500,
								Comments: 100,
								Shares:   50,
								Rate:     3.2,
							},
							Media: &webhook.MediaInfo{
								Type: webhook.MediaTypeVideo,
								URL:  "https://example.com/video.mp4",
							},
							PublishedAt: "2024-12-10T15:30:00Z",
							Permalink:   "https://tiktok.com/test",
						},
					},
				},
				Progress: &webhook.Progress{
					Current:    1,
					Total:      1,
					Percentage: 100.0,
					ETA:        0.0,
					Errors:     []string{},
				},
			},
		},
		{
			name: "failed dry run with errors",
			req: webhook.CallbackRequest{
				JobID:    "job_456",
				Status:   "failed",
				Platform: "youtube",
				Payload: webhook.CallbackPayload{
					Content: []webhook.Content{},
					Errors: []webhook.Error{
						{
							Code:    "RATE_LIMIT",
							Message: "Rate limit exceeded",
							Keyword: "test_keyword",
						},
						{
							Code:    "NETWORK_ERROR",
							Message: "Connection timeout",
						},
					},
				},
			},
			expected: webhook.JobMessage{
				Platform: webhook.PlatformYouTube,
				Status:   webhook.StatusFailed,
				Batch:    nil,
				Progress: &webhook.Progress{
					Current:    1,
					Total:      1,
					Percentage: 100.0,
					ETA:        0.0,
					Errors: []string{
						"[RATE_LIMIT] Rate limit exceeded (keyword: test_keyword)",
						"[NETWORK_ERROR] Connection timeout",
					},
				},
			},
		},
		{
			name: "processing status with unknown platform",
			req: webhook.CallbackRequest{
				JobID:    "job_789",
				Status:   "processing",
				Platform: "unknown_platform",
				Payload:  webhook.CallbackPayload{},
			},
			expected: webhook.JobMessage{
				Platform: webhook.PlatformTikTok, // Default fallback
				Status:   webhook.StatusProcessing,
				Batch:    nil,
				Progress: nil, // No progress for processing without content
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.TransformDryRunCallback(tt.req)
			
			// Assert platform and status
			assert.Equal(t, tt.expected.Platform, result.Platform)
			assert.Equal(t, tt.expected.Status, result.Status)
			
			// Assert batch data
			if tt.expected.Batch == nil {
				assert.Nil(t, result.Batch)
			} else {
				assert.NotNil(t, result.Batch)
				assert.Equal(t, tt.expected.Batch.Keyword, result.Batch.Keyword)
				assert.Len(t, result.Batch.ContentList, len(tt.expected.Batch.ContentList))
				
				// Check content items if any
				for i, expectedItem := range tt.expected.Batch.ContentList {
					actualItem := result.Batch.ContentList[i]
					assert.Equal(t, expectedItem.ID, actualItem.ID)
					assert.Equal(t, expectedItem.Text, actualItem.Text)
					assert.Equal(t, expectedItem.Author, actualItem.Author)
					assert.Equal(t, expectedItem.Metrics, actualItem.Metrics)
					assert.Equal(t, expectedItem.PublishedAt, actualItem.PublishedAt)
					assert.Equal(t, expectedItem.Permalink, actualItem.Permalink)
					
					if expectedItem.Media == nil {
						assert.Nil(t, actualItem.Media)
					} else {
						assert.Equal(t, expectedItem.Media.Type, actualItem.Media.Type)
						assert.Equal(t, expectedItem.Media.URL, actualItem.Media.URL)
					}
				}
			}
			
			// Assert progress
			if tt.expected.Progress == nil {
				assert.Nil(t, result.Progress)
			} else {
				assert.NotNil(t, result.Progress)
				assert.Equal(t, tt.expected.Progress.Current, result.Progress.Current)
				assert.Equal(t, tt.expected.Progress.Total, result.Progress.Total)
				assert.Equal(t, tt.expected.Progress.Percentage, result.Progress.Percentage)
				assert.Equal(t, tt.expected.Progress.ETA, result.Progress.ETA)
				assert.Equal(t, tt.expected.Progress.Errors, result.Progress.Errors)
			}
		})
	}
}

func TestTransformProjectCallback(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		name     string
		req      webhook.ProgressCallbackRequest
		expected webhook.ProjectMessage
	}{
		{
			name: "project in progress",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "proj_123",
				UserID:    "user_456",
				Status:    "PROCESSING",
				Total:     100,
				Done:      75,
				Errors:    0,
			},
			expected: webhook.ProjectMessage{
				Status: webhook.StatusProcessing,
				Progress: &webhook.Progress{
					Current:    75,
					Total:      100,
					Percentage: 75.0,
					ETA:        0.0,
					Errors:     []string{},
				},
			},
		},
		{
			name: "project completed",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "proj_456",
				UserID:    "user_789",
				Status:    "DONE",
				Total:     50,
				Done:      50,
				Errors:    0,
			},
			expected: webhook.ProjectMessage{
				Status: webhook.StatusCompleted,
				Progress: &webhook.Progress{
					Current:    50,
					Total:      50,
					Percentage: 100.0,
					ETA:        0.0,
					Errors:     []string{},
				},
			},
		},
		{
			name: "project failed with errors",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "proj_789",
				UserID:    "user_123",
				Status:    "FAILED",
				Total:     200,
				Done:      150,
				Errors:    25,
			},
			expected: webhook.ProjectMessage{
				Status: webhook.StatusFailed,
				Progress: &webhook.Progress{
					Current:    150,
					Total:      200,
					Percentage: 75.0,
					ETA:        0.0,
					Errors:     []string{"Processing encountered 25 errors"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.TransformProjectCallback(tt.req)
			
			assert.Equal(t, tt.expected.Status, result.Status)
			assert.NotNil(t, result.Progress)
			assert.Equal(t, tt.expected.Progress.Current, result.Progress.Current)
			assert.Equal(t, tt.expected.Progress.Total, result.Progress.Total)
			assert.Equal(t, tt.expected.Progress.Percentage, result.Progress.Percentage)
			assert.Equal(t, tt.expected.Progress.ETA, result.Progress.ETA)
			assert.Equal(t, tt.expected.Progress.Errors, result.Progress.Errors)
		})
	}
}

func TestMapPlatform(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		input    string
		expected webhook.Platform
	}{
		{"TIKTOK", webhook.PlatformTikTok},
		{"tiktok", webhook.PlatformTikTok},
		{"TikTok", webhook.PlatformTikTok},
		{"YOUTUBE", webhook.PlatformYouTube},
		{"youtube", webhook.PlatformYouTube},
		{"YouTube", webhook.PlatformYouTube},
		{"INSTAGRAM", webhook.PlatformInstagram},
		{"instagram", webhook.PlatformInstagram},
		{"Instagram", webhook.PlatformInstagram},
		{"unknown", webhook.PlatformTikTok}, // Default fallback
		{"", webhook.PlatformTikTok},        // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := uc.mapPlatform(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapDryRunStatus(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		input    string
		expected webhook.Status
	}{
		{"success", webhook.StatusCompleted},
		{"failed", webhook.StatusFailed},
		{"processing", webhook.StatusProcessing},
		{"unknown", webhook.StatusProcessing}, // Default fallback
		{"", webhook.StatusProcessing},        // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := uc.mapDryRunStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapProjectStatus(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		input    string
		expected webhook.Status
	}{
		{"DONE", webhook.StatusCompleted},
		{"done", webhook.StatusCompleted},
		{"FAILED", webhook.StatusFailed},
		{"failed", webhook.StatusFailed},
		{"INITIALIZING", webhook.StatusProcessing},
		{"CRAWLING", webhook.StatusProcessing},
		{"PROCESSING", webhook.StatusProcessing},
		{"processing", webhook.StatusProcessing},
		{"unknown", webhook.StatusProcessing}, // Default fallback
		{"", webhook.StatusProcessing},        // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := uc.mapProjectStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculatePercentage(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		name     string
		done     int64
		total    int64
		expected float64
	}{
		{"50% complete", 50, 100, 50.0},
		{"100% complete", 100, 100, 100.0},
		{"0% complete", 0, 100, 0.0},
		{"zero total", 0, 0, 0.0},
		{"partial progress", 33, 100, 33.0},
		{"over 100% (edge case)", 150, 100, 150.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.calculatePercentage(tt.done, tt.total)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformErrorsToStrings(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		name     string
		errors   []webhook.Error
		expected []string
	}{
		{
			name:     "empty errors",
			errors:   []webhook.Error{},
			expected: []string{},
		},
		{
			name: "single error with keyword",
			errors: []webhook.Error{
				{Code: "RATE_LIMIT", Message: "Rate limit exceeded", Keyword: "test_keyword"},
			},
			expected: []string{
				"[RATE_LIMIT] Rate limit exceeded (keyword: test_keyword)",
			},
		},
		{
			name: "single error without keyword",
			errors: []webhook.Error{
				{Code: "NETWORK_ERROR", Message: "Connection timeout"},
			},
			expected: []string{
				"[NETWORK_ERROR] Connection timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.transformErrorsToStrings(tt.errors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringValue(t *testing.T) {
	uc := createTestUsecase()

	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "valid string",
			input:    stringPtr("test_value"),
			expected: "test_value",
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.getStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}