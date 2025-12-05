package usecase

import (
	"context"
	"encoding/json"
	"smap-project/internal/webhook"
	"smap-project/pkg/log"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// mockRedisClient implements redis.Client interface for testing
type mockRedisClient struct {
	data map[string][]byte
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string][]byte),
	}
}

// Connection methods
func (m *mockRedisClient) Disconnect() error {
	return nil
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockRedisClient) IsReady(ctx context.Context) bool {
	return true
}

// Basic Key-Value Operations
func (m *mockRedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, context.DeadlineExceeded
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration int) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	m.data[key] = bytes
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *mockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration int) error {
	return nil
}

// Hash Operations
func (m *mockRedisClient) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return nil
}

func (m *mockRedisClient) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	return nil, nil
}

func (m *mockRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}

func (m *mockRedisClient) HIncrBy(ctx context.Context, key string, field string, incr int64) (int64, error) {
	return 0, nil
}

// Multiple Key Operations
func (m *mockRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return nil, nil
}

// Distributed Lock
func (m *mockRedisClient) Lock(ctx context.Context, key string, expiration int) (bool, error) {
	return true, nil
}

func (m *mockRedisClient) Unlock(ctx context.Context, key string) error {
	return nil
}

// Pub/Sub Operations
func (m *mockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return nil
}

func (m *mockRedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return nil
}

// Pipeline Operations
func (m *mockRedisClient) Pipeline() redis.Pipeliner {
	return nil
}

func (m *mockRedisClient) TxPipeline() redis.Pipeliner {
	return nil
}

