# Integration Guide: Project Service & WebSocket

**Ngày tạo:** 2025-12-15  
**Cập nhật:** 2025-12-15 - Collector là single point gọi webhook  
**Liên quan:** [project-state-proposal.md](./project-state-proposal.md)

---

## 1. Tổng quan

Project Service và WebSocket Service nhận webhook từ **Collector Service** (single point):

```
Crawler ────┐
            │
            ▼
        Collector ──webhook──► Project Service ──► WebSocket ──► Client
            ▲
            │
Analytics ──┘
```

**Lưu ý:** Cả Crawler results và Analytics results đều đi qua Collector, Collector là nơi duy nhất gọi webhook.

---

## 2. Webhook Format Changes

### 2.1. Old Format (deprecated)

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "CRAWLING",
  "total": 100,
  "done": 50,
  "errors": 2
}
```

### 2.2. New Format

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "PROCESSING",
  "crawl": {
    "total": 100,
    "done": 100,
    "errors": 0,
    "progress_percent": 100.0
  },
  "analyze": {
    "total": 100,
    "done": 45,
    "errors": 2,
    "progress_percent": 47.0
  },
  "overall_progress_percent": 73.5
}
```

---

## 3. Project Service Changes

### 3.1. Request Struct Update

```go
// project-service/internal/webhook/types.go

// NEW structs
type PhaseProgress struct {
    Total           int64   `json:"total"`
    Done            int64   `json:"done"`
    Errors          int64   `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}

type ProgressCallbackRequest struct {
    ProjectID              string        `json:"project_id" binding:"required"`
    UserID                 string        `json:"user_id" binding:"required"`
    Status                 string        `json:"status" binding:"required"`
    Crawl                  PhaseProgress `json:"crawl"`
    Analyze                PhaseProgress `json:"analyze"`
    OverallProgressPercent float64       `json:"overall_progress_percent"`
}
```

### 3.2. Handler Update

```go
// project-service/internal/webhook/usecase/webhook.go

func (uc *usecase) HandleProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
    channel := fmt.Sprintf("user_noti:%s", req.UserID)

    // Determine message type
    messageType := MessageTypeProjectProgress
    if req.Status == "DONE" || req.Status == "FAILED" {
        messageType = MessageTypeProjectCompleted
    }

    // Build message with new format
    message := map[string]interface{}{
        "type": messageType,
        "payload": map[string]interface{}{
            "project_id": req.ProjectID,
            "status":     req.Status,
            "crawl": map[string]interface{}{
                "total":            req.Crawl.Total,
                "done":             req.Crawl.Done,
                "errors":           req.Crawl.Errors,
                "progress_percent": req.Crawl.ProgressPercent,
            },
            "analyze": map[string]interface{}{
                "total":            req.Analyze.Total,
                "done":             req.Analyze.Done,
                "errors":           req.Analyze.Errors,
                "progress_percent": req.Analyze.ProgressPercent,
            },
            "overall_progress_percent": req.OverallProgressPercent,
        },
    }

    return uc.redisClient.Publish(ctx, channel, message)
}
```

### 3.3. Backward Compatibility

```go
// Support both old and new format during transition
func (uc *usecase) HandleProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
    // Check if new format (has Crawl field with data)
    if req.Crawl.Total > 0 || req.Analyze.Total > 0 {
        return uc.handleNewFormat(ctx, req)
    }

    // Fallback to old format handling (deprecated)
    // This can be removed after all services are updated
    return uc.handleOldFormat(ctx, req)
}
```

---

## 4. WebSocket Message Format

### 4.1. Progress Message

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "PROCESSING",
    "crawl": {
      "total": 100,
      "done": 80,
      "errors": 2,
      "progress_percent": 82.0
    },
    "analyze": {
      "total": 78,
      "done": 45,
      "errors": 1,
      "progress_percent": 59.0
    },
    "overall_progress_percent": 70.5
  }
}
```

### 4.2. Completion Message

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "proj_xyz",
    "status": "DONE",
    "crawl": {
      "total": 100,
      "done": 98,
      "errors": 2,
      "progress_percent": 100.0
    },
    "analyze": {
      "total": 98,
      "done": 95,
      "errors": 3,
      "progress_percent": 100.0
    },
    "overall_progress_percent": 100.0
  }
}
```

### 4.3. Message Types

| Type                | When                          |
| ------------------- | ----------------------------- |
| `project_progress`  | Mỗi khi có update (throttled) |
| `project_completed` | Khi status = DONE hoặc FAILED |

---

## 5. API Response Changes

### 5.1. GET /projects/:id/progress

**New Response:**

```json
{
  "project_id": "proj_xyz",
  "status": "PROCESSING",
  "crawl": {
    "total": 100,
    "done": 100,
    "errors": 0,
    "progress_percent": 100.0
  },
  "analyze": {
    "total": 100,
    "done": 45,
    "errors": 2,
    "progress_percent": 47.0
  },
  "overall_progress_percent": 73.5
}
```

### 5.2. Response Struct

```go
// project-service/internal/project/delivery/http/types.go

