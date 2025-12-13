# Redis Refactor Impact Analysis

**Ngày phân tích**: 2025-12-14  
**Tác giả**: System Analysis  
**Mục đích**: Phân tích impact của việc refactor Redis pub/sub từ single topic sang multiple topics

---

## Tóm Tắt Thay Đổi

### **Trước đây (Current)**

- **Topic**: `user_noti:{userID}` (single topic)
- **Message**: Có field `type` để phân biệt loại message
- **Structure**: Generic với `type` + `payload`

### **Sau khi refactor (New)**

- **Dry Run Topic**: `job:{jobID}:{userID}`
- **Project Topic**: `project:{projectID}:{userID}`
- **Message**: Specific structure cho từng topic, không cần field `type`

---

## 1. Impact lên Dry Run Job Structure

### 1.1 Thay đổi từ Crawler Message

#### **❌ CRITICAL IMPACT - CẦN THAY ĐỔI**

**Hiện tại**: Crawler gửi callback với structure:

```go
type CallbackRequest struct {
    JobID    string          `json:"job_id"`
    Status   string          `json:"status"`
    Platform string          `json:"platform"`
    Payload  CallbackPayload `json:"payload"`
}
```

**Vấn đề**: Crawler callback **KHÔNG chứa UserID**, nhưng new topic pattern cần `job:{jobID}:{userID}`

#### **Root Cause Analysis**

```go
// Current flow:
1. Crawler gửi callback → Project Service
2. Project Service lookup userID từ Redis job mapping
3. Project Service publish → `user_noti:{userID}`

// New flow requirement:
1. Crawler gửi callback → Project Service
2. Project Service cần publish → `job:{jobID}:{userID}`
3. ❌ Nhưng vẫn cần lookup userID từ Redis job mapping
```

#### **Kết luận**:

- **KHÔNG CẦN thay đổi** crawler message structure
- **Vẫn cần** Redis job mapping lookup như hiện tại
- **Chỉ thay đổi** topic pattern khi publish

### 1.2 Thay đổi Redis Message Structure cho Dry Run

#### **🔄 MODERATE IMPACT - CẦN REFACTOR**

**Hiện tại**:

```go
// Published to: user_noti:{userID}
message := map[string]interface{}{
    "type": "dryrun_result",
    "payload": map[string]interface{}{
        "job_id":   req.JobID,
        "platform": req.Platform,
        "status":   req.Status,
        "content":  req.Payload.Content,
        "errors":   req.Payload.Errors,
    },
}
```

**Mới theo specification**:

```go
// Published to: job:{jobID}:{userID}
type JobMessage struct {
    Platform Platform   `json:"platform"`    // PlatformTikTok, PlatformYouTube, etc.
    Status   Status     `json:"status"`      // StatusProcessing, StatusCompleted, etc.
    Batch    *BatchData `json:"batch,omitempty"`
    Progress *Progress  `json:"progress,omitempty"`
}
```

#### **Mapping Analysis**:

| Current Field | New Field               | Mapping Status         |
| ------------- | ----------------------- | ---------------------- |
| `job_id`      | ❌ Removed              | Moved to topic pattern |
| `platform`    | ✅ `platform`           | Direct mapping         |
| `status`      | ✅ `status`             | Direct mapping         |
| `content`     | 🔄 `batch.content_list` | Needs transformation   |
| `errors`      | 🔄 `progress.errors`    | Needs transformation   |

#### **Required Changes**:

```go
// New dry run message structure
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
```

---

## 2. Impact lên Project Structure

### 2.1 Thay đổi từ Collector Message

#### **✅ NO IMPACT - KHÔNG CẦN THAY ĐỔI**

**Lý do**: Project progress callbacks đã chứa đầy đủ thông tin:

```go
type ProgressCallbackRequest struct {
    ProjectID string `json:"project_id"` // ✅ Có sẵn
    UserID    string `json:"user_id"`    // ✅ Có sẵn
    Status    string `json:"status"`
    Total     int64  `json:"total"`
    Done      int64  `json:"done"`
    Errors    int64  `json:"errors"`
}
```

**Topic pattern**: `project:{projectID}:{userID}` - ✅ Có đủ data

### 2.2 Thay đổi Redis Message Structure cho Project

#### **🔄 MODERATE IMPACT - CẦN REFACTOR**

**Hiện tại**:

```go
// Published to: user_noti:{userID}
message := map[string]interface{}{
    "type": "project_progress", // or "project_completed"
    "payload": map[string]interface{}{
        "project_id":       req.ProjectID,
        "status":           req.Status,
        "total":            req.Total,
        "done":             req.Done,
        "errors":           req.Errors,
        "progress_percent": progressPercent,
    },
}
```

