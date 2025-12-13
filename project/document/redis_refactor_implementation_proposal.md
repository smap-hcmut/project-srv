# Redis Pub/Sub Refactor Implementation Proposal

**Ngày tạo**: 2025-12-14  
**Tác giả**: Development Team  
**Trạng thái**: Ready for Implementation  
**Phiên bản**: 1.0 - Final  
**Scope**: Project Service Only

---

## Executive Summary

Proposal này mô tả việc refactor Redis pub/sub logic trong **Project Service** để chuyển từ single topic pattern sang multiple topic-specific patterns. Thay đổi này sẽ cải thiện message routing và structure consistency mà không ảnh hưởng đến các services khác.

### **Key Changes**

- **Dry Run**: `user_noti:{userID}` → `job:{jobID}:{userID}`
- **Project Progress**: `user_noti:{userID}` → `project:{projectID}:{userID}`
- **Message Structure**: Generic format → Topic-specific formats
- **Scope**: Chỉ thay đổi Project Service

---

## Current State Analysis

### **Current Redis Pub/Sub Pattern**

```go
// Single topic cho tất cả notifications
channel := fmt.Sprintf("user_noti:%s", userID)

// Generic message structure với type discrimination
message := map[string]interface{}{
    "type": "dryrun_result", // or "project_progress", "project_completed"
    "payload": map[string]interface{}{
        // Mixed payload structure
    },
}
```

### **Issues với Current Approach**

1. **Single topic overload** - Tất cả message types qua 1 channel
2. **Type discrimination complexity** - WebSocket service phải parse `type` field
3. **Generic structure** - Không type-safe, khó maintain
4. **Mixed payload formats** - Inconsistent data structures

---

## Proposed Solution

### **Enums and Constants**

```go
// Platform enum for supported social media platforms
type Platform string

const (
    PlatformTikTok    Platform = "TIKTOK"
    PlatformYouTube   Platform = "YOUTUBE"
    PlatformInstagram Platform = "INSTAGRAM"
)

// Status enum for job and project status
type Status string

const (
    StatusProcessing Status = "PROCESSING"
    StatusCompleted  Status = "COMPLETED"
    StatusFailed     Status = "FAILED"
    StatusPaused     Status = "PAUSED"
)

// MediaType enum for content media types
type MediaType string

const (
    MediaTypeVideo MediaType = "video"
    MediaTypeImage MediaType = "image"
    MediaTypeAudio MediaType = "audio"
)
```

### **New Topic-Specific Pattern**

#### **1. Dry Run Jobs**

```go
// Topic pattern
channel := fmt.Sprintf("job:%s:%s", jobID, userID)

// Specific message structure
type JobMessage struct {
    Platform Platform   `json:"platform"`    // PlatformTikTok, PlatformYouTube, etc.
    Status   Status     `json:"status"`      // StatusProcessing, StatusCompleted, etc.
    Batch    *BatchData `json:"batch,omitempty"`
    Progress *Progress  `json:"progress,omitempty"`
}
```

#### **2. Project Progress**

```go
// Topic pattern
channel := fmt.Sprintf("project:%s:%s", projectID, userID)

// Specific message structure
type ProjectMessage struct {
    Status   Status    `json:"status"`             // StatusProcessing, StatusCompleted, etc.
    Progress *Progress `json:"progress,omitempty"` // Overall progress (combined crawl + process)
}
```

### **Shared Data Structures**

