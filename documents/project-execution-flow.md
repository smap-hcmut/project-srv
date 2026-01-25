# Project Execution Flow

> **Related Docs:**
> - [Integration Guide](./integration-guide.md) - Full system integration guide
> - [Dry-Run Data Flow](./dryrun-dataflow.md) - Dry-run keywords flow
> - [WebSocket Progress](./websocket-progress-proposal.md) - WebSocket implementation details

Tài liệu mô tả chi tiết flow từ tạo project đến publish event và khởi tạo state trên Redis.

## Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           USER FLOW                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  1. POST /projects           → Create project (PostgreSQL only)             │
│  2. POST /projects/dryrun    → Optional: Test keywords                      │
│  3. POST /projects/:id/execute → Start execution (Redis + RabbitMQ)         │
│  4. GET /projects/:id/progress → Monitor real-time progress                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Create Project (`POST /projects`)

### Flow

```
HTTP Request → Handler → UseCase → Repository → PostgreSQL
```

### Input

```json
{
  "name": "Project Name",
  "description": "Description",
  "brand_name": "VinFast",
  "brand_keywords": ["VinFast", "VF3"],
  "competitor_keywords": [{ "name": "Toyota", "keywords": ["Toyota", "Vios"] }],
  "from_date": "2025-01-01",
  "to_date": "2025-02-01"
}
```

### Output

```json
{
  "id": "uuid-project-id",
  "name": "Project Name",
  "status": "pending",
  "created_at": "2025-12-05T10:00:00Z"
}
```

### What happens

1. Validate date range (`to_date > from_date`)
2. Validate and normalize keywords via `keywordUC.Validate()`
3. Save to PostgreSQL with status `pending`
4. **NO Redis state, NO RabbitMQ event** - project is just saved

### Files involved

- `internal/project/delivery/http/handler.go` → `Create()`
- `internal/project/usecase/project.go` → `Create()`
- `internal/project/repository/postgre/` → PostgreSQL insert

---

## 2. Execute Project (`POST /projects/:id/execute`)

### Flow

```
HTTP Request → Handler → UseCase → [PostgreSQL check] → [Redis init] → [RabbitMQ publish]
```

### Input

```
POST /projects/uuid-project-id/execute
Authorization: Cookie (smap_auth_token)
```

### Output (Success)

```json
{
  "project_id": "uuid-project-id",
  "status": "executing",
  "message": "Project execution started successfully"
}
```

### Output (Error - Already Executing)

```json
{
  "code": 30008,
  "message": "Project is already executing"
}
```

### Detailed Steps

#### Step 1: Verify Ownership

```go
p, err := uc.repo.Detail(ctx, sc, projectID)
if p.CreatedBy != sc.UserID {
    return project.ErrUnauthorized
}
```

#### Step 2: Check Duplicate Execution

```go
existingState, err := uc.stateUC.GetProjectState(ctx, projectID)
if err == nil && existingState != nil {
    return project.ErrProjectAlreadyExecuting
}
```

- Query Redis key: `smap:proj:{projectID}`
- If exists → return error 30008

#### Step 3: Initialize Redis State

```go
err := uc.stateUC.InitProjectState(ctx, projectID)
```

**Redis Hash created:**

```
Key: smap:proj:{projectID}
TTL: 7 days (604800 seconds)

Fields:
  status: "INITIALIZING"
  total:  "0"
  done:   "0"
  errors: "0"
```

#### Step 4: Publish RabbitMQ Event

```go
event := rabbitmq.ToProjectCreatedEvent(p)
err := uc.producer.PublishProjectCreated(ctx, event)
```

**RabbitMQ Message:**

```
Exchange: smap.events (topic)
Routing Key: project.created
Content-Type: application/json

Body:
{
  "event_id": "uuid-event-id",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "uuid-project-id",
    "user_id": "uuid-user-id",
    "brand_name": "VinFast",
    "brand_keywords": ["VinFast", "VF3"],
    "competitor_names": ["Toyota"],
    "competitor_keywords_map": {
      "Toyota": ["Toyota", "Vios"]
    },
    "date_range": {
      "from": "2025-01-01",
      "to": "2025-02-01"
    }
  }
}
```

#### Step 5: Rollback on Failure

