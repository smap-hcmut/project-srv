package usecase

import (
	"context"
	"encoding/json"
	"smap-project/internal/webhook"
	"smap-project/pkg/log"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Task 3.1: Comprehensive Test Suite
// End-to-end tests from webhook callback to Redis publish

// concurrentMockRedisClient is a thread-safe mock for concurrent testing
type concurrentMockRedisClient struct {
	mu                sync.RWMutex
	data              map[string][]byte
	publishedMessages []publishedMessage
	publishCount      int
	publishError      error // Simulate Redis failures
}

type publishedMessage struct {
	Channel   string
	Message   []byte
	Timestamp time.Time
}

func newConcurrentMockRedisClient() *concurrentMockRedisClient {
	return &concurrentMockRedisClient{
		data:              make(map[string][]byte),
		publishedMessages: make([]publishedMessage, 0),
	}
}

func (m *concurrentMockRedisClient) Disconnect() error                { return nil }
func (m *concurrentMockRedisClient) Ping(ctx context.Context) error   { return nil }
func (m *concurrentMockRedisClient) IsReady(ctx context.Context) bool { return true }

func (m *concurrentMockRedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, context.DeadlineExceeded
}

func (m *concurrentMockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	}
	m.data[key] = bytes
	return nil
}

func (m *concurrentMockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *concurrentMockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok, nil
}

func (m *concurrentMockRedisClient) Expire(ctx context.Context, key string, expiration int) error {
	return nil
}
func (m *concurrentMockRedisClient) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return nil
}
func (m *concurrentMockRedisClient) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	return nil, nil
}
func (m *concurrentMockRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}
func (m *concurrentMockRedisClient) HIncrBy(ctx context.Context, key string, field string, incr int64) (int64, error) {
	return 0, nil
}
func (m *concurrentMockRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return nil, nil
}
func (m *concurrentMockRedisClient) Lock(ctx context.Context, key string, expiration int) (bool, error) {
	return true, nil
}
func (m *concurrentMockRedisClient) Unlock(ctx context.Context, key string) error { return nil }

func (m *concurrentMockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.publishError != nil {
		return m.publishError
	}

	var bytes []byte
	switch v := message.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		jsonBytes, _ := json.Marshal(v)
		bytes = jsonBytes
	}

	m.publishedMessages = append(m.publishedMessages, publishedMessage{
		Channel:   channel,
		Message:   bytes,
		Timestamp: time.Now(),
	})
	m.publishCount++
	return nil
}

func (m *concurrentMockRedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return nil
}
func (m *concurrentMockRedisClient) Pipeline() redis.Pipeliner   { return nil }
func (m *concurrentMockRedisClient) TxPipeline() redis.Pipeliner { return nil }

func (m *concurrentMockRedisClient) getPublishedMessages() []publishedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]publishedMessage, len(m.publishedMessages))
	copy(result, m.publishedMessages)
	return result
}

func (m *concurrentMockRedisClient) getPublishCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publishCount
}

func (m *concurrentMockRedisClient) setPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

func (m *concurrentMockRedisClient) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	m.publishedMessages = make([]publishedMessage, 0)
	m.publishCount = 0
	m.publishError = nil
}

// TestE2E_DryRunCallbackToRedisPublish tests the complete flow from webhook callback to Redis publish
func TestE2E_DryRunCallbackToRedisPublish(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	// Setup: Store job mapping
	jobID := "e2e_job_123"
	userID := "e2e_user_456"
	projectID := "e2e_project_789"

	err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
	require.NoError(t, err, "Failed to store job mapping")

	// Create a realistic callback request
	req := webhook.CallbackRequest{
		JobID:    jobID,
		Status:   "success",
		Platform: "tiktok",
		Payload: webhook.CallbackPayload{
			Content: []webhook.Content{
				createTestContent("content_1", "First test content", "keyword1"),
				createTestContent("content_2", "Second test content", "keyword1"),
			},
			Errors: []webhook.Error{},
		},
	}

	// Execute the handler
	err = uc.HandleDryRunCallback(ctx, req)
	require.NoError(t, err, "HandleDryRunCallback should succeed")

	// Verify Redis publish
	messages := mockRedis.getPublishedMessages()
	require.Len(t, messages, 1, "Should publish exactly one message")

	// Verify channel pattern
	expectedChannel := "job:e2e_job_123:e2e_user_456"
	assert.Equal(t, expectedChannel, messages[0].Channel, "Channel should match expected pattern")

	// Verify message structure
	var jobMessage webhook.JobMessage
	err = json.Unmarshal(messages[0].Message, &jobMessage)
	require.NoError(t, err, "Should unmarshal to JobMessage")

	assert.Equal(t, webhook.PlatformTikTok, jobMessage.Platform)
	assert.Equal(t, webhook.StatusCompleted, jobMessage.Status)
	assert.NotNil(t, jobMessage.Batch)
	assert.Len(t, jobMessage.Batch.ContentList, 2)
	assert.NotNil(t, jobMessage.Progress)
	assert.Equal(t, 100.0, jobMessage.Progress.Percentage)
}