```go
type Progress struct {
    Current    int      `json:"current"`     // Current completed items
    Total      int      `json:"total"`       // Total items to process
    Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
    ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
    Errors     []string `json:"errors"`      // Array of error messages
}

type BatchData struct {
    Keyword     string        `json:"keyword"`      // Search keyword
    ContentList []ContentItem `json:"content_list"` // Crawled content items
    CrawledAt   string        `json:"crawled_at"`   // ISO timestamp
}

type ContentItem struct {
    ID          string      `json:"id"`           // Content unique ID
    Text        string      `json:"text"`         // Content text/caption
    Author      AuthorInfo  `json:"author"`       // Author information
    Metrics     MetricsInfo `json:"metrics"`      // Engagement statistics
    Media       *MediaInfo  `json:"media,omitempty"`        // Media information
    PublishedAt string      `json:"published_at"` // ISO timestamp
    Permalink   string      `json:"permalink"`    // Direct link to content
}

type AuthorInfo struct {
    ID         string `json:"id"`          // Author unique ID
    Username   string `json:"username"`    // Author username/handle
    Name       string `json:"name"`        // Author display name
    Followers  int    `json:"followers"`   // Follower count
    IsVerified bool   `json:"is_verified"` // Verification status
    AvatarURL  string `json:"avatar_url"`  // Profile picture URL
}

type MetricsInfo struct {
    Views    int     `json:"views"`    // View count
    Likes    int     `json:"likes"`    // Like count
    Comments int     `json:"comments"` // Comment count
    Shares   int     `json:"shares"`   // Share count
    Rate     float64 `json:"rate"`     // Engagement rate percentage
}

type MediaInfo struct {
    Type      MediaType `json:"type"`      // MediaTypeVideo, MediaTypeImage, MediaTypeAudio
    Duration  int       `json:"duration,omitempty"`  // Duration in seconds
    Thumbnail string    `json:"thumbnail"` // Thumbnail/preview URL
    URL       string    `json:"url"`       // Media file URL
}
```

---

## Implementation Details

### **Phase 1: Add New Message Structures**

#### **1.1 Create New Types**

```go
// File: project/internal/webhook/redis_types.go
package webhook

import "time"

// JobMessage represents dry run job notifications
type JobMessage struct {
    Platform Platform   `json:"platform"`
    Status   Status     `json:"status"`
    Batch    *BatchData `json:"batch,omitempty"`
    Progress *Progress  `json:"progress,omitempty"`
}

// ProjectMessage represents project progress notifications
type ProjectMessage struct {
    Status   Status    `json:"status"`
    Progress *Progress `json:"progress,omitempty"` // Overall progress
}

// ... (other shared structs as defined above)
```

#### **1.2 Add Transformation Functions**