// TestCheckpoint_Phase1And2 verifies the checkpoint requirements:
// 1. Job mappings are stored when jobs are created
// 2. Callbacks work with both lookup and fallback
// 3. Old collector callbacks (with UserID) still work
func TestCheckpoint_Phase1And2(t *testing.T) {
	ctx := context.Background()
	mockRedis := newMockRedisClient()
	logger := log.NewNopLogger()

	uc := &usecase{
		l:           logger,
		redisClient: mockRedis,
	}

	t.Run("1. Verify job mappings are stored when jobs are created", func(t *testing.T) {
		jobID := "test-job-123"
		userID := "user-456"
		projectID := "project-789"

		// Store job mapping (simulating job creation)
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		if err != nil {
			t.Fatalf("Failed to store job mapping: %v", err)
		}

		// Verify the mapping was stored
		retrievedUserID, retrievedProjectID, err := uc.getJobMapping(ctx, jobID)
		if err != nil {
			t.Fatalf("Failed to retrieve job mapping: %v", err)
		}

		if retrievedUserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, retrievedUserID)
		}

		if retrievedProjectID != projectID {
			t.Errorf("Expected projectID %s, got %s", projectID, retrievedProjectID)
		}

		t.Logf("✓ Job mapping stored and retrieved successfully: jobID=%s, userID=%s, projectID=%s", jobID, userID, projectID)
	})

	t.Run("2. Verify callbacks work with lookup (no UserID in request)", func(t *testing.T) {
		jobID := "test-job-lookup"
		userID := "user-lookup"
		projectID := "project-lookup"

		// Store job mapping first
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		if err != nil {
			t.Fatalf("Failed to store job mapping: %v", err)
		}

		// Create callback request WITHOUT UserID (new format)
		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "success",
			Platform: "youtube",
			Payload:  webhook.CallbackPayload{},
			UserID:   "", // Empty UserID - should use lookup
		}

		// Process callback
		err = uc.HandleDryRunCallback(ctx, req)
		if err != nil {
			t.Fatalf("Failed to handle callback with lookup: %v", err)
		}

		t.Logf("✓ Callback processed successfully using lookup: jobID=%s", jobID)
	})

	t.Run("3. Verify callbacks fail when job mapping not found (no fallback)", func(t *testing.T) {
		jobID := "test-job-not-found"

		// Create callback request WITHOUT stored job mapping
		// Current implementation has NO fallback - it should fail
		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "success",
			Platform: "tiktok",
			Payload:  webhook.CallbackPayload{},
			UserID:   "user-fallback", // This is ignored - no fallback in current impl
		}

		// Process callback - should fail because no job mapping exists
		err := uc.HandleDryRunCallback(ctx, req)
		if err == nil {
			t.Fatalf("Expected error when job mapping not found, got nil")
		}

		t.Logf("✓ Callback correctly failed when job mapping not found: jobID=%s, error=%v", jobID, err)
	})

	t.Run("4. Verify old collector callbacks (with UserID) still work", func(t *testing.T) {
		jobID := "test-job-old-format"
		userID := "user-old-format"
		projectID := "project-old-format"

		// Store job mapping (simulating a job that was created)
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		if err != nil {
			t.Fatalf("Failed to store job mapping: %v", err)
		}

		// Create callback request WITH UserID (old collector format)
		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "success",
			Platform: "youtube",
			Payload:  webhook.CallbackPayload{},
			UserID:   userID, // Old format includes UserID
		}

		// Process callback - should prefer lookup but fallback works too
		err = uc.HandleDryRunCallback(ctx, req)
		if err != nil {
			t.Fatalf("Failed to handle old format callback: %v", err)
		}

		t.Logf("✓ Old collector callback processed successfully: jobID=%s, userID=%s", jobID, userID)
	})

	t.Run("5. Verify missing job mapping returns error", func(t *testing.T) {
		jobID := "test-job-missing"

		// Create callback request WITHOUT stored mapping
		req := webhook.CallbackRequest{
			JobID:    jobID,
			Status:   "failed",
			Platform: "youtube",
			Payload:  webhook.CallbackPayload{},
			UserID:   "", // No UserID provided
		}

		// Process callback - should return error because job mapping not found
		// Current implementation does NOT acknowledge callback without valid mapping
		err := uc.HandleDryRunCallback(ctx, req)
		if err == nil {
			t.Errorf("Expected error for missing job mapping, got nil")
		}

		t.Logf("✓ Missing job mapping correctly returns error: %v", err)
	})

	t.Run("6. Verify dry-run jobs with empty projectID work", func(t *testing.T) {
		jobID := "test-job-dryrun"
		userID := "user-dryrun"
		projectID := "" // Empty for dry-run jobs

		// Store job mapping with empty projectID (simulating dry-run job creation)
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		if err != nil {
			t.Fatalf("Failed to store job mapping with empty projectID: %v", err)
		}

		// Verify the mapping was stored
		retrievedUserID, retrievedProjectID, err := uc.getJobMapping(ctx, jobID)
		if err != nil {
			t.Fatalf("Failed to retrieve job mapping: %v", err)
		}

		if retrievedUserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, retrievedUserID)
		}

		if retrievedProjectID != projectID {
			t.Errorf("Expected empty projectID, got %s", retrievedProjectID)
		}

		t.Logf("✓ Dry-run job mapping (empty projectID) stored and retrieved successfully")
	})

	t.Run("7. Verify job mapping data includes timestamp", func(t *testing.T) {
		jobID := "test-job-timestamp"
		userID := "user-timestamp"
		projectID := "project-timestamp"

		// Store job mapping
		err := uc.StoreJobMapping(ctx, jobID, userID, projectID)
		if err != nil {
			t.Fatalf("Failed to store job mapping: %v", err)
		}

		// Retrieve raw data from Redis to check timestamp
		key := "job:mapping:" + jobID
		jsonData, err := mockRedis.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get raw data from Redis: %v", err)
		}

		var data webhook.JobMappingData
		if err := json.Unmarshal(jsonData, &data); err != nil {
			t.Fatalf("Failed to unmarshal job mapping data: %v", err)
		}

		if data.CreatedAt.IsZero() {
			t.Error("Expected non-zero CreatedAt timestamp")
		}

		if time.Since(data.CreatedAt) > 5*time.Second {
			t.Errorf("CreatedAt timestamp seems too old: %v", data.CreatedAt)
		}

		t.Logf("✓ Job mapping includes timestamp: createdAt=%v", data.CreatedAt)
	})
}