**Mới theo specification**:

```go
// Published to: project:{projectID}:{userID}
type ProjectInputMessage struct {
    Status   string         `json:"status"`             // "PROCESSING", "COMPLETED", "FAILED", "PAUSED"
    Progress *ProgressInput `json:"progress,omitempty"` // Overall progress
}
```

#### **Mapping Analysis**:

| Current Field      | New Field                | Mapping Status         |
| ------------------ | ------------------------ | ---------------------- |
| `project_id`       | ❌ Removed               | Moved to topic pattern |
| `status`           | ✅ `status`              | Direct mapping         |
| `total`            | 🔄 `progress.total`      | Needs transformation   |
| `done`             | 🔄 `progress.current`    | Needs transformation   |
| `errors`           | 🔄 `progress.errors`     | Needs transformation   |
| `progress_percent` | 🔄 `progress.percentage` | Needs transformation   |

#### **Required Changes**:

```go
// New project message structure
func buildProjectMessage(req ProgressCallbackRequest) ProjectInputMessage {
    progress := &ProgressInput{
        Current:    int(req.Done),
        Total:      int(req.Total),
        Percentage: calculatePercentage(req.Done, req.Total),
        ETA:        0.0, // TODO: Calculate ETA if needed
        Errors:     transformErrorCount(req.Errors),
    }

    return ProjectInputMessage{
        Status:   mapProjectStatus(req.Status),
        Progress: progress,
    }
}
```

---

## 3. Critical Issues & Solutions

### 3.1 Error Message Transformation

#### **🔄 MODERATE ISSUE**

Current: Error count (`int64`)  
New: Error messages (`[]string`)

#### **Solution**:

```go
func transformErrorCount(errorCount int64) []string {
    if errorCount == 0 {
        return []string{}
    }

    // Generic error message since we don't have details
    return []string{
        fmt.Sprintf("Processing encountered %d errors", errorCount),
    }
}
```

### 3.2 Content Transformation for Dry Run

#### **🔄 MODERATE ISSUE**

Current: Rich `Content` objects  
New: Simplified `ContentItem` objects

#### **Solution**:

```go
func transformContentItems(contents []Content) []ContentItem {
    items := make([]ContentItem, len(contents))

    for i, content := range contents {
        items[i] = ContentItem{
            ID:          content.Meta.ID,
            Text:        content.Content.Text,
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
```

---

## 4. Implementation Plan

### Phase 1: Preparation

1. **Create transformation functions** for content and errors
2. **Update Redis message structures** according to specification
3. **Add new topic publishing** logic

### Phase 2: Implementation

1. **Update dry run handler**:

   ```go
   // Old
   channel := fmt.Sprintf("user_noti:%s", userID)

   // New
   channel := fmt.Sprintf("job:%s:%s", req.JobID, userID)
   message := buildDryRunMessage(req)
   ```

2. **Update project handler**:

   ```go
   // Old
   channel := fmt.Sprintf("user_noti:%s", req.UserID)

   // New
   channel := fmt.Sprintf("project:%s:%s", req.ProjectID, req.UserID)
   message := buildProjectMessage(req)
   ```

### Phase 3: Testing

1. **Unit tests** for transformation functions
2. **Integration tests** with WebSocket service
3. **End-to-end tests** with real crawler callbacks

### Phase 4: Deployment

1. **Deploy WebSocket service** with new subscription patterns
2. **Deploy project service** with new publishing logic
3. **Monitor** for any issues
4. **Remove old topic support** after verification

---

## 5. Risk Assessment

### High Risk

- **❌ Content transformation errors** → Data loss
- **❌ Topic pattern mismatch** → Messages not delivered

### Medium Risk

- **🔄 Message size increase** from detailed structures
- **🔄 Backward compatibility** during transition

### Low Risk

- **✅ Error message format changes** → Graceful degradation
- **✅ Missing optional fields** → WebSocket handles gracefully

---

## 6. Conclusion

### **Dry Run Changes**

- **Crawler message**: ✅ No changes needed
- **Redis structure**: 🔄 Moderate refactor required
- **Critical issues**: ❌ None

### **Project Changes**

- **Collector message**: ✅ No changes needed
- **Redis structure**: 🔄 Moderate refactor required
- **Critical issues**: ✅ None

### **Overall Assessment**

- **Feasibility**: ✅ Fully implementable
- **Breaking changes**: ❌ None for external services
- **Implementation effort**: 🔄 Medium (2-3 days)
- **Risk level**: 🔄 Medium (manageable with proper testing)

**Recommendation**: Proceed with implementation with comprehensive testing.