```go
// File: project/internal/webhook/transformers.go
package webhook

func TransformDryRunCallback(req CallbackRequest) JobMessage {
    // Transform content to batch format
    var batch *BatchData
    if len(req.Payload.Content) > 0 {
        batch = &BatchData{
            Keyword:     extractKeywordFromContent(req.Payload.Content),
            ContentList: transformContentItems(req.Payload.Content),
            CrawledAt:   time.Now().Format(time.RFC3339),
        }
    }

    // Transform errors to progress format
    var progress *Progress
    if req.Status == "success" || len(req.Payload.Errors) > 0 {
        progress = &Progress{
            Current:    1, // Single batch for dry run
            Total:      1,
            Percentage: 100.0,
            ETA:        0.0,
            Errors:     transformErrorsToStrings(req.Payload.Errors),
        }
    }

    return JobMessage{
        Platform: mapPlatform(req.Platform),
        Status:   mapDryRunStatus(req.Status),
        Batch:    batch,
        Progress: progress,
    }
}

func TransformProjectCallback(req ProgressCallbackRequest) ProjectMessage {
    progress := &Progress{
        Current:    int(req.Done),
        Total:      int(req.Total),
        Percentage: calculatePercentage(req.Done, req.Total),
        ETA:        0.0, // TODO: Calculate ETA if needed
        Errors:     transformErrorCount(req.Errors),
    }

    return ProjectMessage{
        Status:   mapProjectStatus(req.Status),
        Progress: progress,
    }
}

// Helper functions
func transformContentItems(contents []Content) []ContentItem {
    items := make([]ContentItem, len(contents))
    for i, content := range contents {
        items[i] = ContentItem{
            ID:   content.Meta.ID,
            Text: content.Content.Text,
            Author: AuthorInfo{
                ID:         content.Author.ID,
                Username:   content.Author.Username,
                Name:       content.Author.Name,
                Followers:  content.Author.Followers,
                IsVerified: content.Author.IsVerified,
                AvatarURL:  getStringValue(content.Author.AvatarURL),
            },
            Metrics: MetricsInfo{
                Views:    content.Interaction.Views,
                Likes:    content.Interaction.Likes,
                Comments: content.Interaction.CommentsCount,
                Shares:   content.Interaction.Shares,
                Rate:     content.Interaction.EngagementRate,
            },
            Media:       transformMedia(content.Content.Media),
            PublishedAt: content.Meta.PublishedAt.Format(time.RFC3339),
            Permalink:   content.Meta.Permalink,
        }
    }
    return items
}

func transformMedia(media *ContentMedia) *MediaInfo {
    if media == nil {
        return nil
    }
    return &MediaInfo{
        Type:      mapMediaType(media.Type),
        Duration:  0, // Map from content.Duration if available
        Thumbnail: "", // Map from appropriate field
        URL:       media.VideoPath,
    }
}

func transformErrorsToStrings(errors []Error) []string {
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

func transformErrorCount(errorCount int64) []string {
    if errorCount == 0 {
        return []string{}
    }
    return []string{
        fmt.Sprintf("Processing encountered %d errors", errorCount),
    }
}

func mapPlatform(platform string) Platform {
    switch strings.ToUpper(platform) {
    case "TIKTOK":
        return PlatformTikTok
    case "YOUTUBE":
        return PlatformYouTube
    case "INSTAGRAM":
        return PlatformInstagram
    default:
        return PlatformTikTok // Default fallback
    }
}

func mapDryRunStatus(status string) Status {
    switch status {
    case "success":
        return StatusCompleted
    case "failed":
        return StatusFailed
    default:
        return StatusProcessing
    }
}

func mapProjectStatus(status string) Status {
    // Map from collector status to standard status
    switch strings.ToUpper(status) {
    case "DONE":
        return StatusCompleted
    case "FAILED":
        return StatusFailed
    case "INITIALIZING", "CRAWLING", "PROCESSING":
        return StatusProcessing
    default:
        return StatusProcessing
    }
}

func mapMediaType(mediaType string) MediaType {
    switch strings.ToLower(mediaType) {
    case "video":
        return MediaTypeVideo
    case "image":
        return MediaTypeImage
    case "audio":
        return MediaTypeAudio
    default:
        return MediaTypeVideo // Default fallback
    }
}

func calculatePercentage(done, total int64) float64 {
    if total == 0 {
        return 0.0
    }
    return float64(done) / float64(total) * 100.0
}

func extractKeywordFromContent(contents []Content) string {
    if len(contents) > 0 {
        return contents[0].Meta.KeywordSource
    }
    return ""
}

func getStringValue(ptr *string) string {
    if ptr == nil {
        return ""
    }
    return *ptr
}
```

### **Phase 2: Update Webhook Handlers**

#### **2.1 Update Dry Run Handler**

```go
// File: project/internal/webhook/usecase/webhook.go

func (uc *usecase) HandleDryRunCallback(ctx context.Context, req webhook.CallbackRequest) error {
    // Lookup userID from Redis job mapping (unchanged)
    userID, projectID, err := uc.getJobMapping(ctx, req.JobID)
    if err != nil {
        uc.l.Errorf(ctx, "Failed to get job mapping: job_id=%s, platform=%s, status=%s, error=%v",
            req.JobID, req.Platform, req.Status, err)
        return err
    }

    // NEW: Use job-specific topic pattern
    channel := fmt.Sprintf("job:%s:%s", req.JobID, userID)

    // NEW: Transform to job message structure
    message := webhook.TransformDryRunCallback(req)

    // Marshal and publish
    body, err := json.Marshal(message)
    if err != nil {
        uc.l.Errorf(ctx, "internal.webhook.usecase.HandleDryRunCallback.Marshal: %v", err)
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
        uc.l.Errorf(ctx, "internal.webhook.usecase.HandleDryRunCallback.Publish: %v", err)
        return fmt.Errorf("failed to publish to Redis: %w", err)
    }

    uc.l.Infof(ctx, "Published dry-run result to Redis: channel=%s, job_id=%s, platform=%s, status=%s",
        channel, req.JobID, req.Platform, req.Status)

    return nil
}
```

#### **2.2 Update Project Handler**

