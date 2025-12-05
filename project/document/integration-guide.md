# SMAP Integration Guide

## Hướng Dẫn Tích Hợp Giữa Các Services (SMAP)

### Collector Service – Việc Cần Triển Khai

| Nhiệm Vụ                  | Độ Ưu Tiên | Mô Tả                                                               |
| ------------------------- | ---------- | ------------------------------------------------------------------- |
| Consume `project.created` | HIGH       | Đọc từ exchange `smap.events`, routing key `project.created`        |
| Extract `user_id`         | HIGH       | Lưu mapping `project_id` → `user_id` để gọi webhook tương ứng       |
| Update Redis state        | HIGH       | HINCRBY `smap:proj:{id}` `done` 1 (tăng tiến độ project trên Redis) |
| Call progress webhook     | HIGH       | Gọi `POST /internal/progress/callback` để cập nhật tiến độ          |
| Throttle notifications    | MEDIUM     | Không gọi webhook mỗi item, gộp/throttle tối thiểu 5 giây/callback  |

### WebSocket Service – Việc Cần Triển Khai

| Nhiệm Vụ                | Độ Ưu Tiên | Mô Tả                                                                            |
| ----------------------- | ---------- | -------------------------------------------------------------------------------- |
| Subscribe `user_noti:*` | HIGH       | Sử dụng Redis PSUBSCRIBE pattern trên các channel thông báo                      |
| Route to user           | HIGH       | Lấy `user_id` từ tên channel, gửi message về đúng user                           |
| Handle message types    | MEDIUM     | Xử lý các loại message: `project_progress`, `project_completed`, `dryrun_result` |

### Frontend – Việc Cần Triển Khai

| Nhiệm Vụ             | Độ Ưu Tiên | Mô Tả                                                                        |
| -------------------- | ---------- | ---------------------------------------------------------------------------- |
| WebSocket connection | HIGH       | Kết nối WebSocket, gửi kèm JWT token để xác thực                             |
| Handle messages      | HIGH       | Xử lý các message nhận về (switch theo `message.type`)                       |
| Progress bar         | MEDIUM     | Hiển thị tiến độ project theo message `project_progress`                     |
| Fallback polling     | LOW        | Khi WebSocket lỗi/thất bại, chuyển sang polling `GET /projects/:id/progress` |

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SMAP SYSTEM OVERVIEW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │   Frontend   │────▶│   Project    │────▶│  Collector   │                │
│  │   (React)    │◀────│   Service    │◀────│   Service    │                │
│  └──────────────┘     └──────────────┘     └──────────────┘                │
│         │                    │                    │                         │
│         │                    │                    │                         │
│         ▼                    ▼                    ▼                         │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │  WebSocket   │◀────│    Redis     │◀────│   Crawler    │                │
│  │   Service    │     │  (Pub/Sub)   │     │   Workers    │                │
│  └──────────────┘     └──────────────┘     └──────────────┘                │
│                              │                                              │
│                              ▼                                              │
│                       ┌──────────────┐                                      │
│                       │   RabbitMQ   │                                      │
│                       └──────────────┘                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Services Overview

| Service           | Port | Responsibility                                   |
| ----------------- | ---- | ------------------------------------------------ |
| Project Service   | 8080 | Project CRUD, event publishing, webhook handling |
| Collector Service | 8081 | Crawl orchestration, progress updates            |
| WebSocket Service | 8082 | Real-time notifications to clients               |
| Identity Service  | 8083 | Authentication, user management                  |

---

## Communication Patterns

### 1. RabbitMQ Events (Async)

- Project → Collector: `project.created`
- Project → Collector: `crawler.dryrun_keyword`

### 2. HTTP Webhooks (Sync)

- Collector → Project: `/internal/dryrun/callback`
- Collector → Project: `/internal/progress/callback`

### 3. Redis Pub/Sub (Real-time)

- Project → WebSocket: `user_noti:{user_id}`

---