// TestE2E_ProjectProgressToRedisPublish tests the complete flow for project progress
func TestE2E_ProjectProgressToRedisPublish(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	// Create progress callback request
	req := webhook.ProgressCallbackRequest{
		ProjectID: "e2e_proj_123",
		UserID:    "e2e_user_456",
		Status:    "PROCESSING",
		Total:     100,
		Done:      50,
		Errors:    0,
	}

	// Execute the handler
	err := uc.HandleProgressCallback(ctx, req)
	require.NoError(t, err, "HandleProgressCallback should succeed")

	// Verify Redis publish
	messages := mockRedis.getPublishedMessages()
	require.Len(t, messages, 1, "Should publish exactly one message")

	// Verify channel pattern
	expectedChannel := "project:e2e_proj_123:e2e_user_456"
	assert.Equal(t, expectedChannel, messages[0].Channel, "Channel should match expected pattern")

	// Verify message structure
	var projectMessage webhook.ProjectMessage
	err = json.Unmarshal(messages[0].Message, &projectMessage)
	require.NoError(t, err, "Should unmarshal to ProjectMessage")

	assert.Equal(t, webhook.StatusProcessing, projectMessage.Status)
	assert.NotNil(t, projectMessage.Progress)
	assert.Equal(t, 50, projectMessage.Progress.Current)
	assert.Equal(t, 100, projectMessage.Progress.Total)
	assert.Equal(t, 50.0, projectMessage.Progress.Percentage)
}

// createTestContent creates a test content item for testing
func createTestContent(id, text, keyword string) webhook.Content {
	return webhook.Content{
		Meta: webhook.ContentMeta{
			ID:            id,
			KeywordSource: keyword,
			PublishedAt:   time.Date(2024, 12, 10, 15, 30, 0, 0, time.UTC),
			Permalink:     "https://tiktok.com/" + id,
		},
		Content: webhook.ContentData{
			Text: text,
		},
		Author: webhook.ContentAuthor{
			ID:         "author_" + id,
			Username:   "user_" + id,
			Name:       "Test User " + id,
			Followers:  1000,
			IsVerified: false,
		},
		Interaction: webhook.ContentInteraction{
			Views:          10000,
			Likes:          500,
			CommentsCount:  50,
			Shares:         25,
			EngagementRate: 5.0,
			UpdatedAt:      time.Now(),
		},
	}
}

// TestErrorCases_MalformedInputs tests error handling for malformed inputs
func TestErrorCases_MalformedInputs(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	t.Run("dry run with missing job mapping", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.CallbackRequest{
			JobID:    "nonexistent_job",
			Status:   "success",
			Platform: "tiktok",
			Payload:  webhook.CallbackPayload{},
		}

		err := uc.HandleDryRunCallback(ctx, req)
		assert.Error(t, err, "Should fail when job mapping not found")
		assert.Equal(t, 0, mockRedis.getPublishCount(), "Should not publish on error")
	})

	t.Run("progress callback with empty project ID", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.ProgressCallbackRequest{
			ProjectID: "",
			UserID:    "user_123",
			Status:    "PROCESSING",
			Total:     100,
			Done:      50,
		}

		err := uc.HandleProgressCallback(ctx, req)
		assert.Error(t, err, "Should fail with empty project ID")
		assert.Equal(t, 0, mockRedis.getPublishCount(), "Should not publish on error")
	})

	t.Run("progress callback with empty user ID", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.ProgressCallbackRequest{
			ProjectID: "proj_123",
			UserID:    "",
			Status:    "PROCESSING",
			Total:     100,
			Done:      50,
		}

		err := uc.HandleProgressCallback(ctx, req)
		assert.Error(t, err, "Should fail with empty user ID")
		assert.Equal(t, 0, mockRedis.getPublishCount(), "Should not publish on error")
	})
}

