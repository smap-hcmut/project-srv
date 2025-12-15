# SMAP Project Service - Service Behavior

> Tài liệu mô tả chi tiết behavior, business logic và các quy tắc xử lý của Project Service

---

## Mục Lục

- [1. Tổng Quan Service](#1-tổng-quan-service)
- [2. Domain Models](#2-domain-models)
- [3. Business Rules](#3-business-rules)
- [4. API Behaviors](#4-api-behaviors)
- [5. State Management](#5-state-management)
- [6. Event Publishing](#6-event-publishing)
- [7. Webhook Handling](#7-webhook-handling)
- [8. Error Handling](#8-error-handling)
- [9. Security & Authorization](#9-security--authorization)

---

## 1. Tổng Quan Service

### 1.1. Mục Đích

**Project Service** là service quản lý vòng đời của các dự án phân tích thương hiệu và đối thủ cạnh tranh trong hệ thống SMAP. Service này đóng vai trò:

- **Project Management**: CRUD operations cho projects
- **Execution Orchestrator**: Khởi tạo và điều phối quá trình crawl dữ liệu
- **Progress Tracker**: Theo dõi tiến độ thực thi project
- **Event Publisher**: Phát sự kiện cho các service khác consume
- **Webhook Receiver**: Nhận callback từ Collector Service

### 1.2. Các Module Chính

```
┌─────────────────────────────────────────────────────────────────┐
│                      PROJECT SERVICE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Project   │  │   Keyword   │  │    State    │             │
│  │   Module    │  │   Module    │  │   Module    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐                              │
│  │   Webhook   │  │ Middleware  │                              │
│  │   Module    │  │   Module    │                              │
│  └─────────────┘  └─────────────┘                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

| Module         | Trách Nhiệm                                    |
| -------------- | ---------------------------------------------- |
| **Project**    | CRUD projects, execute, progress tracking      |
| **Keyword**    | Validate và suggest keywords (LLM integration) |
| **State**      | Quản lý trạng thái project trên Redis          |
| **Webhook**    | Xử lý callback từ Collector Service            |
| **Middleware** | Authentication, CORS, Error handling           |

---

## 2. Domain Models

### 2.1. Project Model

```go
type Project struct {
    ID                   string              // UUID, auto-generated
    Name                 string              // Tên project
    Description          string              // Mô tả (optional)
    Status               ProjectStatus       // Trạng thái project
    FromDate             time.Time           // Ngày bắt đầu crawl
    ToDate               time.Time           // Ngày kết thúc crawl
    BrandName            string              // Tên thương hiệu theo dõi
    BrandKeywords        []string            // Keywords của thương hiệu
    CompetitorNames      []string            // Danh sách đối thủ
    CompetitorKeywordsMap map[string][]string // Map: competitor -> keywords
    CreatedBy            string              // User ID (từ JWT)
    CreatedAt            time.Time           // Thời điểm tạo
    UpdatedAt            time.Time           // Thời điểm cập nhật
    DeletedAt            *time.Time          // Soft delete timestamp
}
```

### 2.2. Project Status

```go
type ProjectStatus string

const (
    ProjectStatusDraft     ProjectStatus = "draft"      // Nháp, chưa execute
    ProjectStatusProcess   ProjectStatus = "process"    // Đang xử lý
    ProjectStatusCompleted ProjectStatus = "completed"  // Hoàn thành
)
```

**Status Lifecycle:**

- **draft**: Initial state when project is created. Project can be fully edited. No Redis execution state exists.
- **process**: Project is being executed. Redis state tracks detailed progress (INITIALIZING → CRAWLING → PROCESSING → DONE/FAILED).
- **completed**: All processing finished. Results available, no re-execution allowed. Redis state retained for historical reference.

**Valid Status Transitions:**

- `draft` → `process` (via Execute API)
- `process` → `completed` (via system/webhook when processing finishes)
- `process` → `draft` (rollback on failure during execution)

**Invalid Transitions:**

- `draft` → `completed` (must go through process state)
- `completed` → `draft` (no going back)
- `completed` → `process` (no re-execution)
- Any manual status change via Patch API (status changes only through Execute or system events)

### 2.3. Execution State (Redis)

```go
type ProjectState struct {
    Status ProjectStatus  // INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED
    Total  int64          // Tổng số items cần xử lý
    Done   int64          // Số items đã hoàn thành
    Errors int64          // Số items bị lỗi
}

type ExecutionStatus string

const (
    ExecutionStatusInitializing ExecutionStatus = "INITIALIZING"
    ExecutionStatusCrawling     ExecutionStatus = "CRAWLING"
    ExecutionStatusProcessing   ExecutionStatus = "PROCESSING"
    ExecutionStatusDone         ExecutionStatus = "DONE"
    ExecutionStatusFailed       ExecutionStatus = "FAILED"
)
```

### 2.4. Scope (User Context)

```go
type Scope struct {
    UserID   string  // UUID của user
    Username string  // Email/username
    Role     Role    // admin, user
}
```

---

## 3. Business Rules

### 3.1. Project Validation Rules

| Rule                  | Mô Tả                                                                      | Error Code |
| --------------------- | -------------------------------------------------------------------------- | ---------- |
| **Date Range**        | `to_date` phải sau `from_date`                                             | 30007      |
| **Status Valid**      | Status phải là một trong 3 giá trị hợp lệ: "draft", "process", "completed" | 30006      |
| **Name Required**     | Tên project không được rỗng                                                | 30002      |
| **Brand Required**    | Brand name không được rỗng khi execute                                     | 30002      |
| **Status Transition** | Status transitions phải tuân theo quy tắc hợp lệ                           | 30009      |

### 3.2. Authorization Rules

| Rule             | Mô Tả                                             | Error Code |
| ---------------- | ------------------------------------------------- | ---------- |
| **Ownership**    | User chỉ có thể truy cập project của mình         | 30005      |
| **JWT Required** | Tất cả API đều yêu cầu JWT token hợp lệ           | 401        |
| **Internal Key** | Webhook endpoints yêu cầu `X-Internal-Key` header | 401        |

### 3.3. Execution Rules

| Rule                    | Mô Tả                                                                  | Error Code |
| ----------------------- | ---------------------------------------------------------------------- | ---------- |
| **Draft Only**          | Chỉ có thể execute project ở trạng thái "draft"                        | 30009      |
| **No Duplicate**        | Không thể execute project đang chạy (status "process")                 | 30008      |
| **Rollback on Failure** | Nếu Redis init hoặc publish event thất bại, rollback status về "draft" | -          |
| **State TTL**           | Redis state có TTL 7 ngày                                              | -          |
| **No Re-execution**     | Project với status "completed" không thể execute lại                   | 30009      |

### 3.4. Soft Delete Rules

- Khi delete, chỉ set `deleted_at = NOW()`
- Tất cả queries đều filter `deleted_at IS NULL`
- Data được giữ lại cho audit/recovery

---

## 4. API Behaviors

### 4.1. Create Project

**Endpoint:** `POST /projects`

**Behavior:**

1. Parse và validate request body
2. Validate date range (`to_date > from_date`)
3. Validate status (nếu có)
4. Normalize keywords qua `keywordUC.Validate()`
5. Set `created_by = scope.UserID`
6. Insert vào PostgreSQL (ID auto-generated)
7. Return project với ID

**Không thực hiện:**

- Không init Redis state
- Không publish RabbitMQ event
- Không trigger crawling

```
Request → Validate → PostgreSQL Insert → Response
```

### 4.2. Execute Project

**Endpoint:** `POST /projects/:id/execute`

**Behavior:**

1. Get project từ PostgreSQL
2. Verify ownership (`project.created_by == scope.user_id`)
3. Verify project status is "draft" (reject if not)
4. Check duplicate execution (query Redis state)
5. Update PostgreSQL status to "process"
6. Init Redis state với status `INITIALIZING`
7. Publish `project.created` event to RabbitMQ
8. Nếu Redis init thất bại → rollback PostgreSQL status về "draft"
9. Nếu publish thất bại → rollback cả Redis state và PostgreSQL status về "draft"
10. Return success

```
Request → Auth Check → Status Check → Duplicate Check → PostgreSQL Update → Redis Init → RabbitMQ Publish → Response
                                                              ↓                  ↓              ↓
                                                        (Rollback on failure) ────────────────┘
```

**Status Transition:** `draft` → `process`

### 4.3. Get Progress

**Endpoint:** `GET /projects/:id/progress`

**Behavior:**

1. Get project từ PostgreSQL (verify ownership)
2. Query Redis state (`smap:proj:{id}`)
3. Nếu có Redis state → return từ Redis
4. Nếu không có → return từ PostgreSQL status

```
Request → Auth Check → Redis Query → Response
                           ↓
                    (Fallback to PostgreSQL)
```

### 4.4. Dry-Run Keywords

**Endpoint:** `POST /projects/dryrun`

**Behavior:**

1. Parse keywords từ request
2. Generate `job_id` (UUID)
3. Store job mapping trong Redis: `job:{job_id}` → `{user_id, project_id}`
4. Publish task to RabbitMQ: `collector.inbound / crawler.dryrun_keyword`
5. Return `job_id` immediately (async processing)

```
Request → Generate Job ID → Redis Store → RabbitMQ Publish → Response (job_id)
```

### 4.5. List/Get Projects

**Behavior chung:**

- Tự động filter theo `created_by = scope.user_id`
- Exclude soft-deleted records (`deleted_at IS NULL`)
- Support filtering by status, search by name
- Pagination với `page` và `limit` parameters

### 4.6. Get Phase Progress

**Endpoint:** `GET /projects/:id/phase-progress`

**Behavior:**

1. Get project từ PostgreSQL (verify ownership)
2. Query Redis state (`smap:proj:{id}`)
3. Build phase progress response with crawl/analyze breakdown
4. Return detailed progress with phase-specific data

**Response:**

```json
{
  "project_id": "uuid",
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

```
Request → Auth Check → Redis Query → Build Phase Response → Response
```

**Note:** This endpoint provides more detailed progress than `GET /projects/:id/progress` by breaking down progress into crawl and analyze phases.

---

## 5. State Management

### 5.1. Redis Key Schema

```
smap:proj:{projectID}    → Hash (project execution state)
job:mapping:{jobID}      → String (dry-run job mapping)
```

### 5.2. Project State Hash

**Key:** `smap:proj:{projectID}`
**TTL:** 7 days (604800 seconds)

| Field            | Type   | Writer            | Description                   |
| ---------------- | ------ | ----------------- | ----------------------------- |
| `status`         | String | Project/Collector | Execution status              |
| `total`          | Int    | Collector         | Tổng items cần xử lý (legacy) |
| `done`           | Int    | Collector         | Items đã hoàn thành (legacy)  |
| `errors`         | Int    | Collector         | Items bị lỗi (legacy)         |
| `crawl_total`    | Int    | Collector         | Tổng items crawl phase        |
| `crawl_done`     | Int    | Collector         | Items crawl đã hoàn thành     |
| `crawl_errors`   | Int    | Collector         | Items crawl bị lỗi            |
| `analyze_total`  | Int    | Collector         | Tổng items analyze phase      |
| `analyze_done`   | Int    | Collector         | Items analyze đã hoàn thành   |
| `analyze_errors` | Int    | Collector         | Items analyze bị lỗi          |

**Note:** Phase-based fields (`crawl_*`, `analyze_*`) are the new format. Legacy fields (`total`, `done`, `errors`) are kept for backward compatibility.

### 5.3. State Transitions

```
                    ┌──────────────┐
                    │ INITIALIZING │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
              ┌─────│   CRAWLING   │─────┐
              │     └──────┬───────┘     │
              │            │             │
              ▼            ▼             ▼
        ┌──────────┐ ┌──────────────┐ ┌────────┐
        │  FAILED  │ │  PROCESSING  │ │ FAILED │
        └──────────┘ └──────┬───────┘ └────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │     DONE     │
                    └──────────────┘
```

### 5.4. State Operations

**Init State (Project Service):**

```go
pipe := redis.Pipeline()
pipe.HSet(key, "status", "INITIALIZING")
// Legacy fields
pipe.HSet(key, "total", "0")
pipe.HSet(key, "done", "0")
pipe.HSet(key, "errors", "0")
// Phase-based fields
pipe.HSet(key, "crawl_total", "0")
pipe.HSet(key, "crawl_done", "0")
pipe.HSet(key, "crawl_errors", "0")
pipe.HSet(key, "analyze_total", "0")
pipe.HSet(key, "analyze_done", "0")
pipe.HSet(key, "analyze_errors", "0")
pipe.Expire(key, 7 * 24 * time.Hour)
pipe.Exec()
```

**Update State - Phase-Based (Collector Service - Recommended):**

```go
// Set crawl phase totals
redis.HSet(key, "crawl_total", crawlCount)
redis.HSet(key, "status", "CRAWLING")

// Increment crawl done
redis.HIncrBy(key, "crawl_done", 1)

// Set analyze phase totals (after crawl complete)
redis.HSet(key, "analyze_total", analyzeCount)
redis.HSet(key, "status", "PROCESSING")

// Increment analyze done
redis.HIncrBy(key, "analyze_done", 1)

// Mark complete
redis.HSet(key, "status", "DONE")
```

**Update State - Legacy (Collector Service - Deprecated):**

```go
// Set total items
redis.HSet(key, "total", count)
redis.HSet(key, "status", "CRAWLING")

// Increment done
redis.HIncrBy(key, "done", 1)

// Increment errors
redis.HIncrBy(key, "errors", 1)

// Mark complete
redis.HSet(key, "status", "DONE")
```

---

## 6. Event Publishing

### 6.1. RabbitMQ Configuration

| Config        | Value         |
| ------------- | ------------- |
| Exchange      | `smap.events` |
| Exchange Type | `topic`       |
| Durable       | `true`        |

### 6.2. Event: `project.created`

**Routing Key:** `project.created`
**Trigger:** Khi user gọi `POST /projects/:id/execute`

**Payload:**

```json
{
  "event_id": "uuid",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "uuid",
    "user_id": "uuid",
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

**Lưu ý quan trọng:**

- `user_id` được include để Collector biết notify cho ai
- Event chỉ được publish sau khi Redis state đã init thành công

### 6.3. Event: `crawler.dryrun_keyword`

**Exchange:** `collector.inbound`
**Routing Key:** `crawler.dryrun_keyword`
**Trigger:** Khi user gọi `POST /projects/dryrun`

**Payload:**

```json
{
  "job_id": "uuid",
  "task_type": "dryrun_keyword",
  "payload": {
    "keywords": ["VinFast", "VF3"],
    "limit_per_keyword": 3,
    "include_comments": true,
    "max_comments": 5
  },
  "attempt": 1,
  "max_attempts": 3,
  "emitted_at": "2025-12-05T10:00:00Z"
}
```

---

## 7. Webhook Handling

### 7.1. Progress Callback

**Endpoint:** `POST /internal/progress/callback`
**Auth:** `X-Internal-Key` header

**Request (New Phase-Based Format - Recommended):**

```json
{
  "project_id": "uuid",
  "user_id": "uuid",
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

**Request (Legacy Flat Format - Deprecated):**

```json
{
  "project_id": "uuid",
  "user_id": "uuid",
  "status": "CRAWLING",
  "total": 1000,
  "done": 250,
  "errors": 5
}
```

**Behavior:**

1. Validate `X-Internal-Key`
2. Detect format (new phase-based vs old flat)
3. If old format: log deprecation warning
4. Calculate progress percent (from phases or flat fields)
5. Determine message type (`project_progress` or `project_completed`)
6. Build WebSocket message with phase structure
7. Publish to Redis Pub/Sub channel `project:{project_id}:{user_id}`

**Message Types:**

- `project_progress`: Khi status là INITIALIZING, CRAWLING, PROCESSING
- `project_completed`: Khi status là DONE hoặc FAILED

**WebSocket Message Format:**

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "uuid",
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

### 7.2. Dry-Run Callback

**Endpoint:** `POST /internal/dryrun/callback`
**Auth:** `X-Internal-Key` header

**Request:**

```json
{
  "job_id": "uuid",
  "status": "success",
  "platform": "tiktok",
  "payload": {
    "content": [...],
    "errors": []
  }
}
```

**Behavior:**

1. Validate `X-Internal-Key`
2. Lookup `user_id` từ Redis job mapping
3. Publish to Redis Pub/Sub channel `user_noti:{user_id}`
4. Message type: `dryrun_result`

### 7.3. Redis Pub/Sub Channels

**Channel Pattern:** `user_noti:{user_id}`

**Message Format:**

```json
{
  "type": "project_progress|project_completed|dryrun_result",
  "payload": {
    // Type-specific payload
  }
}
```

---

## 8. Error Handling

### 8.1. Error Codes

| Code  | Name                       | HTTP Status | Description                                                        |
| ----- | -------------------------- | ----------- | ------------------------------------------------------------------ |
| 30001 | ErrInternal                | 500         | Internal server error                                              |
| 30002 | ErrWrongBody               | 400         | Invalid request body                                               |
| 30003 | ErrInvalidID               | 400         | Invalid project ID format                                          |
| 30004 | ErrProjectNotFound         | 404         | Project không tồn tại                                              |
| 30005 | ErrUnauthorized            | 403         | Không có quyền truy cập                                            |
| 30006 | ErrInvalidStatus           | 400         | Status không hợp lệ (phải là "draft", "process", hoặc "completed") |
| 30007 | ErrInvalidDateRange        | 400         | Date range không hợp lệ                                            |
| 30008 | ErrProjectAlreadyExecuting | 409         | Project đang được execute                                          |
| 30009 | ErrInvalidStatusTransition | 400         | Status transition không hợp lệ                                     |

### 8.2. Error Response Format

```json
{
  "error_code": 30004,
  "message": "project not found"
}
```

### 8.3. Error Handling Strategy

**Layer-specific errors:**

- **Repository Layer:** Database errors, not found
- **UseCase Layer:** Business logic errors, validation
- **Delivery Layer:** HTTP-specific errors, request parsing

**Error Propagation:**

```
Repository Error → UseCase (wrap/transform) → Handler (HTTP response)
```

---

## 9. Security & Authorization

### 9.1. Authentication Methods

**Primary: HttpOnly Cookie**

- Cookie name: `smap_auth_token`
- Attributes: HttpOnly, Secure, SameSite=Lax
- Set by Identity Service

**Legacy: Bearer Token**

- Header: `Authorization: Bearer {token}`
- Deprecated, for backward compatibility

### 9.2. JWT Token Structure

```json
{
  "user_id": "uuid",
  "username": "user@example.com",
  "role": "user",
  "exp": 1234567890
}
```

### 9.3. Authorization Flow

```
Request
    │
    ▼
┌─────────────────┐
│ JWT Middleware  │ ─── Invalid Token ──→ 401 Unauthorized
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Extract Scope   │
│ (user_id, role) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ UseCase Layer   │ ─── Not Owner ──→ 403 Forbidden
│ Ownership Check │
└────────┬────────┘
         │
         ▼
    Continue...
```

### 9.4. Internal API Security

**Webhook Endpoints:**

- Require `X-Internal-Key` header
- Key configured via `INTERNAL_KEY` env var
- No JWT required (service-to-service)

---

## Appendix: Configuration Reference

### Environment Variables

| Variable         | Description                          | Default    |
| ---------------- | ------------------------------------ | ---------- |
| `ENV`            | Environment (production/staging/dev) | production |
| `APP_PORT`       | HTTP server port                     | 8080       |
| `JWT_SECRET`     | JWT signing key                      | -          |
| `POSTGRES_*`     | PostgreSQL connection                | -          |
| `REDIS_HOST`     | Redis address                        | -          |
| `REDIS_STATE_DB` | Redis DB for state                   | 1          |
| `RABBITMQ_URL`   | RabbitMQ connection                  | -          |
| `INTERNAL_KEY`   | Internal API key                     | -          |
| `LLM_PROVIDER`   | LLM provider (gemini)                | gemini     |
| `LLM_API_KEY`    | LLM API key                          | -          |

---

## Appendix A: Project Status State Machine

### PostgreSQL Status State Machine

The project lifecycle follows a simplified three-state model:

```
                    ┌──────────────┐
                    │   [START]    │
                    └──────┬───────┘
                           │
                           │ Create Project
                           ▼
                    ┌──────────────┐
              ┌─────│    draft     │◄────┐
              │     └──────┬───────┘     │
              │            │             │
              │            │ Execute     │ Rollback on
              │            │             │ failure
              │            ▼             │
              │     ┌──────────────┐    │
              │     │   process    │────┘
              │     └──────┬───────┘
              │            │
              │            │ All tasks complete
              │            │ (via webhook)
              │            ▼
              │     ┌──────────────┐
              └────►│  completed   │
                    └──────────────┘
```

**Valid Transitions:**

- `draft` → `process` (via Execute API)
- `process` → `completed` (via system/webhook)
- `process` → `draft` (rollback on failure)

**Invalid Transitions:**

- `draft` → `completed` (must go through process)
- `completed` → `draft` (no going back)
- `completed` → `process` (no re-execution)
- Any transition via Patch API (status changes only through Execute or system)

### Redis Execution State vs PostgreSQL Status

The system maintains two separate status tracking mechanisms:

**PostgreSQL Status (Persistent):**

- Lifecycle state: `draft`, `process`, `completed`
- Persisted permanently
- Visible to users via API
- Updated by Project Service

**Redis Execution State (Runtime):**

- Processing state: `INITIALIZING`, `CRAWLING`, `PROCESSING`, `DONE`, `FAILED`
- Temporary (7-day TTL)
- Used for progress tracking
- Updated by Collector Service

**Relationship:**

```
PostgreSQL "draft" status
    ↓
    No Redis state exists

PostgreSQL "process" status
    ↓
    Contains detailed Redis state:
    - INITIALIZING (just started)
    - CRAWLING (collecting data)
    - PROCESSING (analyzing data)
    - DONE (finished successfully)
    - FAILED (error occurred)
    ↓
PostgreSQL "completed" status (when Redis shows DONE)
    ↓
    Redis state retained for historical reference
```

This separation allows:

- Simple, clear lifecycle states in PostgreSQL
- Detailed progress tracking in Redis
- Clean separation of concerns between services

---

**Last Updated:** December 2025
**Version:** 3.0.0 (Status Model Simplified)
**Maintained By:** SMAP Development Team