type PhaseProgressResponse struct {
    Total           int64   `json:"total"`
    Done            int64   `json:"done"`
    Errors          int64   `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}

type ProjectProgressResponse struct {
    ProjectID              string                `json:"project_id"`
    Status                 string                `json:"status"`
    Crawl                  PhaseProgressResponse `json:"crawl"`
    Analyze                PhaseProgressResponse `json:"analyze"`
    OverallProgressPercent float64               `json:"overall_progress_percent"`
}
```

### 5.3. Handler

```go
// project-service/internal/project/delivery/http/handler.go

func (h *handler) GetProgress(c *gin.Context) {
    projectID := c.Param("id")

    // Get state from Redis
    state, err := h.stateUC.GetState(c.Request.Context(), projectID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
        return
    }

    response := ProjectProgressResponse{
        ProjectID: projectID,
        Status:    string(state.Status),
        Crawl: PhaseProgressResponse{
            Total:           state.CrawlTotal,
            Done:            state.CrawlDone,
            Errors:          state.CrawlErrors,
            ProgressPercent: state.CrawlProgressPercent(),
        },
        Analyze: PhaseProgressResponse{
            Total:           state.AnalyzeTotal,
            Done:            state.AnalyzeDone,
            Errors:          state.AnalyzeErrors,
            ProgressPercent: state.AnalyzeProgressPercent(),
        },
        OverallProgressPercent: state.OverallProgressPercent(),
    }

    c.JSON(http.StatusOK, response)
}
```

---

## 6. State Model Update

### 6.1. Project Service State Model

```go
// project-service/internal/model/state.go

type ProjectStatus string

const (
    ProjectStatusInitializing ProjectStatus = "INITIALIZING"
    ProjectStatusProcessing   ProjectStatus = "PROCESSING"
    ProjectStatusDone         ProjectStatus = "DONE"
    ProjectStatusFailed       ProjectStatus = "FAILED"
)

type ProjectState struct {
    Status ProjectStatus `json:"status"`

    // Crawl counters
    CrawlTotal  int64 `json:"crawl_total"`
    CrawlDone   int64 `json:"crawl_done"`
    CrawlErrors int64 `json:"crawl_errors"`

    // Analyze counters
    AnalyzeTotal  int64 `json:"analyze_total"`
    AnalyzeDone   int64 `json:"analyze_done"`
    AnalyzeErrors int64 `json:"analyze_errors"`
}

func (s *ProjectState) CrawlProgressPercent() float64 {
    if s.CrawlTotal <= 0 {
        return 0
    }
    return float64(s.CrawlDone+s.CrawlErrors) / float64(s.CrawlTotal) * 100
}

func (s *ProjectState) AnalyzeProgressPercent() float64 {
    if s.AnalyzeTotal <= 0 {
        return 0
    }
    return float64(s.AnalyzeDone+s.AnalyzeErrors) / float64(s.AnalyzeTotal) * 100
}

func (s *ProjectState) OverallProgressPercent() float64 {
    return s.CrawlProgressPercent()/2 + s.AnalyzeProgressPercent()/2
}
```

### 6.2. Redis Field Names

```go
const (
    FieldStatus        = "status"
    FieldCrawlTotal    = "crawl_total"
    FieldCrawlDone     = "crawl_done"
    FieldCrawlErrors   = "crawl_errors"
    FieldAnalyzeTotal  = "analyze_total"
    FieldAnalyzeDone   = "analyze_done"
    FieldAnalyzeErrors = "analyze_errors"
)
```

---

## 7. Frontend/Client Changes

### 7.1. TypeScript Types

```typescript
// types/project.ts

interface PhaseProgress {
  total: number;
  done: number;
  errors: number;
  progress_percent: number;
}

interface ProjectProgress {
  project_id: string;
  status: "INITIALIZING" | "PROCESSING" | "DONE" | "FAILED";
  crawl: PhaseProgress;
  analyze: PhaseProgress;
  overall_progress_percent: number;
}

// WebSocket message
interface WebSocketMessage {
  type: "project_progress" | "project_completed" | "dryrun_result";
  payload: ProjectProgress;
}
```

### 7.2. UI Display Example

```tsx
// components/ProjectProgressBar.tsx

function ProjectProgressBar({ progress }: { progress: ProjectProgress }) {
  const getStatusLabel = (status: string) => {
    switch (status) {
      case "INITIALIZING":
        return "Đang khởi tạo...";
      case "PROCESSING":
        return "Đang xử lý...";
      case "DONE":
        return "Hoàn thành";
      case "FAILED":
        return "Thất bại";
      default:
        return status;
    }
  };

  return (
    <div className="progress-container">
      <div className="status">{getStatusLabel(progress.status)}</div>

      <div className="phase">
        <div className="phase-label">
          <span>Thu thập dữ liệu</span>
          <span>
            {progress.crawl.done}/{progress.crawl.total}
          </span>
        </div>
        <ProgressBar value={progress.crawl.progress_percent} />
        {progress.crawl.errors > 0 && (
          <span className="errors">{progress.crawl.errors} lỗi</span>
        )}
      </div>

      <div className="phase">
        <div className="phase-label">
          <span>Phân tích</span>
          <span>
            {progress.analyze.done}/{progress.analyze.total}
          </span>
        </div>
        <ProgressBar value={progress.analyze.progress_percent} />
        {progress.analyze.errors > 0 && (
          <span className="errors">{progress.analyze.errors} lỗi</span>
        )}
      </div>

      <div className="overall">
        <div className="overall-label">
          <span>Tổng tiến độ</span>
          <span>{progress.overall_progress_percent.toFixed(1)}%</span>
        </div>
        <ProgressBar value={progress.overall_progress_percent} />
      </div>
    </div>
  );
}
```

---

## 8. Files to Modify

### Project Service

| File                                        | Change                                            |
| ------------------------------------------- | ------------------------------------------------- |
| `internal/webhook/types.go`                 | Add PhaseProgress, update ProgressCallbackRequest |
| `internal/webhook/usecase/webhook.go`       | Update handler, add backward compat               |
| `internal/model/state.go`                   | Update ProjectState struct                        |
| `internal/state/repository/redis/redis.go`  | Update field names                                |
| `internal/project/delivery/http/types.go`   | Add response structs                              |
| `internal/project/delivery/http/handler.go` | Update GetProgress handler                        |

### WebSocket Service

| File                    | Change                                     |
| ----------------------- | ------------------------------------------ |
| (No code change needed) | Just passes through Redis Pub/Sub messages |

### Frontend

| File                                | Change                  |
| ----------------------------------- | ----------------------- |
| `types/project.ts`                  | Update TypeScript types |
| `components/ProjectProgressBar.tsx` | Update UI component     |

---

## 9. Checklist

### Project Service

- [x] Update `internal/webhook/types.go` - request struct
- [x] Update `internal/webhook/usecase/webhook.go` - handler
- [x] Update `internal/model/state.go` - state struct
- [x] Update `internal/state/repository/redis/redis.go` - field names
- [x] Update `internal/project/delivery/http/types.go` - response struct
- [x] Update `internal/project/delivery/http/handler.go` - GetProgress
- [x] Add backward compatibility for old webhook format
- [x] Test webhook handler with new format
- [x] Test GET /progress API

### WebSocket Service

- [ ] Verify message passthrough works (no code change needed)

### Frontend

- [ ] Update TypeScript types
- [ ] Update progress display components
- [ ] Test WebSocket message handling

---

## 10. Migration Notes

**Implemented:** 2025-12-15

### What Changed

1. **Webhook Format**: Added phase-based progress (crawl/analyze) alongside flat progress
2. **State Model**: `ProjectState` now tracks separate counters for crawl and analyze phases
3. **WebSocket Messages**: Now include `crawl`, `analyze`, and `overall_progress_percent` fields
4. **New API**: `GET /projects/:id/phase-progress` returns detailed phase progress

### Backward Compatibility

- Old webhook format (flat `total/done/errors`) is still supported
- Old format triggers deprecation warning in logs
- Old format is mapped to crawl phase internally
- No breaking changes for existing clients

### Rollback Procedure

1. Revert Project Service deployment to previous version
2. Old format webhooks will continue to work
3. No database migration needed (Redis state is ephemeral)

### Files Modified

| File                                            | Description                               |
| ----------------------------------------------- | ----------------------------------------- |
| `internal/webhook/type.go`                      | Added `PhaseProgress` struct              |
| `internal/webhook/usecase/webhook.go`           | Updated handler with format detection     |
| `internal/webhook/usecase/transformers.go`      | Added format transformation functions     |
| `internal/model/state.go`                       | Added phase counters and progress methods |
| `internal/state/repository/redis/state_repo.go` | Updated Redis field constants             |
| `internal/project/type.go`                      | Added output types                        |
| `internal/project/interface.go`                 | Added `GetPhaseProgress` method           |
| `internal/project/usecase/project.go`           | Implemented `GetPhaseProgress`            |
| `internal/project/delivery/http/handler.go`     | Added `GetPhaseProgress` handler          |
| `internal/project/delivery/http/presenter.go`   | Added response presenters                 |
| `internal/project/delivery/http/routes.go`      | Added new route                           |

### Test Coverage

- Unit tests: `transformers_test.go`, `state_test.go`
- Integration tests: `handlers_integration_test.go`
- E2E tests: `e2e_test.go`
- All 40+ tests pass