## Flow 1: Project Execution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PROJECT EXECUTION FLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Client: POST /projects/:id/execute                                      │
│       ↓                                                                     │
│  2. Project Service:                                                        │
│       - Init Redis state: smap:proj:{id}                                   │
│       - Publish RabbitMQ: smap.events / project.created                    │
│       ↓                                                                     │
│  3. Collector Service:                                                      │
│       - Consume project.created event                                       │
│       - Extract user_id from event payload                                  │
│       - Dispatch crawler workers                                            │
│       ↓                                                                     │
│  4. Collector Service (on progress):                                        │
│       - Update Redis: HINCRBY smap:proj:{id} done 1                        │
│       - POST /internal/progress/callback                                    │
│       ↓                                                                     │
│  5. Project Service:                                                        │
│       - Publish Redis: user_noti:{user_id}                                 │
│       ↓                                                                     │
│  6. WebSocket Service:                                                      │
│       - Subscribe user_noti:*                                              │
│       - Send to client                                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Collector Service - Required Implementation

### 1. Consume `project.created` Event

**Exchange:** `smap.events`  
**Routing Key:** `project.created`  
**Queue:** `collector.project.created`

```python
# Python example
def handle_project_created(event):
    payload = event['payload']

    project_id = payload['project_id']
    user_id = payload['user_id']  # NEW: For progress notifications
    brand_keywords = payload['brand_keywords']
    competitor_keywords_map = payload['competitor_keywords_map']
    date_range = payload['date_range']

    # Store user_id for progress callbacks
    store_project_mapping(project_id, user_id)

    # Dispatch crawlers
    dispatch_crawlers(project_id, brand_keywords, competitor_keywords_map, date_range)
```

**Event Schema:**

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

### 2. Update Redis State

```python
def update_state(project_id, field, value):
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

### 3. Call Progress Webhook

```python
def notify_progress(project_id, user_id):
    # Get current state from Redis
    key = f"smap:proj:{project_id}"
    state = redis.hgetall(key)

    # Call Project Service webhook
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

### 4. When to Call Progress Webhook

| Event             | Action                                                          |
| ----------------- | --------------------------------------------------------------- |
| Found total items | `update_state(id, "total", count)` + `notify_progress()`        |
| Crawled 1 item    | `update_state(id, "done", 1)` + `notify_progress()` (throttle!) |
| Item failed       | `update_state(id, "errors", 1)` + `notify_progress()`           |
| All done          | `update_state(id, "status", "DONE")` + `notify_progress()`      |
| Fatal error       | `update_state(id, "status", "FAILED")` + `notify_progress()`    |

**Throttling Recommendation:**

```python
# Don't call webhook on every item - throttle to every 10 items or 5 seconds
last_notify_time = {}
THROTTLE_INTERVAL = 5  # seconds

def should_notify(project_id):
    now = time.time()
    last = last_notify_time.get(project_id, 0)
    if now - last > THROTTLE_INTERVAL:
        last_notify_time[project_id] = now
        return True
    return False
```

---

## Flow 2: Dry-Run Keywords

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DRY-RUN FLOW                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Client: POST /projects/dryrun                                           │
│       ↓                                                                     │
│  2. Project Service:                                                        │
│       - Store job mapping: job_id → user_id                                │
│       - Publish RabbitMQ: collector.inbound / crawler.dryrun_keyword       │
│       ↓                                                                     │
│  3. Collector Service:                                                      │
│       - Consume dryrun task                                                 │
│       - Dispatch crawler workers (limit: 3 posts/keyword)                  │
│       ↓                                                                     │
│  4. Collector Service (on complete):                                        │
│       - POST /internal/dryrun/callback                                      │
│       ↓                                                                     │
│  5. Project Service:                                                        │
│       - Lookup user_id from job_id                                         │
│       - Publish Redis: user_noti:{user_id}                                 │
│       ↓                                                                     │
│  6. WebSocket Service → Client                                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Collector: Dry-Run Callback