```go
func (uc *usecase) HandleProgressCallback(ctx context.Context, req webhook.ProgressCallbackRequest) error {
    // NEW: Use project-specific topic pattern
    channel := fmt.Sprintf("project:%s:%s", req.ProjectID, req.UserID)

    // NEW: Transform to project message structure
    message := webhook.TransformProjectCallback(req)

    // Marshal and publish
    body, err := json.Marshal(message)
    if err != nil {
        uc.l.Errorf(ctx, "internal.webhook.usecase.HandleProgressCallback.Marshal: %v", err)
        return err
    }

    if err := uc.redisClient.Publish(ctx, channel, body); err != nil {
        uc.l.Errorf(ctx, "internal.webhook.usecase.HandleProgressCallback.Publish: %v", err)
        return err
    }

    uc.l.Infof(ctx, "Published project progress to Redis: channel=%s, project_id=%s, status=%s",
        channel, req.ProjectID, req.Status)

    return nil
}
```

### **Phase 3: Testing Strategy**

#### **3.1 Unit Tests**

```go
// File: project/internal/webhook/transformers_test.go
func TestTransformDryRunCallback(t *testing.T) {
    // Test successful dry run transformation
    req := webhook.CallbackRequest{
        JobID:    "job_123",
        Status:   "success",
        Platform: "tiktok",
        Payload: webhook.CallbackPayload{
            Content: []webhook.Content{
                {
                    Meta: webhook.ContentMeta{
                        ID: "content_456",
                        KeywordSource: "test_keyword",
                        // ... other fields
                    },
                    // ... other content fields
                },
            },
            Errors: []webhook.Error{},
        },
    }

    result := webhook.TransformDryRunCallback(req)

    assert.Equal(t, PlatformTikTok, result.Platform)
    assert.Equal(t, StatusCompleted, result.Status)
    assert.NotNil(t, result.Batch)
    assert.Equal(t, "test_keyword", result.Batch.Keyword)
    assert.Len(t, result.Batch.ContentList, 1)
}

func TestTransformProjectCallback(t *testing.T) {
    req := webhook.ProgressCallbackRequest{
        ProjectID: "proj_123",
        UserID:    "user_456",
        Status:    "PROCESSING",
        Total:     100,
        Done:      75,
        Errors:    2,
    }

    result := webhook.TransformProjectCallback(req)

    assert.Equal(t, StatusProcessing, result.Status)
    assert.NotNil(t, result.Progress)
    assert.Equal(t, 75, result.Progress.Current)
    assert.Equal(t, 100, result.Progress.Total)
    assert.Equal(t, 75.0, result.Progress.Percentage)
}
```

#### **3.2 Integration Tests**

```go
// File: project/internal/webhook/usecase/webhook_integration_test.go
func TestHandleDryRunCallbackNewFormat(t *testing.T) {
    // Setup mock Redis client
    mockRedis := &mockRedisClient{data: make(map[string][]byte)}

    // Store job mapping
    jobID := "test_job_123"
    userID := "test_user_456"

    // Test the new topic pattern and message structure
    uc := &usecase{
        redisClient: mockRedis,
        l:          logger,
    }

    req := webhook.CallbackRequest{
        JobID:    jobID,
        Status:   "success",
        Platform: "tiktok",
        // ... payload
    }

    err := uc.HandleDryRunCallback(context.Background(), req)
    assert.NoError(t, err)

    // Verify new topic pattern
    expectedChannel := fmt.Sprintf("job:%s:%s", jobID, userID)
    assert.Contains(t, mockRedis.publishedChannels, expectedChannel)

    // Verify message structure
    var message webhook.JobMessage
    err = json.Unmarshal(mockRedis.publishedMessages[expectedChannel], &message)
    assert.NoError(t, err)
    assert.Equal(t, PlatformTikTok, message.Platform)
    assert.Equal(t, StatusCompleted, message.Status)
}
```

---

## ⚠️ CRITICAL: Completion Trigger Logic

### **Vấn đề quan trọng cần đặc biệt chú ý**

#### **Project Completion - Đơn giản**

- **Trigger**: Khi `status = StatusCompleted` hoặc `status = StatusFailed`
- **Logic**: Collector service gửi **1 callback duy nhất** khi project hoàn thành
- **Client handling**: Đơn giản - nhận 1 message với status COMPLETED là xong

```
Timeline Project:
T+0s:    Project started
T+10s:   Progress update (status: PROCESSING, 25%)
T+20s:   Progress update (status: PROCESSING, 50%)
T+30s:   Progress update (status: PROCESSING, 75%)
T+40s:   ✅ Final update (status: COMPLETED, 100%) → JOB DONE
```

