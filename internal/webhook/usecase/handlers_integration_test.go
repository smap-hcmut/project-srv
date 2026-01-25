package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"smap-project/internal/webhook"
	"smap-project/pkg/log"

	"github.com/stretchr/testify/assert"
)

// extendedMockRedisClient extends the existing mock to track published messages
type extendedMockRedisClient struct {
	*mockRedisClient
	publishedChannels map[string][]byte
	publishCount      int
}

func newExtendedMockRedisClient() *extendedMockRedisClient {
	return &extendedMockRedisClient{
		mockRedisClient:   newMockRedisClient(),
		publishedChannels: make(map[string][]byte),
		publishCount:      0,
	}
}

func (m *extendedMockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	m.publishCount++

	// Convert message to bytes
	var bytes []byte
	switch v := message.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		// This shouldn't happen in normal usage, but handle it gracefully
		jsonBytes, _ := json.Marshal(v)
		bytes = jsonBytes
	}

	m.publishedChannels[channel] = bytes
	return nil
}

func TestHandleDryRunCallbackIntegration(t *testing.T) {
	ctx := context.Background()
	mockRedis := newExtendedMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	tests := []struct {
		name            string
		setupJobMapping bool
		jobID           string
		userID          string
		projectID       string
		req             webhook.CallbackRequest
		expectedChannel string
		wantError       bool
	}{
		{
			name:            "successful dry run with new topic pattern",
			setupJobMapping: true,
			jobID:           "test_job_123",
			userID:          "user_456",
			projectID:       "project_789",
			req: webhook.CallbackRequest{
				JobID:    "test_job_123",
				Status:   "success",
				Platform: "tiktok",
				Payload: webhook.CallbackPayload{
					Content: []webhook.Content{
						{
							Meta: webhook.ContentMeta{
								ID:            "content_1",
								KeywordSource: "test_keyword",
								PublishedAt:   time.Date(2024, 12, 10, 15, 30, 0, 0, time.UTC),
								Permalink:     "https://tiktok.com/test",
							},
							Content: webhook.ContentData{
								Text: "Test content",
							},
							Author: webhook.ContentAuthor{
								ID:         "author_1",
								Username:   "testuser",
								Name:       "Test User",
								Followers:  1000,
								IsVerified: true,
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
			expectedChannel: "job:test_job_123:user_456",
			wantError:       false,
		},
		{
			name:            "failed dry run with errors and new topic pattern",
			setupJobMapping: true,
			jobID:           "test_job_456",
			userID:          "user_789",
			projectID:       "project_123",
			req: webhook.CallbackRequest{
				JobID:    "test_job_456",
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
					},
				},
			},
			expectedChannel: "job:test_job_456:user_789",
			wantError:       false,
		},
		{
			name:            "dry run without job mapping fails",
			setupJobMapping: false,
			jobID:           "nonexistent_job",
			req: webhook.CallbackRequest{
				JobID:    "nonexistent_job",
				Status:   "success",
				Platform: "tiktok",
				Payload:  webhook.CallbackPayload{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous test data
			mockRedis.publishedChannels = make(map[string][]byte)
			mockRedis.publishCount = 0

			// Setup job mapping if required
			if tt.setupJobMapping {
				err := uc.StoreJobMapping(ctx, tt.jobID, tt.userID, tt.projectID)
				assert.NoError(t, err, "Failed to setup job mapping")
			}

			// Execute the handler
			err := uc.HandleDryRunCallback(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, 0, mockRedis.publishCount, "Should not publish on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, mockRedis.publishCount, "Should publish exactly once")

				// Verify the correct channel was used
				_, channelExists := mockRedis.publishedChannels[tt.expectedChannel]
				assert.True(t, channelExists, "Expected channel should exist: %s", tt.expectedChannel)

				// Verify message structure
				messageBytes := mockRedis.publishedChannels[tt.expectedChannel]
				var jobMessage webhook.JobMessage
				err := json.Unmarshal(messageBytes, &jobMessage)
				assert.NoError(t, err, "Should be able to unmarshal JobMessage")

				// Verify message content
				expectedPlatform := webhook.PlatformTikTok
				if tt.req.Platform == "youtube" {
					expectedPlatform = webhook.PlatformYouTube
				}
				assert.Equal(t, expectedPlatform, jobMessage.Platform)

				expectedStatus := webhook.StatusCompleted
				if tt.req.Status == "failed" {
					expectedStatus = webhook.StatusFailed
				}
				assert.Equal(t, expectedStatus, jobMessage.Status)

				// Verify batch and progress data based on request content
				if len(tt.req.Payload.Content) > 0 {
					assert.NotNil(t, jobMessage.Batch, "Batch should be present when content exists")
					assert.NotEmpty(t, jobMessage.Batch.ContentList, "Content list should not be empty")
				}

				if tt.req.Status == "success" || len(tt.req.Payload.Errors) > 0 {
					assert.NotNil(t, jobMessage.Progress, "Progress should be present")
					assert.Equal(t, 1, jobMessage.Progress.Current, "Current should be 1 for dry run")
					assert.Equal(t, 1, jobMessage.Progress.Total, "Total should be 1 for dry run")
					assert.Equal(t, 100.0, jobMessage.Progress.Percentage, "Percentage should be 100.0 for dry run")
				}
			}
		})
	}
}

func TestHandleProgressCallbackIntegration(t *testing.T) {
	ctx := context.Background()
	mockRedis := newExtendedMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	tests := []struct {
		name            string
		req             webhook.ProgressCallbackRequest
		expectedChannel string
		wantError       bool
	}{
		{
			name: "project progress with new topic pattern",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "proj_123",
				UserID:    "user_456",
				Status:    "PROCESSING",
				Total:     100,
				Done:      75,
				Errors:    0,
			},
			expectedChannel: "project:proj_123:user_456",
			wantError:       false,
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
			expectedChannel: "project:proj_456:user_789",
			wantError:       false,
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
			expectedChannel: "project:proj_789:user_123",
			wantError:       false,
		},
		{
			name: "missing project ID should fail",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "",
				UserID:    "user_456",
				Status:    "PROCESSING",
				Total:     100,
				Done:      75,
				Errors:    0,
			},
			wantError: true,
		},
		{
			name: "missing user ID should fail",
			req: webhook.ProgressCallbackRequest{
				ProjectID: "proj_123",
				UserID:    "",
				Status:    "PROCESSING",
				Total:     100,
				Done:      75,
				Errors:    0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous test data
			mockRedis.publishedChannels = make(map[string][]byte)
			mockRedis.publishCount = 0

			// Execute the handler
			err := uc.HandleProgressCallback(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, 0, mockRedis.publishCount, "Should not publish on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, mockRedis.publishCount, "Should publish exactly once")

				// Verify the correct channel was used
				_, channelExists := mockRedis.publishedChannels[tt.expectedChannel]
				assert.True(t, channelExists, "Expected channel should exist: %s", tt.expectedChannel)

				// Verify message structure (now WebSocket message format)
				messageBytes := mockRedis.publishedChannels[tt.expectedChannel]
				var wsMessage map[string]interface{}
				err := json.Unmarshal(messageBytes, &wsMessage)
				assert.NoError(t, err, "Should be able to unmarshal WebSocket message")

				// Verify message type
				expectedType := "project_progress"
				if tt.req.Status == "DONE" || tt.req.Status == "FAILED" {
					expectedType = "project_completed"
				}
				assert.Equal(t, expectedType, wsMessage["type"])

				// Verify payload
				payload := wsMessage["payload"].(map[string]interface{})
				assert.Equal(t, tt.req.Status, payload["status"])
				assert.Equal(t, tt.req.ProjectID, payload["project_id"])
			}
		})
	}
}

func TestTopicPatternConsistency(t *testing.T) {
	// Test that topic patterns follow the exact specification
	tests := []struct {
		name                 string
		jobID                string
		projectID            string
		userID               string
		expectedJobTopic     string
		expectedProjectTopic string
	}{
		{
			name:                 "standard IDs",
			jobID:                "job_123",
			projectID:            "proj_456",
			userID:               "user_789",
			expectedJobTopic:     "job:job_123:user_789",
			expectedProjectTopic: "project:proj_456:user_789",
		},
		{
			name:                 "UUID format IDs",
			jobID:                "550e8400-e29b-41d4-a716-446655440000",
			projectID:            "550e8400-e29b-41d4-a716-446655440001",
			userID:               "550e8400-e29b-41d4-a716-446655440002",
			expectedJobTopic:     "job:550e8400-e29b-41d4-a716-446655440000:550e8400-e29b-41d4-a716-446655440002",
			expectedProjectTopic: "project:550e8400-e29b-41d4-a716-446655440001:550e8400-e29b-41d4-a716-446655440002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test job topic pattern
			jobTopic := fmt.Sprintf("job:%s:%s", tt.jobID, tt.userID)
			assert.Equal(t, tt.expectedJobTopic, jobTopic, "Job topic pattern should match specification")

			// Test project topic pattern
			projectTopic := fmt.Sprintf("project:%s:%s", tt.projectID, tt.userID)
			assert.Equal(t, tt.expectedProjectTopic, projectTopic, "Project topic pattern should match specification")
		})
	}
}