// TestErrorCases_RedisFailures tests error handling for Redis failures
func TestErrorCases_RedisFailures(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	t.Run("Redis publish failure for dry run", func(t *testing.T) {
		mockRedis.reset()

		// Setup job mapping
		jobID := "redis_fail_job"
		userID := "user_123"
		projectID := "proj_123"
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		require.NoError(t, err)

		// Set Redis to fail on publish
		mockRedis.setPublishError(context.DeadlineExceeded)

		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "success",
			Platform: "tiktok",
			Payload:  webhook.CallbackPayload{},
		}

		err = uc.HandleDryRunCallback(ctx, req)
		assert.Error(t, err, "Should fail when Redis publish fails")
		assert.Contains(t, err.Error(), "failed to publish to Redis")
	})

	t.Run("Redis publish failure for progress", func(t *testing.T) {
		mockRedis.reset()
		mockRedis.setPublishError(context.DeadlineExceeded)

		req := webhook.ProgressCallbackRequest{
			ProjectID: "proj_123",
			UserID:    "user_123",
			Status:    "PROCESSING",
			Total:     100,
			Done:      50,
		}

		err := uc.HandleProgressCallback(ctx, req)
		assert.Error(t, err, "Should fail when Redis publish fails")
		assert.Contains(t, err.Error(), "failed to publish to Redis")
	})
}

// TestBoundaryConditions tests edge cases with boundary values
func TestBoundaryConditions(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	t.Run("empty content list", func(t *testing.T) {
		mockRedis.reset()

		jobID := "empty_content_job"
		userID := "user_123"
		projectID := "proj_123"
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		require.NoError(t, err)

		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "success",
			Platform: "tiktok",
			Payload: webhook.CallbackPayload{
				Content: []webhook.Content{}, // Empty content
				Errors:  []webhook.Error{},
			},
		}

		err = uc.HandleDryRunCallback(ctx, req)
		require.NoError(t, err)

		messages := mockRedis.getPublishedMessages()
		require.Len(t, messages, 1)

		var jobMessage webhook.JobMessage
		err = json.Unmarshal(messages[0].Message, &jobMessage)
		require.NoError(t, err)

		assert.Nil(t, jobMessage.Batch, "Batch should be nil for empty content")
		assert.NotNil(t, jobMessage.Progress, "Progress should still be present")
	})

	t.Run("zero progress values", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.ProgressCallbackRequest{
			ProjectID: "zero_progress_proj",
			UserID:    "user_123",
			Status:    "INITIALIZING",
			Total:     0,
			Done:      0,
			Errors:    0,
		}

		err := uc.HandleProgressCallback(ctx, req)
		require.NoError(t, err)

		messages := mockRedis.getPublishedMessages()
		require.Len(t, messages, 1)

		var projectMessage webhook.ProjectMessage
		err = json.Unmarshal(messages[0].Message, &projectMessage)
		require.NoError(t, err)

		assert.Equal(t, 0, projectMessage.Progress.Current)
		assert.Equal(t, 0, projectMessage.Progress.Total)
		assert.Equal(t, 0.0, projectMessage.Progress.Percentage, "Percentage should be 0 when total is 0")
	})

	t.Run("100% progress", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.ProgressCallbackRequest{
			ProjectID: "complete_proj",
			UserID:    "user_123",
			Status:    "DONE",
			Total:     100,
			Done:      100,
			Errors:    0,
		}

		err := uc.HandleProgressCallback(ctx, req)
		require.NoError(t, err)

		messages := mockRedis.getPublishedMessages()
		require.Len(t, messages, 1)

		var projectMessage webhook.ProjectMessage
		err = json.Unmarshal(messages[0].Message, &projectMessage)
		require.NoError(t, err)

		assert.Equal(t, webhook.StatusCompleted, projectMessage.Status)
		assert.Equal(t, 100.0, projectMessage.Progress.Percentage)
	})

	t.Run("large error count", func(t *testing.T) {
		mockRedis.reset()

		req := webhook.ProgressCallbackRequest{
			ProjectID: "error_proj",
			UserID:    "user_123",
			Status:    "FAILED",
			Total:     1000,
			Done:      500,
			Errors:    999,
		}

		err := uc.HandleProgressCallback(ctx, req)
		require.NoError(t, err)

		messages := mockRedis.getPublishedMessages()
		require.Len(t, messages, 1)

		var projectMessage webhook.ProjectMessage
		err = json.Unmarshal(messages[0].Message, &projectMessage)
		require.NoError(t, err)

		assert.Equal(t, webhook.StatusFailed, projectMessage.Status)
		assert.Len(t, projectMessage.Progress.Errors, 1)
		assert.Contains(t, projectMessage.Progress.Errors[0], "999 errors")
	})
}