```python
def send_dryrun_callback(job_id, platform, status, content, errors):
    response = requests.post(
        f"{PROJECT_SERVICE_URL}/internal/dryrun/callback",
        headers={"X-Internal-Key": INTERNAL_KEY},
        json={
            "job_id": job_id,
            "status": status,  # "success" or "failed"
            "platform": platform,  # "youtube" or "tiktok"
            "payload": {
                "content": content,
                "errors": errors
            }
        }
    )
    return response.status_code == 200
```

---

## WebSocket Service - Required Implementation

### 1. Subscribe to Redis Pub/Sub

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

### 2. Handle Message Types

```javascript
// Client-side
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

---

## API Endpoints Summary

### Project Service

| Method | Endpoint                      | Auth           | Description            |
| ------ | ----------------------------- | -------------- | ---------------------- |
| POST   | `/projects`                   | Cookie         | Create project         |
| POST   | `/projects/:id/execute`       | Cookie         | Start execution        |
| GET    | `/projects/:id/progress`      | Cookie         | Get progress (polling) |
| POST   | `/projects/dryrun`            | Cookie         | Start dry-run          |
| POST   | `/internal/dryrun/callback`   | X-Internal-Key | Dry-run webhook        |
| POST   | `/internal/progress/callback` | X-Internal-Key | Progress webhook       |

### Collector Service (to implement)

| Method | Endpoint          | Auth | Description                      |
| ------ | ----------------- | ---- | -------------------------------- |
| -      | RabbitMQ consumer | -    | Consume `project.created`        |
| -      | RabbitMQ consumer | -    | Consume `crawler.dryrun_keyword` |

---

## Configuration

### Project Service

```env
# Redis
REDIS_HOST=localhost:6379
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Internal API
INTERNAL_KEY=your-internal-key
```

### Collector Service

```env
# Project Service
PROJECT_SERVICE_URL=http://localhost:8080
PROJECT_INTERNAL_KEY=your-internal-key

# Redis (same instance as Project Service)
REDIS_HOST=localhost:6379

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### WebSocket Service

```env
# Redis (for Pub/Sub)
REDIS_HOST=localhost:6379

# JWT (same secret as Identity Service)
JWT_SECRET=your-jwt-secret
```

---

## Testing Checklist

### Project Service ✅

- [x] Create project
- [x] Execute project (Redis + RabbitMQ)
- [x] Get progress (polling)
- [x] Dry-run keywords
- [x] Progress webhook
- [x] Dry-run webhook

### Collector Service (TODO)

- [ ] Consume `project.created` event
- [ ] Extract `user_id` from event
- [ ] Update Redis state
- [ ] Call progress webhook
- [ ] Consume `dryrun_keyword` task
- [ ] Call dry-run webhook

### WebSocket Service (TODO)

- [ ] Subscribe to `user_noti:*`
- [ ] Route messages to correct user
- [ ] Handle connection/disconnection

### Frontend (TODO)

- [ ] WebSocket connection
- [ ] Handle `project_progress` messages
- [ ] Handle `project_completed` messages
- [ ] Handle `dryrun_result` messages
- [ ] Fallback to polling API

---

## Error Handling

### Project Service Error Codes

| Code  | Message                   | HTTP Status |
| ----- | ------------------------- | ----------- |
| 30004 | Project not found         | 404         |
| 30005 | Unauthorized              | 403         |
| 30007 | Invalid date range        | 400         |
| 30008 | Project already executing | 409         |

### Webhook Error Handling

```python
# Collector should retry on failure
def call_webhook_with_retry(url, payload, max_retries=3):
    for attempt in range(max_retries):
        try:
            response = requests.post(url, json=payload, timeout=10)
            if response.status_code == 200:
                return True
        except Exception as e:
            logger.error(f"Webhook failed (attempt {attempt+1}): {e}")
        time.sleep(2 ** attempt)  # Exponential backoff
    return False
```