```go
if err := uc.producer.PublishProjectCreated(ctx, event); err != nil {
    // Clean up Redis state
    _ = uc.stateUC.DeleteProjectState(ctx, projectID)
    return fmt.Errorf("failed to publish execution event: %w", err)
}
```

### Files involved

- `internal/project/delivery/http/handler.go` → `Execute()`
- `internal/project/usecase/project.go` → `Execute()`
- `internal/state/usecase/state.go` → `InitProjectState()`, `GetProjectState()`
- `internal/state/repository/redis/state_repo.go` → Redis operations
- `internal/project/delivery/rabbitmq/producer/producer.go` → `PublishProjectCreated()`

---

## 3. Get Progress (`GET /projects/:id/progress`) - Polling API

### Flow

```
HTTP Request → Handler → UseCase → [PostgreSQL auth] → [Redis state] → Response
```

### Output (From Redis)

```json
{
  "project_id": "uuid-project-id",
  "status": "CRAWLING",
  "total_items": 1000,
  "processed_items": 250,
  "failed_items": 5,
  "progress_percent": 25.0
}
```

### Output (Fallback to PostgreSQL)

```json
{
  "project_id": "uuid-project-id",
  "status": "pending",
  "total_items": 0,
  "processed_items": 0,
  "failed_items": 0,
  "progress_percent": 0
}
```

### Files involved

- `internal/project/delivery/http/handler.go` → `GetProgress()`
- `internal/project/usecase/project.go` → `GetProgress()`
- `internal/state/usecase/state.go` → `GetProjectState()`

---

## 4. Real-time Progress via WebSocket (Implemented)

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     WEBSOCKET PROGRESS NOTIFICATION                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  project.created event includes user_id:                                    │
│  {"project_id": "xxx", "user_id": "yyy", ...}                               │
│                                                                             │
│  Collector Service                                                          │
│       │ 1. Update Redis state: HINCRBY smap:proj:{id} done 1                │
│       │ 2. POST /internal/progress/callback                                 │
│       ▼                                                                     │
│  Project Service (Webhook Handler)                                          │
│       │ 3. Publish to Redis Pub/Sub: user_noti:{user_id}                    │
│       ▼                                                                     │
│  Redis Pub/Sub                                                              │
│       │ 4. Subscribe: user_noti:*                                           │
│       ▼                                                                     │
│  WebSocket Service                                                          │
│       │ 5. Send to client                                                   │
│       ▼                                                                     │
│  Client Browser                                                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Webhook Endpoint

**POST /internal/progress/callback**

```bash
curl -X POST http://localhost:8080/internal/progress/callback \
  -H "Content-Type: application/json" \
  -H "X-Internal-Key: your-internal-key" \
  -d '{
    "project_id": "uuid-project-id",
    "user_id": "uuid-user-id",
    "status": "CRAWLING",
    "total": 1000,
    "done": 250,
    "errors": 5
  }'
```

### Message Types

#### Progress Update

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "uuid-project-id",
    "status": "CRAWLING",
    "total": 1000,
    "done": 250,
    "errors": 5,
    "progress_percent": 25.0
  }
}
```

#### Project Completed

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "uuid-project-id",
    "status": "DONE",
    "total": 1000,
    "done": 1000,
    "errors": 10,
    "completed_at": "2025-12-05T15:30:00Z"
  }
}
```

### Client Integration

```javascript
// Connect to WebSocket
const ws = new WebSocket("ws://localhost:8082/ws?token=<jwt_token>");

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch (message.type) {
    case "project_progress":
      updateProgressBar(message.payload);
      break;
    case "project_completed":
      showCompletionNotification(message.payload);
      break;
    case "dryrun_result":
      showDryRunResult(message.payload);
      break;
  }
};
```

### Why WebSocket over Polling?

| Aspect      | Polling API                    | WebSocket               |
| ----------- | ------------------------------ | ----------------------- |
| Latency     | 1-5 seconds (polling interval) | Real-time (<100ms)      |
| Bandwidth   | High (repeated requests)       | Low (push only)         |
| Server Load | High (handle many requests)    | Low (single connection) |
| UX          | Progress bar jumps             | Smooth updates          |

---

## Data Structures

### Redis State (`smap:proj:{projectID}`)