// TestLoadTesting_HighMessageVolume tests handling of high message volume
func TestLoadTesting_HighMessageVolume(t *testing.T) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	const numMessages = 100

	t.Run("concurrent dry run callbacks", func(t *testing.T) {
		mockRedis.reset()

		// Setup job mappings
		for i := 0; i < numMessages; i++ {
			jobID := "load_job_" + string(rune('0'+i%10)) + string(rune('0'+i/10))
			userID := "user_" + string(rune('0'+i%10))
			projectID := "proj_" + string(rune('0'+i%10))
			err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, numMessages)

		for i := 0; i < numMessages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				jobID := "load_job_" + string(rune('0'+idx%10)) + string(rune('0'+idx/10))
				req := webhook.CallbackRequest{
					JobID:    jobID,
					Status:   "success",
					Platform: "tiktok",
					Payload: webhook.CallbackPayload{
						Content: []webhook.Content{
							createTestContent("content_"+jobID, "Test content", "keyword"),
						},
					},
				}

				if err := uc.HandleDryRunCallback(ctx, req); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errCount int
		for err := range errors {
			t.Logf("Error during load test: %v", err)
			errCount++
		}

		assert.Equal(t, 0, errCount, "Should have no errors during load test")
		assert.Equal(t, numMessages, mockRedis.getPublishCount(), "Should publish all messages")
	})

	t.Run("concurrent progress callbacks", func(t *testing.T) {
		mockRedis.reset()

		var wg sync.WaitGroup
		errors := make(chan error, numMessages)

		for i := 0; i < numMessages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				req := webhook.ProgressCallbackRequest{
					ProjectID: "load_proj_" + string(rune('0'+idx%10)) + string(rune('0'+idx/10)),
					UserID:    "user_" + string(rune('0'+idx%10)),
					Status:    "PROCESSING",
					Total:     100,
					Done:      int64(idx),
					Errors:    0,
				}

				if err := uc.HandleProgressCallback(ctx, req); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errCount int
		for err := range errors {
			t.Logf("Error during load test: %v", err)
			errCount++
		}

		assert.Equal(t, 0, errCount, "Should have no errors during load test")
		assert.Equal(t, numMessages, mockRedis.getPublishCount(), "Should publish all messages")
	})
}

// TestPerformanceBenchmark benchmarks message transformation and publishing
func BenchmarkTransformDryRunCallback(b *testing.B) {
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	req := webhook.CallbackRequest{
		JobID:    "bench_job",
		Status:   "success",
		Platform: "tiktok",
		Payload: webhook.CallbackPayload{
			Content: []webhook.Content{
				createTestContent("content_1", "Test content 1", "keyword"),
				createTestContent("content_2", "Test content 2", "keyword"),
				createTestContent("content_3", "Test content 3", "keyword"),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.TransformDryRunCallback(req)
	}
}

func BenchmarkTransformProjectCallback(b *testing.B) {
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	req := webhook.ProgressCallbackRequest{
		ProjectID: "bench_proj",
		UserID:    "bench_user",
		Status:    "PROCESSING",
		Total:     1000,
		Done:      500,
		Errors:    10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.TransformProjectCallback(req)
	}
}

func BenchmarkHandleDryRunCallback(b *testing.B) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	// Setup job mapping
	_ = uc.StoreJobMapping(ctx, "bench_job", "bench_user", "bench_proj")

	req := webhook.CallbackRequest{
		JobID:    "bench_job",
		Status:   "success",
		Platform: "tiktok",
		Payload: webhook.CallbackPayload{
			Content: []webhook.Content{
				createTestContent("content_1", "Test content", "keyword"),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.HandleDryRunCallback(ctx, req)
	}
}

func BenchmarkHandleProgressCallback(b *testing.B) {
	ctx := context.Background()
	mockRedis := newConcurrentMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	req := webhook.ProgressCallbackRequest{
		ProjectID: "bench_proj",
		UserID:    "bench_user",
		Status:    "PROCESSING",
		Total:     1000,
		Done:      500,
		Errors:    0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.HandleProgressCallback(ctx, req)
	}
}
