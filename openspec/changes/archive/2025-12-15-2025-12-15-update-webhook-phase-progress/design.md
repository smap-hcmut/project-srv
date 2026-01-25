# Phase-Based Progress Design Document

## Architecture Overview

### Current Architecture

```
Collector ──webhook──► Project Service ──► Redis ──► WebSocket ──► Client
                           ↓
                    Flat Progress Model
              {total, done, errors, status}
```

### Proposed Architecture

```
Collector ──webhook──► Project Service ──► Redis ──► WebSocket ──► Client
                           ↓
                  Phase-Based Progress Model
         {crawl: {...}, analyze: {...}, overall_progress_percent}
```

## Design Principles

### 1. Phase-Based Progress Tracking

**Rationale**: Project execution has two distinct phases (crawl and analyze) that should be tracked independently.

**Implementation**:

- Separate counters for each phase: total, done, errors
- Independent progress percentage calculation per phase
- Overall progress as weighted average of phases

**Benefits**:

- Clear visibility into which phase is running
- Better error attribution
- More accurate progress estimation

### 2. Backward Compatibility

**Rationale**: Ensure smooth transition without breaking existing integrations.

**Detection Logic**:

```go
func (uc *usecase) HandleProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
    // New format detection: has Crawl or Analyze data
    if req.Crawl.Total > 0 || req.Analyze.Total > 0 {
        return uc.handleNewFormat(ctx, req)
    }
    // Fallback to old format
    return uc.handleOldFormat(ctx, req)
}
```

**Benefits**:

- Zero downtime migration
- Services can be updated independently
- Rollback capability maintained

### 3. Progress Calculation Model

**Rationale**: Provide accurate overall progress from phase data.

**Formula**:

```go
func (s *ProjectState) OverallProgressPercent() float64 {
    crawlProgress := s.CrawlProgressPercent()
    analyzeProgress := s.AnalyzeProgressPercent()
    // Equal weight for both phases (50/50)
    return crawlProgress/2 + analyzeProgress/2
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
```

**Note**: Errors count toward progress (item processed, even if failed).

## Data Flow Design

### New Format Flow

```
1. Collector sends webhook → Project Service
   ProgressCallbackRequest {
     ProjectID, UserID, Status,
     Crawl: { Total, Done, Errors, ProgressPercent },
     Analyze: { Total, Done, Errors, ProgressPercent },
     OverallProgressPercent
   }

2. Project Service validates and stores state in Redis

3. Transform to WebSocket message:
   - Preserve phase structure
   - Include overall progress

4. Publish to: project:{projectID}:{userID}
   ProjectMessage {
     Status, Crawl, Analyze, OverallProgressPercent
   }
```

### Old Format Flow (Backward Compat)

```
1. Collector sends old webhook → Project Service
   OldProgressCallbackRequest {
     ProjectID, UserID, Status, Total, Done, Errors
   }

2. Project Service detects old format (no Crawl/Analyze data)

3. Map to single phase based on status:
   - CRAWLING → Crawl phase
   - ANALYZING → Analyze phase
   - Other → Crawl phase (default)

4. Continue with new format processing
```

## Message Structure Design

### PhaseProgress Structure

```go
type PhaseProgress struct {
    Total           int64   `json:"total"`
    Done            int64   `json:"done"`
    Errors          int64   `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}
```

### Request Structure

```go
type ProgressCallbackRequest struct {
    ProjectID              string        `json:"project_id" binding:"required"`
    UserID                 string        `json:"user_id" binding:"required"`
    Status                 string        `json:"status" binding:"required"`
    Crawl                  PhaseProgress `json:"crawl"`
    Analyze                PhaseProgress `json:"analyze"`
    OverallProgressPercent float64       `json:"overall_progress_percent"`

    // Old format fields (deprecated, for backward compat)
    Total  int64 `json:"total,omitempty"`
    Done   int64 `json:"done,omitempty"`
    Errors int64 `json:"errors,omitempty"`
}
```

### Response Structure

```go
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

## State Model Design

### Redis Hash Fields

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

### State Struct

```go
type ProjectState struct {
    Status ProjectStatus `json:"status"`

    // Crawl phase counters
    CrawlTotal  int64 `json:"crawl_total"`
    CrawlDone   int64 `json:"crawl_done"`
    CrawlErrors int64 `json:"crawl_errors"`

    // Analyze phase counters
    AnalyzeTotal  int64 `json:"analyze_total"`
    AnalyzeDone   int64 `json:"analyze_done"`
    AnalyzeErrors int64 `json:"analyze_errors"`
}
```

## Status Model

### Status Values

| Status         | Description                          |
| -------------- | ------------------------------------ |
| `INITIALIZING` | Project created, waiting to start    |
| `PROCESSING`   | Active processing (crawl or analyze) |
| `DONE`         | All phases completed successfully    |
| `FAILED`       | Processing failed                    |

### Status Determination

- Use `PROCESSING` for both crawl and analyze phases
- Phase details indicate which phase is active
- `DONE` when both phases complete
- `FAILED` on critical errors

## WebSocket Message Types

### Progress Message

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

### Completion Message

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

## Error Handling Strategy

### Validation Errors

- Required fields: project_id, user_id, status
- Phase data validation: non-negative values
- Progress percent: 0.0 to 100.0 range

### Transformation Errors

- Log with full context
- Continue with partial data if possible
- Don't fail entire request for non-critical errors

### Publishing Errors

- Retry with exponential backoff
- Log failure with channel and message details
- Alert on repeated failures

## Testing Strategy

### Unit Tests

- Test new format handling
- Test old format backward compatibility
- Test progress calculation methods
- Test edge cases (zero total, all errors)

### Integration Tests

- End-to-end webhook to Redis publish
- WebSocket message delivery
- API response format

### Compatibility Tests

- Old format continues to work
- Mixed format scenarios
- Transition period handling

## Migration Strategy

### Phase 1: Deploy Project Service

- Deploy with backward compatibility
- Both formats supported
- Monitor for issues

### Phase 2: Update Collector Service

- Start sending new format
- Old format as fallback

### Phase 3: Update Frontend

- Display phase-based progress
- Handle both formats during transition

### Phase 4: Cleanup

- Remove old format support (optional)
- Update documentation
