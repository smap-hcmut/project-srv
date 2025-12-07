# SMAP Integration Guide

> Hướng dẫn tích hợp Project Service với các service khác trong hệ thống SMAP

---

## Mục Lục

- [1. System Overview](#1-system-overview)
- [2. Service Dependencies](#2-service-dependencies)
- [3. Communication Patterns](#3-communication-patterns)
- [4. Integration với Collector Service](#4-integration-với-collector-service)
- [5. Integration với WebSocket Service](#5-integration-với-websocket-service)
- [6. Integration với Identity Service](#6-integration-với-identity-service)
- [7. Integration với Frontend](#7-integration-với-frontend)
- [8. Data Flow Diagrams](#8-data-flow-diagrams)
- [9. Configuration Reference](#9-configuration-reference)
- [10. Testing & Troubleshooting](#10-testing--troubleshooting)

---

## 1. System Overview

### 1.1. Services trong hệ thống SMAP

| Service               | Port | Responsibility                                          |
| --------------------- | ---- | ------------------------------------------------------- |
| **Project Service**   | 8080 | Project CRUD, execution orchestration, webhook handling |
| **Collector Service** | 8081 | Crawl orchestration, progress updates                   |
| **WebSocket Service** | 8082 | Real-time notifications to clients                      |
| **Identity Service**  | 8083 | Authentication, user management, JWT issuing            |

### 1.2. Infrastructure Components

| Component      | Purpose                                |
| -------------- | -------------------------------------- |
| **PostgreSQL** | Primary database cho Project Service   |
| **Redis**      | State management, Pub/Sub, job mapping |
| **RabbitMQ**   | Async event messaging giữa services    |

---

## 2. Service Dependencies

### 2.1. Project Service Dependencies

```
Project Service
├── PostgreSQL (Required)
│   └── Projects data persistence
├── Redis (Required)
│   ├── DB 0: Job mapping, Pub/Sub
│   └── DB 1: Project execution state
├── RabbitMQ (Required)
│   └── Event publishing to Collector
└── Identity Service (Indirect)
    └── JWT token validation (stateless)
```

### 2.2. Dependency Matrix

| Dependency        | Type          | Required    | Fallback             |
| ----------------- | ------------- | ----------- | -------------------- |
| PostgreSQL        | Database      | ✅ Yes      | None                 |
| Redis             | Cache/Pub-Sub | ✅ Yes      | None                 |
| RabbitMQ          | Message Queue | ✅ Yes      | None                 |
| Identity Service  | Auth Provider | ⚠️ Indirect | None (JWT stateless) |
| Collector Service | Consumer      | ⚠️ Async    | Events queued        |
| WebSocket Service | Subscriber    | ⚠️ Async    | Polling fallback     |

---

## 3. Communication Patterns

### 3.1. Pattern Overview

| Pattern             | From → To           | Use Case                |
| ------------------- | ------------------- | ----------------------- |
| **RabbitMQ Events** | Project → Collector | Async task dispatch     |
| **HTTP Webhooks**   | Collector → Project | Progress callbacks      |
| **Redis Pub/Sub**   | Project → WebSocket | Real-time notifications |
| **HTTP REST**       | Frontend → Project  | API calls               |
| **JWT Token**       | Identity → Project  | Authentication          |

### 3.2. RabbitMQ Events (Async)

**Exchange:** `smap.events` (topic)

| Routing Key       | Producer | Consumer  | Description               |
| ----------------- | -------- | --------- | ------------------------- |
| `project.created` | Project  | Collector | Project execution started |

**Exchange:** `collector.inbound` (direct)

| Routing Key              | Producer | Consumer  | Description          |
| ------------------------ | -------- | --------- | -------------------- |
| `crawler.dryrun_keyword` | Project  | Collector | Dry-run keyword test |

### 3.3. HTTP Webhooks (Sync)

| Endpoint                           | Caller    | Handler | Description     |
| ---------------------------------- | --------- | ------- | --------------- |
| `POST /internal/progress/callback` | Collector | Project | Progress update |
| `POST /internal/dryrun/callback`   | Collector | Project | Dry-run result  |

### 3.4. Redis Pub/Sub (Real-time)

| Channel Pattern       | Publisher | Subscriber | Description        |
| --------------------- | --------- | ---------- | ------------------ |
| `user_noti:{user_id}` | Project   | WebSocket  | User notifications |

---

## 4. Integration với Collector Service

### 4.1. Tổng Quan

```
┌─────────────────────────────────────────────────────────────────┐
│                PROJECT ↔ COLLECTOR INTEGRATION                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Project Service                    Collector Service           │
│  ┌─────────────┐                   ┌─────────────┐              │
│  │   Execute   │ ──RabbitMQ──────▶ │   Consume   │              │
│  │   Project   │   project.created │   Event     │              │
│  └─────────────┘                   └──────┬──────┘              │
│                                           │                     │
│                                           ▼                     │
│                                    ┌─────────────┐              │
│                                    │   Dispatch  │              │
│                                    │   Crawlers  │              │
│                                    └──────┬──────┘              │
│                                           │                     │
│  ┌─────────────┐                          │                     │
│  │   Webhook   │ ◀──HTTP POST────────────┘                      │
│  │   Handler   │   /internal/progress/callback                  │
│  └─────────────┘                                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2. Event: project.created

**Khi nào publish:** User gọi `POST /projects/:id/execute`

**Event Schema:**

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "uuid-project-id",
    "user_id": "uuid-user-id",
    "brand_name": "VinFast",
    "brand_keywords": ["VinFast", "VF3", "VF8"],
    "competitor_names": ["Toyota", "Honda"],
    "competitor_keywords_map": {
      "Toyota": ["Toyota", "Vios", "Camry"],
      "Honda": ["Honda", "City", "Civic"]
    },
    "date_range": {
      "from": "2025-01-01",
      "to": "2025-02-01"
    }
  }
}
```

**Lưu ý quan trọng:**

- `user_id` được include để Collector biết notify cho user nào
- Collector cần lưu mapping `project_id → user_id` để gọi webhook

### 4.3. Collector Implementation Guide

#### 4.3.1. Consume Event

```python
# Python example
def handle_project_created(event):
    payload = event['payload']

    project_id = payload['project_id']
    user_id = payload['user_id']  # QUAN TRỌNG: Lưu lại để gọi webhook

    # Store mapping
    store_project_user_mapping(project_id, user_id)

    # Dispatch crawlers
    dispatch_crawlers(
        project_id=project_id,
        brand_keywords=payload['brand_keywords'],
        competitor_keywords_map=payload['competitor_keywords_map'],
        date_range=payload['date_range']
    )
```

#### 4.3.2. Update Redis State

```python
def update_project_state(project_id, field, value):
    key = f"smap:proj:{project_id}"

    if field == "total":
        redis.hset(key, "total", value)
        redis.hset(key, "status", "CRAWLING")
    elif field == "done":
        redis.hincrby(key, "done", 1)
    elif field == "errors":
        redis.hincrby(key, "errors", 1)
    elif field == "status":
        redis.hset(key, "status", value)
```

#### 4.3.3. Call Progress Webhook

```python
PROJECT_SERVICE_URL = "http://project-service:8080"
INTERNAL_KEY = os.getenv("PROJECT_INTERNAL_KEY")

def notify_progress(project_id, user_id):
    key = f"smap:proj:{project_id}"
    state = redis.hgetall(key)

    response = requests.post(
        f"{PROJECT_SERVICE_URL}/internal/progress/callback",
        headers={"X-Internal-Key": INTERNAL_KEY},
        json={
            "project_id": project_id,
            "user_id": user_id,
            "status": state["status"],
            "total": int(state["total"]),
            "done": int(state["done"]),
            "errors": int(state["errors"])
        }
    )
    return response.status_code == 200
```

#### 4.3.4. Throttling Recommendation

```python
# Không gọi webhook mỗi item - throttle tối thiểu 5 giây
last_notify_time = {}
THROTTLE_INTERVAL = 5  # seconds

def should_notify(project_id):
    now = time.time()
    last = last_notify_time.get(project_id, 0)
    if now - last > THROTTLE_INTERVAL:
        last_notify_time[project_id] = now
        return True
    return False

def on_item_crawled(project_id, user_id, is_error=False):
    # Always update Redis
    if is_error:
        update_project_state(project_id, "errors", 1)
    update_project_state(project_id, "done", 1)

    # Throttle webhook calls
    if should_notify(project_id):
        notify_progress(project_id, user_id)
```

### 4.4. Webhook Timing Guide

| Event             | Redis Update                                 | Webhook Call      |
| ----------------- | -------------------------------------------- | ----------------- |
| Found total items | `HSET total {count}`, `HSET status CRAWLING` | ✅ Immediately    |
| Crawled 1 item    | `HINCRBY done 1`                             | ⏱️ Throttled (5s) |
| Item failed       | `HINCRBY errors 1`                           | ⏱️ Throttled (5s) |
| All done          | `HSET status DONE`                           | ✅ Immediately    |
| Fatal error       | `HSET status FAILED`                         | ✅ Immediately    |

### 4.5. Collector Checklist

- [x] Declare queue `collector.project.created`
- [x] Bind to exchange `smap.events` với routing key `project.created`
- [x] Parse `ProjectCreatedEvent` schema
- [x] Extract và lưu `user_id` từ payload
- [x] Connect to Redis DB 1 (state)
- [x] Implement `update_project_state()` function
- [x] Implement `notify_progress()` function
- [x] Add throttling (5 giây minimum)
- [x] Retry webhook calls với exponential backoff

---

## 5. Integration với WebSocket Service

### 5.1. Tổng Quan

```
┌─────────────────────────────────────────────────────────────────┐
│              PROJECT → WEBSOCKET INTEGRATION                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Project Service          Redis           WebSocket Service     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│  │   Webhook   │───▶│   Pub/Sub   │───▶│  Subscribe  │          │
│  │   Handler   │    │   Channel   │    │  Handler    │          │
│  └─────────────┘    └─────────────┘    └──────┬──────┘          │
│                                               │                 │
│                                               ▼                 │
│                                        ┌─────────────┐          │
│                                        │    Send     │          │
│                                        │  to Client  │          │
│                                        └─────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2. Redis Pub/Sub Channel

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

### 5.3. Message Types

#### 5.3.1. project_progress

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "uuid",
    "status": "CRAWLING",
    "total": 1000,
    "done": 250,
    "errors": 5,
    "progress_percent": 25.0
  }
}
```

#### 5.3.2. project_completed

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "uuid",
    "status": "DONE",
    "total": 1000,
    "done": 1000,
    "errors": 10,
    "progress_percent": 100.0
  }
}
```

#### 5.3.3. dryrun_result

```json
{
  "type": "dryrun_result",
  "payload": {
    "job_id": "uuid",
    "platform": "tiktok",
    "status": "success",
    "content": [...],
    "errors": []
  }
}
```

### 5.4. WebSocket Implementation Guide

#### 5.4.1. Subscribe to Redis

```go
// Subscribe to all user notification channels
pubsub := redis.PSubscribe(ctx, "user_noti:*")

for msg := range pubsub.Channel() {
    // Extract user_id from channel name
    // channel format: "user_noti:{user_id}"
    userID := strings.TrimPrefix(msg.Channel, "user_noti:")

    // Parse message
    var message map[string]interface{}
    json.Unmarshal([]byte(msg.Payload), &message)

    // Send to user's WebSocket connections
    hub.SendToUser(userID, msg.Payload)
}
```

#### 5.4.2. WebSocket Checklist

- [x] Subscribe to `user_noti:*` pattern
- [x] Extract `user_id` từ channel name
- [x] Route messages to correct user connections
- [x] Handle `project_progress` messages
- [x] Handle `project_completed` messages
- [x] Handle `dryrun_result` messages
- [x] Handle connection/disconnection gracefully

---

## 6. Integration với Identity Service

### 6.1. Authentication Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION FLOW                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Client              Identity Service        Project Service    │
│    │                       │                       │            │
│    │ POST /login           │                       │            │
│    │──────────────────────▶│                       │            │
│    │                       │                       │            │
│    │ Set-Cookie: smap_auth_token=JWT               │            │
│    │◀──────────────────────│                       │            │
│    │                       │                       │            │
│    │ GET /projects (with cookie)                   │            │
│    │──────────────────────────────────────────────▶│            │
│    │                       │                       │            │
│    │                       │    Validate JWT       │            │
│    │                       │    (stateless)        │            │
│    │                       │                       │            │
│    │ Response                                      │            │
│    │◀──────────────────────────────────────────────│            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2. JWT Token Structure

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "user@example.com",
  "role": "user",
  "exp": 1733400000
}
```

### 6.3. Authentication Methods

| Method              | Header/Cookie                 | Status        |
| ------------------- | ----------------------------- | ------------- |
| **HttpOnly Cookie** | `Cookie: smap_auth_token=JWT` | ✅ Primary    |
| **Bearer Token**    | `Authorization: Bearer JWT`   | ⚠️ Deprecated |

### 6.4. Stateless Integration

- Project Service **không gọi** Identity Service trực tiếp
- JWT token được validate locally bằng shared secret
- User info được extract từ JWT claims
- Không cần database lookup cho authentication

---

## 7. Integration với Frontend

### 7.1. API Endpoints

| Method | Endpoint                 | Auth   | Description            |
| ------ | ------------------------ | ------ | ---------------------- |
| POST   | `/projects`              | Cookie | Create project         |
| GET    | `/projects`              | Cookie | List projects          |
| GET    | `/projects/:id`          | Cookie | Get project detail     |
| PUT    | `/projects/:id`          | Cookie | Update project         |
| DELETE | `/projects/:id`          | Cookie | Delete project         |
| POST   | `/projects/:id/execute`  | Cookie | Start execution        |
| GET    | `/projects/:id/progress` | Cookie | Get progress (polling) |
| POST   | `/projects/dryrun`       | Cookie | Start dry-run          |

### 7.2. WebSocket Connection

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

ws.onerror = (error) => {
  // Fallback to polling
  startPolling();
};
```

### 7.3. Polling Fallback

```javascript
// Fallback when WebSocket fails
async function pollProgress(projectId) {
  const response = await fetch(`/projects/${projectId}/progress`, {
    credentials: "include", // Send cookies
  });
  const data = await response.json();
  updateProgressBar(data);
}

// Poll every 5 seconds
setInterval(() => pollProgress(projectId), 5000);
```

### 7.4. Frontend Checklist

- [x] WebSocket connection với JWT token
- [x] Handle `project_progress` messages
- [x] Handle `project_completed` messages
- [x] Handle `dryrun_result` messages
- [x] Fallback to polling khi WebSocket fail
- [x] Progress bar UI component
- [x] Completion notification UI

---

## 8. Data Flow Diagrams

### 8.1. Project Execution Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PROJECT EXECUTION FLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Client: POST /projects/:id/execute                                      │
│       ↓                                                                     │
│  2. Project Service:                                                        │
│       - Verify ownership                                                    │
│       - Check duplicate execution                                           │
│       - Init Redis state: smap:proj:{id}                                    │
│       - Publish RabbitMQ: smap.events / project.created                     │
│       ↓                                                                     │
│  3. Collector Service:                                                      │
│       - Consume project.created event                                       │
│       - Extract user_id from payload                                        │
│       - Dispatch crawler workers                                            │
│       ↓                                                                     │
│  4. Collector Service (on progress):                                        │
│       - Update Redis: HINCRBY smap:proj:{id} done 1                         │
│       - POST /internal/progress/callback (throttled)                        │
│       ↓                                                                     │
│  5. Project Service:                                                        │
│       - Publish Redis Pub/Sub: user_noti:{user_id}                          │
│       ↓                                                                     │
│  6. WebSocket Service:                                                      │
│       - Subscribe user_noti:*                                               │
│       - Send to client                                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 8.2. Dry-Run Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DRY-RUN FLOW                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Client: POST /projects/dryrun                                           │
│       ↓                                                                     │
│  2. Project Service:                                                        │
│       - Generate job_id                                                     │
│       - Store job mapping: job:{job_id} → {user_id}                         │
│       - Publish RabbitMQ: collector.inbound / crawler.dryrun_keyword        │
│       ↓                                                                     │
│  3. Collector Service:                                                      │
│       - Consume dryrun task                                                 │
│       - Dispatch crawler workers (limit: 3 posts/keyword)                   │
│       ↓                                                                     │
│  4. Collector Service (on complete):                                        │
│       - POST /internal/dryrun/callback                                      │
│       ↓                                                                     │
│  5. Project Service:                                                        │
│       - Lookup user_id from job mapping                                     │
│       - Publish Redis Pub/Sub: user_noti:{user_id}                          │
│       ↓                                                                     │
│  6. WebSocket Service → Client                                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Configuration Reference

### 9.1. Project Service

```env
# Environment
ENV=production

# Server
APP_PORT=8080

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_DB=smap_project

# Redis
REDIS_HOST=localhost:6379
REDIS_PASSWORD=password
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# JWT (same as Identity Service)
JWT_SECRET=your-jwt-secret

# Cookie
COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true
COOKIE_NAME=smap_auth_token

# Internal API
INTERNAL_KEY=your-internal-key
```

### 9.2. Collector Service

```env
# Project Service
PROJECT_SERVICE_URL=http://project-service:8080
PROJECT_INTERNAL_KEY=your-internal-key

# Redis (same instance as Project Service)
REDIS_HOST=localhost:6379
REDIS_PASSWORD=password
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### 9.3. WebSocket Service

```env
# Redis (for Pub/Sub)
REDIS_HOST=localhost:6379
REDIS_PASSWORD=password

# JWT (same as Identity Service)
JWT_SECRET=your-jwt-secret
```

---

## 10. Testing & Troubleshooting

### 10.1. Integration Test Scenarios

#### Scenario 1: Happy Path - Project Execution

```bash
# 1. Create project
curl -X POST http://localhost:8080/projects \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"name": "Test", "brand_name": "Test", "brand_keywords": ["test"]}'

# 2. Execute project
curl -X POST http://localhost:8080/projects/{id}/execute \
  -b cookies.txt

# 3. Verify Redis state
redis-cli -n 1 HGETALL smap:proj:{id}

# 4. Check progress
curl http://localhost:8080/projects/{id}/progress \
  -b cookies.txt
```

#### Scenario 2: Duplicate Execution

```bash
# Execute twice - second should fail with 409
curl -X POST http://localhost:8080/projects/{id}/execute -b cookies.txt
curl -X POST http://localhost:8080/projects/{id}/execute -b cookies.txt
# Expected: {"error_code": 30008, "message": "project already executing"}
```

#### Scenario 3: Webhook Test

```bash
# Simulate Collector callback
curl -X POST http://localhost:8080/internal/progress/callback \
  -H "Content-Type: application/json" \
  -H "X-Internal-Key: your-internal-key" \
  -d '{
    "project_id": "uuid",
    "user_id": "uuid",
    "status": "CRAWLING",
    "total": 100,
    "done": 50,
    "errors": 2
  }'
```

### 10.2. Common Issues

| Issue                 | Cause                       | Solution                           |
| --------------------- | --------------------------- | ---------------------------------- |
| 401 Unauthorized      | Invalid/expired JWT         | Re-login via Identity Service      |
| 403 Forbidden         | User doesn't own project    | Check `created_by` field           |
| 409 Conflict          | Project already executing   | Wait for completion or check Redis |
| Webhook 401           | Invalid internal key        | Check `INTERNAL_KEY` config        |
| No WebSocket messages | Redis Pub/Sub not connected | Check Redis connection             |

### 10.3. Debug Commands

```bash
# Check Redis state
redis-cli -n 1 HGETALL smap:proj:{project_id}

# Monitor Redis Pub/Sub
redis-cli PSUBSCRIBE "user_noti:*"

# Check RabbitMQ queues
rabbitmqctl list_queues

# Check RabbitMQ bindings
rabbitmqctl list_bindings
```

---

## Appendix: Error Codes

| Code  | Message            | HTTP Status | Description            |
| ----- | ------------------ | ----------- | ---------------------- |
| 30001 | Internal error     | 500         | Server error           |
| 30002 | Wrong body         | 400         | Invalid request        |
| 30003 | Invalid ID         | 400         | Invalid UUID format    |
| 30004 | Project not found  | 404         | Project doesn't exist  |
| 30005 | Unauthorized       | 403         | No permission          |
| 30006 | Invalid status     | 400         | Invalid project status |
| 30007 | Invalid date range | 400         | to_date <= from_date   |
| 30008 | Already executing  | 409         | Duplicate execution    |

---

**Last Updated:** December 2025
**Version:** 2.0.0
**Maintained By:** SMAP Development Team