#### **Job (Dry Run) Completion - Phức tạp**

- **Vấn đề**: Mỗi platform gửi callback **RIÊNG BIỆT** với `status = StatusCompleted`
- **Thực tế**: 1 job có thể có **NHIỀU messages** với status COMPLETED (mỗi platform 1 message)
- **Client KHÔNG THỂ** chỉ dựa vào `status = COMPLETED` để biết job đã hoàn thành

```
Timeline Dry Run Job (2 platforms: TikTok + YouTube):
T+0s:    Job started, dispatched to 2 platforms
T+15s:   TikTok callback (platform: TIKTOK, status: COMPLETED) → ❌ JOB CHƯA XONG!
T+30s:   YouTube callback (platform: YOUTUBE, status: COMPLETED) → ✅ JOB XONG!
```

---

## Migration Plan

### **Step 1: Preparation (Day 1)**

1. ✅ Add new message structures to `webhook/redis_types.go`
2. ✅ Implement transformation functions in `webhook/transformers.go`
3. ✅ Write comprehensive unit tests
4. ✅ Code review and approval

### **Step 2: Implementation (Day 2)**

1. ✅ Update `HandleDryRunCallback()` with new logic
2. ✅ Update `HandleProgressCallback()` with new logic
3. ✅ Run integration tests
4. ✅ Local testing with mock Redis

### **Step 3: Deployment (Day 3)**

1. ✅ Deploy to staging environment
2. ✅ Test with WebSocket service integration
3. ✅ Monitor Redis pub/sub patterns
4. ✅ Deploy to production
5. ✅ Monitor for any issues

### **Step 4: Cleanup (Day 4)**

1. ✅ Remove old message format code (if any)
2. ✅ Update documentation
3. ✅ Performance monitoring

---

## Benefits

### **1. Improved Message Routing**

- **Topic-specific subscriptions** - WebSocket service can subscribe to specific patterns
- **Reduced message processing** - No need to parse `type` field
- **Better scalability** - Can scale different topic types independently

### **2. Type Safety**

- **Structured messages** - Proper Go structs instead of `map[string]interface{}`
- **Compile-time validation** - Catch errors during development
- **Better IDE support** - Autocomplete and type checking

### **3. Maintainability**

- **Clear separation** - Each message type has its own structure
- **Easier testing** - Type-safe test assertions
- **Better documentation** - Self-documenting struct fields

### **4. Performance**

- **Smaller messages** - No redundant `type` field
- **Faster parsing** - Direct struct unmarshaling
- **Reduced memory** - No generic interface{} allocations

---

## Risk Mitigation

### **1. Backward Compatibility**

- **WebSocket service** sẽ handle cả old và new formats during transition
- **Gradual rollout** - Deploy và test từng component một cách cẩn thận
- **Rollback plan** - Có thể revert về old format nếu cần

### **2. Data Validation**

- **Input validation** - Validate all required fields before publishing
- **Error handling** - Graceful degradation nếu transformation fails
- **Monitoring** - Log all transformation errors for debugging

### **3. Testing Coverage**

- **Unit tests** - 100% coverage cho transformation functions
- **Integration tests** - End-to-end testing với real Redis
- **Load testing** - Verify performance under high message volume

---

## Success Metrics

### **1. Technical Metrics**

- ✅ **Zero message loss** during transition
- ✅ **Improved message processing time** (target: 20% faster)
- ✅ **Reduced memory usage** (target: 15% less)
- ✅ **100% test coverage** for new code

### **2. Operational Metrics**

- ✅ **Zero downtime** deployment
- ✅ **No increase in error rates**
- ✅ **Successful WebSocket message delivery**
- ✅ **Positive developer feedback** on new structure

---

## Conclusion

Proposal này cung cấp một solution hoàn chỉnh để refactor Redis pub/sub pattern trong Project Service. Với scope giới hạn chỉ trong một service, risk được minimize trong khi benefits về performance, maintainability và type safety được maximize.

**Timeline**: 3-4 ngày implementation  
**Risk Level**: Low-Medium  
**Impact**: High positive impact on system architecture  
**Recommendation**: ✅ **Proceed with implementation**