```go
type ProjectState struct {
    Status ProjectStatus `json:"status"`  // INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED
    Total  int64         `json:"total"`   // Total items to process
    Done   int64         `json:"done"`    // Completed items
    Errors int64         `json:"errors"`  // Failed items
}
```

### Status Transitions

```
INITIALIZING → CRAWLING → PROCESSING → DONE
                    ↓           ↓
                 FAILED      FAILED
```

### RabbitMQ Event (`project.created`)

```go
type ProjectCreatedEvent struct {
    EventID   string                `json:"event_id"`
    Timestamp time.Time             `json:"timestamp"`
    Payload   ProjectCreatedPayload `json:"payload"`
}

type ProjectCreatedPayload struct {
    ProjectID             string              `json:"project_id"`
    UserID                string              `json:"user_id"` // For WebSocket progress notifications
    BrandName             string              `json:"brand_name"`
    BrandKeywords         []string            `json:"brand_keywords"`
    CompetitorNames       []string            `json:"competitor_names"`
    CompetitorKeywordsMap map[string][]string `json:"competitor_keywords_map"`
    DateRange             DateRange           `json:"date_range"`
}
```

---

## Integration Test Scenarios

### Scenario 1: Happy Path

```
1. POST /projects → 201 Created (project_id)
2. POST /projects/{id}/execute → 200 OK
3. Verify Redis: HGETALL smap:proj:{id} → status=INITIALIZING
4. Verify RabbitMQ: Message in smap.events with routing key project.created
   - Verify payload contains user_id
5. GET /projects/{id}/progress → status=INITIALIZING, progress=0%
```

### Scenario 2: Duplicate Execution

```
1. POST /projects → 201 Created
2. POST /projects/{id}/execute → 200 OK
3. POST /projects/{id}/execute → 409 Conflict (code: 30008)
```

### Scenario 3: Unauthorized Access

```
1. User A: POST /projects → 201 Created (project_id)
2. User B: POST /projects/{id}/execute → 403 Forbidden (code: 30005)
```

### Scenario 4: RabbitMQ Failure (Rollback)

```
1. POST /projects → 201 Created
2. Stop RabbitMQ
3. POST /projects/{id}/execute → 500 Error
4. Verify Redis: Key smap:proj:{id} does NOT exist (rolled back)
```

### Scenario 5: Progress Tracking (Polling)

```
1. Execute project
2. Collector updates Redis: HINCRBY smap:proj:{id} total 1000
3. Collector updates Redis: HSET smap:proj:{id} status CRAWLING
4. Analytics updates Redis: HINCRBY smap:proj:{id} done 1
5. GET /projects/{id}/progress → total=1000, done=1, progress=0.1%
```

### Scenario 6: Progress Tracking (WebSocket)

```
1. Client connects WebSocket: ws://localhost:8082/ws?token=<jwt>
2. Execute project (user_id included in event)
3. Collector receives project.created event with user_id
4. Collector updates Redis state
5. Collector calls POST /internal/progress/callback
6. Project Service publishes to user_noti:{user_id}
7. WebSocket receives message type=project_progress
8. Client UI updates progress bar in real-time
```

### Scenario 7: Project Completion via WebSocket

```
1. Project executing, client connected via WebSocket
2. Analytics marks last item done
3. Collector calls POST /internal/progress/callback with status=DONE
4. Project Service publishes type=project_completed to user_noti:{user_id}
5. Client receives completion notification
```

---

## Configuration

### Environment Variables

```env
# Redis (DB 0: job mapping, pub/sub)
REDIS_HOST=localhost:6379
REDIS_PASSWORD=password

# Redis State DB (DB 1: project progress tracking)
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### Redis Key Schema

```
smap:proj:{projectID}  → Hash (TTL: 7 days)
```

### RabbitMQ Exchange

```
Exchange: smap.events
Type: topic
Durable: true
Routing Keys:
  - project.created
```

---

## Error Codes

| Code  | Error                     | HTTP Status |
| ----- | ------------------------- | ----------- |
| 30004 | Project not found         | 404         |
| 30005 | Unauthorized              | 403         |
| 30007 | Invalid date range        | 400         |
| 30008 | Project already executing | 409         |
