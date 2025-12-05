# IMPLEMENTATION GUIDE: EVENT-DRIVEN CHOREOGRAPHY FOR SMAP

**Mục tiêu:** Thiết kế luồng dữ liệu tự hành (Autonomous Data Flow) từ Project → Collector → Analytics → Dashboard.

---

## 1. Thiết kế Hạ tầng Sự kiện (Event Infrastructure)

Để Choreography hoạt động, chúng ta cần một **Message Broker (RabbitMQ)** được cấu hình đúng chuẩn **Topic Exchange**.

### Cấu trúc Exchange & Routing Key

Chúng ta sẽ sử dụng 1 Exchange chính: `smap.events` (Type: `topic`).

| Routing Key         | Ý nghĩa                       | Producer (Người gửi)            | Consumers (Người nhận)         |
| :------------------ | :---------------------------- | :------------------------------ | :----------------------------- |
| `project.created`   | Có dự án mới cần chạy         | Project Service                 | Collector Service              |
| `data.collected`    | Dữ liệu thô đã nằm trên MinIO | Collector Service               | Analytics Service              |
| `analysis.finished` | Phân tích xong 1 bài          | Analytics Service               | Insight Service / Notification |
| `job.completed`     | Toàn bộ dự án đã xong         | Analytics Service (Logic Redis) | Notification Service           |

---

## 2. Quản lý Trạng thái Phân tán (Distributed State Management)

Vì không có ông Nhạc trưởng cầm sổ theo dõi, chúng ta dùng **Redis** làm "Bảng thông báo chung" - đóng vai trò "Trọng tài" (The Arbitrator) để đảm bảo các service rời rạc hiểu được bức tranh toàn cảnh.

### 2.1. Chiến lược chọn Database (Trong 16 DBs)

Để tránh việc cơ chế "dọn dẹp bộ nhớ" (Cache Eviction) của Redis vô tình xóa mất bộ đếm tiến độ, hãy tách biệt DB:

- **DB 0:** Dùng cho Cache (Session, API Response...).
- **DB 1:** Dùng riêng cho **SMAP State Management**.
  - _Lý do:_ Dữ liệu này cần sống dai cho đến khi Project xong. Nếu dùng chung DB 0, khi RAM đầy, Redis có thể xóa nhầm key tracking → Hệ thống mất khả năng theo dõi.

### 2.2. Thiết kế Data Structure & Key

Thay vì dùng nhiều key rời rạc (`proj:{id}:status`, `proj:{id}:total`...), sử dụng **Redis HASH** để gom nhóm dữ liệu của 1 Project vào 1 key duy nhất.

**Tại sao dùng Hash?**

- Gọn gàng hơn khi đọc/ghi
- Dễ set TTL (hạn sử dụng) cho toàn bộ project
- Atomic operations trên nhiều fields

**Cấu trúc:**

| Key              | Field    | Kiểu   | Mô tả                                            | Ai Ghi?                         |
| :--------------- | :------- | :----- | :----------------------------------------------- | :------------------------------ |
| `smap:proj:{id}` | `status` | String | `INITIALIZING`, `CRAWLING`, `PROCESSING`, `DONE` | Project / Collector / Analytics |
|                  | `total`  | Int    | Tổng số bài cần xử lý (VD: 1000)                 | Collector                       |
|                  | `done`   | Int    | Số bài đã xong (Atomic Counter)                  | Analytics                       |
|                  | `errors` | Int    | Số bài bị lỗi                                    | Analytics                       |

### 2.3. Kiến trúc Phân lớp cho Redis State Management

**QUAN TRỌNG: Áp dụng Clean Architecture cho Redis giống như PostgreSQL**

Redis state management phải tuân theo kiến trúc **4 lớp** để tách biệt rõ ràng:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Project UseCase Layer (Orchestration)                    │
│  internal/project/usecase/project.go                                        │
│  - Orchestrate flow: PostgreSQL → Redis State → RabbitMQ                    │
│  - Quyết định WHEN gọi state operations                                     │
│  - Gọi state.UseCase để thao tác state                                      │
└─────────────────┬───────────────────────────────────────────────────────────┘
                  │ depends on
                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    State UseCase Layer (Business Logic)                     │
│  internal/state/usecase/state.go                                            │
│  - Business logic: completion check (done >= total && total > 0)            │
│  - Status transitions: INITIALIZING → CRAWLING → PROCESSING → DONE          │
│  - Progress calculation: done/total * 100                                   │
│  - Duplicate completion prevention                                          │
└─────────────────┬───────────────────────────────────────────────────────────┘
                  │ depends on
                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│               State Repository Layer (Data Access ONLY)                     │
│  internal/state/repository/redis/state_repo.go                              │
│  - CHỈ chứa Redis CRUD operations                                           │
│  - Biết về key schema: smap:proj:{id}                                       │
│  - Biết về Hash fields: status, total, done, errors                         │
│  - KHÔNG chứa business logic (completion check, status transitions)         │
└─────────────────┬───────────────────────────────────────────────────────────┘
                  │ uses
                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                pkg/redis (Infrastructure Layer)                             │
│  pkg/redis/client.go                                                        │
│  - CHỈ chứa Redis connection logic                                          │
│  - Generic operations: HSet, HGet, HIncrBy, HGetAll, Pipeline, Expire       │
│  - KHÔNG biết về business domain (project, state, etc)                      │
│  - KHÔNG biết về key naming conventions                                     │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────┐
│                Domain Types (Model Layer)                                    │
│  internal/model/state.go                                                     │
│  - ProjectState struct (Status, Total, Done, Errors)                         │
│  - ProjectStatus constants (INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED)│
└──────────────────────────────────────────────────────────────────────────────┘
```

**Ví dụ Phân chia Trách nhiệm:**

❌ **SAI - Business logic trong Repository:**

```go
// internal/state/repository/redis/state_repo.go - KHÔNG NÊN LÀM NHƯ NÀY
func (r *redisStateRepository) IncrementDone(ctx context.Context, projectID string) (IncrementResult, error) {
    newDone := r.client.HIncrBy(ctx, key, "done", 1)
    total := r.client.HGet(ctx, key, "total")

    // ❌ Business logic trong Repository
    if newDone >= total {
        r.client.HSet(ctx, key, "status", "DONE")
        return IncrementResult{IsComplete: true}, nil
    }
    return IncrementResult{IsComplete: false}, nil
}
```

✅ **ĐÚNG - Business logic trong UseCase, Repository chỉ CRUD:**

```go
// internal/state/repository/redis/state_repo.go - Data Access ONLY
func (r *redisStateRepository) IncrementDone(ctx context.Context, projectID string) (int64, error) {
    key := buildKey(projectID)
    return r.client.HIncrBy(ctx, key, fieldDone, 1) // ✅ Chỉ trả về giá trị mới
}

func (r *redisStateRepository) GetState(ctx context.Context, projectID string) (*model.ProjectState, error) {
    key := buildKey(projectID)
    data, err := r.client.HGetAll(ctx, key)
    // Parse and return state...
}

func (r *redisStateRepository) SetStatus(ctx context.Context, projectID string, status model.ProjectStatus) error {
    key := buildKey(projectID)
    return r.client.HSet(ctx, key, fieldStatus, string(status))
}

// internal/state/usecase/state.go - Business Logic HERE
func (uc *stateUseCase) IncrementDone(ctx context.Context, projectID string) (state.IncrementResult, error) {
    // Step 1: Atomic increment (Repository)
    newDone, err := uc.repo.IncrementDone(ctx, projectID)
    if err != nil {
        return state.IncrementResult{}, err
    }

    // Step 2: Get state for completion check (Repository)
    currentState, err := uc.repo.GetState(ctx, projectID)
    if err != nil {
        return state.IncrementResult{NewDoneCount: newDone}, nil
    }

    // Step 3: Business logic - completion check (UseCase)
    isComplete := currentState.Total > 0 && newDone >= currentState.Total

    if isComplete && currentState.Status != model.ProjectStatusDone {
        uc.repo.SetStatus(ctx, projectID, model.ProjectStatusDone)
    } else if isComplete {
        isComplete = false // Already DONE, prevent duplicate
    }

    return state.IncrementResult{
        NewDoneCount: newDone,
        Total:        currentState.Total,
        IsComplete:   isComplete,
    }, nil
}
```

**Lợi ích của kiến trúc 4 lớp:**

1. **Separation of Concerns**: Mỗi layer có 1 trách nhiệm duy nhất
2. **Testability**: Mock Repository trong UseCase tests, không cần Redis thật
3. **Flexibility**: Đổi từ Redis sang DynamoDB chỉ cần viết Repository mới
4. **Maintainability**: Business logic tập trung ở UseCase, dễ đọc và sửa
5. **Consistency**: Giống pattern đã dùng cho PostgreSQL repositories

### 2.4. Cấu hình kết nối Redis

**Go Implementation (Project Service):**

```go
// pkg/redis/redis.go - Core connection logic only
func Connect(opts ClientOptions) (Client, error) {
    // Generic Redis client initialization
}

// cmd/api/main.go - Initialize connections
stateRedisClient, err := pkgRedis.Connect(pkgRedis.NewClientOptions().
    SetOptions(cfg.Redis).
    SetDB(1)) // DB 1 for state

// internal/state/repository/redis/state_repo.go - Use in repository
stateRepo := redis.NewStateRepository(stateRedisClient)
```

---

## 3. Chi tiết Luồng Phối hợp (The Choreographed Flow)

Dưới đây là kịch bản chi tiết cho một vòng đời dữ liệu.

### BƯỚC 1: KÍCH HOẠT (The Trigger)

- **Tại:** `Project Service`
- **Hành động:** User bấm "Create Project".
- **Logic:**
  1. Lưu thông tin dự án vào PostgreSQL (`projects` table).
  2. Khởi tạo trạng thái trên Redis với TTL để tránh rác hệ thống.
  3. **Publish Event:** `project.created`.

**Implementation (Go - Project Service):**

```go
// internal/state/repository/interface.go
type StateRepository interface {
    InitProjectState(ctx context.Context, projectID string) error
    UpdateStatus(ctx context.Context, projectID string, status string) error
    GetProjectState(ctx context.Context, projectID string) (*State, error)
    // ... other methods
}

// internal/state/repository/redis/state_repo.go
func (r *redisStateRepository) InitProjectState(ctx context.Context, projectID string) error {
    key := fmt.Sprintf("smap:proj:%s", projectID) // Repository knows domain schema

    // Use pipeline for atomic operations
    pipe := r.redis.Pipeline(ctx)
    pipe.HSet(ctx, key, "status", "INITIALIZING")
    pipe.HSet(ctx, key, "total", "0")
    pipe.HSet(ctx, key, "done", "0")
    pipe.HSet(ctx, key, "errors", "0")
    pipe.Expire(ctx, key, 604800) // 7 days TTL

    if err := pipe.Exec(ctx); err != nil {
        r.logger.Errorf(ctx, "state.InitProjectState: %v", err)
        return err
    }

    r.logger.Infof(ctx, "Project %s state initialized in Redis", projectID)
    return nil
}

// internal/project/usecase/project.go
func (u *projectUseCase) Create(ctx context.Context, input CreateInput) (*Output, error) {
    // 1. Save to PostgreSQL
    project, err := u.repo.Create(ctx, input)
    if err != nil {
        return nil, err
    }

    // 2. Initialize Redis state
    if err := u.stateRepo.InitProjectState(ctx, project.ID); err != nil {
        u.logger.Errorf(ctx, "Failed to init state: %v", err)
        // Continue - state can be initialized lazily
    }

    // 3. Publish event
    event := ToProjectCreatedEvent(project)
    if err := u.eventPublisher.PublishProjectCreated(ctx, event); err != nil {
        return nil, fmt.Errorf("event publishing failed: %w", err)
    }

    return ToOutput(project), nil
}
```

**Event Message:**

```json
// Topic: smap.events | Key: project.created
{
  "event_id": "evt_001",
  "timestamp": "2025-11-29T10:00:00Z",
  "payload": {
    "project_id": "proj_abc",
    "keywords": ["VinFast", "VF3"],
    "targets": [{ "platform": "tiktok", "url": "..." }],
    "date_range": ["2025-01-01", "2025-02-01"]
  }
}
```

### BƯỚC 2: THU THẬP & SẢN XUẤT (The Producer)

- **Tại:** `Collector Service` (Collector Manager)
- **Hành động:**
  1. Lắng nghe `project.created`.
  2. Phân rã thành các Job con. Ví dụ: Tìm thấy 1000 bài viết cần crawl.
  3. Cập nhật Redis với tổng số items.
  4. Điều phối Worker đi crawl.
  5. Worker crawl xong 1 bài → Upload MinIO → **Publish Event:** `data.collected`.

**Implementation (Python/Go - Collector Service):**

```python
# Collector Service uses same layered architecture

# state/repository/redis_state_repo.py - Repository layer
def set_project_total(project_id, total_items):
    key = f"smap:proj:{project_id}"  # Repository knows key schema

    # Update Redis state
    r.hset(key, "total", total_items)
    r.hset(key, "status", "CRAWLING")

    logger.info(f"Project {project_id} total items set to {total_items}")

# usecase/collector_manager.py - UseCase layer
def handle_project_created_event(event):
    project_id = event['payload']['project_id']

    # Business logic: Find all posts to crawl
    posts = find_posts_to_crawl(event['payload'])
    total_count = len(posts)

    # Update state via repository
    state_repo.set_project_total(project_id, total_count)

    # Dispatch workers
    for post in posts:
        dispatch_crawler_worker(post)
```

**Event Message:**

```json
// Topic: smap.events | Key: data.collected
{
  "event_id": "evt_002",
  "payload": {
    "project_id": "proj_abc",
    "platform": "TIKTOK",
    "minio_path": "raw/tiktok/vid_888.json",
    "crawled_at": "..."
  }
}
```

### BƯỚC 3: XỬ LÝ & KIỂM TRA ĐÍCH (The Processor) - **QUAN TRỌNG NHẤT**

- **Tại:** `Analytics Service`
- **Hành động:**
  1. Lắng nghe `data.collected`. (Queue này nên set `prefetch_count` để không bị quá tải).
  2. Tải JSON từ MinIO dựa trên `minio_path`.
  3. Chạy Pipeline (5 Modules AI).
  4. Lưu kết quả vào PostgreSQL (`post_analytics`).
  5. Cập nhật Redis và kiểm tra hoàn thành.
  6. Nếu hoàn thành → **Publish Event:** `job.completed`.

**Implementation (Atomic Finish Check) - Analytics Service:**

Đây là nơi logic phức tạp nhất. Cần tăng biến đếm `done` và kiểm tra xem đã xong chưa. Thao tác này phải **Atomic** (bất khả phân chia) để tránh Race Condition khi có 100 workers chạy cùng lúc.

```python
# state/repository/redis_state_repo.py - Repository layer
# Business logic ở đây, KHÔNG ở pkg/redis
def mark_item_done(project_id, is_error=False):
    key = f"smap:proj:{project_id}"  # Repository biết key schema

    # 1. Tăng biến đếm (Atomic Increment)
    # HINCRBY trả về giá trị MỚI sau khi cộng
    current_done = r.hincrby(key, "done", 1)  # pkg/redis cung cấp hincrby generic

    if is_error:
        r.hincrby(key, "errors", 1)

    # 2. Lấy tổng số (Total) để so sánh
    total_str = r.hget(key, "total")
    total = int(total_str) if total_str else 0

    # 3. Kiểm tra vạch đích (The Finish Line Check)
    if total > 0 and current_done >= total:
        # Double check để đảm bảo chỉ bắn event 1 lần duy nhất
        current_status = r.hget(key, "status")
        if current_status != "DONE":
            r.hset(key, "status", "DONE")
            return "COMPLETED"  # Báo hiệu để code bên ngoài bắn RabbitMQ Event

    return "PROCESSING"


# usecase/analytics_processor.py - UseCase layer
def process_post(post_data):
    project_id = post_data['project_id']

    # Run AI pipeline (5 modules)
    result = run_ai_pipeline(post_data)

    # Save to PostgreSQL
    save_analytics_result(result)

    # Update state via repository (business logic in repo)
    completion_status = state_repo.mark_item_done(project_id, is_error=False)

    # UseCase decides what to do with completion
    if completion_status == "COMPLETED":
        event_publisher.publish("smap.events", "job.completed", {
            "project_id": project_id
        })
        logger.info(f"Job finished! Event sent for project {project_id}")
```

### BƯỚC 4: HIỂN THỊ & THÔNG BÁO (The View)

- **Tại:** `Insight Service` (hoặc Notification Service)
- **Hành động:**
  - **Real-time:** Frontend gọi API polling vào Redis để hiện Progress Bar.
  - **Hoàn thành:** Notification Service nghe `job.completed` → Gửi Email/Zalo cho User.

**Implementation (Go - Project Service):**

```go
// internal/state/repository/redis/state_repo.go - Repository layer
func (r *redisStateRepository) GetProjectState(ctx context.Context, projectID string) (*State, error) {
    key := fmt.Sprintf("smap:proj:%s", projectID)  // Repository knows key schema

    // Get all Hash fields (generic operation from pkg/redis)
    data, err := r.redis.HGetAll(ctx, key)
    if err != nil {
        return nil, err
    }

    if len(data) == 0 {
        return nil, ErrStateNotFound
    }

    // Parse and convert to domain State object
    total, _ := strconv.Atoi(data["total"])
    done, _ := strconv.Atoi(data["done"])
    errors, _ := strconv.Atoi(data["errors"])

    return &State{
        Status: data["status"],
        Total:  total,
        Done:   done,
        Errors: errors,
    }, nil
}

// internal/project/usecase/project.go - UseCase layer
func (u *projectUseCase) GetProgress(ctx context.Context, projectID, userID string) (*ProgressOutput, error) {
    // 1. Authorization: Verify user owns project
    project, err := u.projectRepo.GetByID(ctx, projectID)
    if err != nil {
        return nil, err
    }
    if project.CreatedBy != userID {
        return nil, ErrUnauthorized
    }

    // 2. Get state from Redis via repository
    state, err := u.stateRepo.GetProjectState(ctx, projectID)
    if err != nil {
        // Fallback to PostgreSQL status if Redis state not found
        return &ProgressOutput{
            ProjectID:      projectID,
            Status:         project.Status,
            TotalItems:     0,
            ProcessedItems: 0,
            FailedItems:    0,
            ProgressPercent: 0.0,
            StateSource:    "postgresql",
        }, nil
    }

    // 3. Calculate progress percentage (business logic in UseCase)
    percent := 0.0
    if state.Total > 0 {
        percent = float64(state.Done) / float64(state.Total) * 100
    }

    return &ProgressOutput{
        ProjectID:       projectID,
        Status:          state.Status,
        TotalItems:      state.Total,
        ProcessedItems:  state.Done,
        FailedItems:     state.Errors,
        ProgressPercent: math.Round(percent*100) / 100, // Round to 2 decimals
        StateSource:     "redis",
    }, nil
}

// internal/project/delivery/http/handler.go - HTTP Handler
func (h *Handler) GetProgress(c *gin.Context) {
    projectID := c.Param("id")
    userID := c.GetString("user_id") // From JWT middleware

    progress, err := h.usecase.GetProgress(c.Request.Context(), projectID, userID)
    if err != nil {
        h.handleError(c, err)
        return
    }

    c.Header("Cache-Control", "no-cache")
    c.JSON(http.StatusOK, ToProgressResponse(progress))
}
```

---

## 4. Giải quyết các Bài toán Hóc búa (Advanced Scenarios)

### 4.1. Vấn đề "Con gà quả trứng" (Collector chậm hơn Analytics)

**Tình huống:** Collector mới tìm được 10 bài, update `total=10`. Analytics chạy nhanh quá, xử lý xong 10 bài → Redis thấy `done=10, total=10` → Bắn event `job.completed`. Nhưng thực tế Collector vẫn đang tìm tiếp và sau đó update `total=100`.

**Giải pháp:**

- Collector chỉ update trạng thái là `CRAWLING` khi đang tìm.
- Khi Collector tìm xong HẾT, nó update trạng thái sang `PROCESSING_WAIT`.
- Analytics chỉ bắn event `job.completed` khi: `done == total` **VÀ** `status != CRAWLING`.

### 4.2. Vấn đề Dead Letter (Bài lỗi)

**Quan trọng:** Dù bài viết bị lỗi (crash, file hỏng), bạn **vẫn phải gọi `mark_item_done(..., is_error=True)`**.

_Lý do:_ Nếu không tăng biến `done`, tổng số `done` sẽ mãi mãi nhỏ hơn `total` (ví dụ 999/1000) và hệ thống không bao giờ Finish được.

**Cơ chế xử lý:**

1. Analytics bắt lỗi (`try...except`).
2. **Ack** message đó (để xóa khỏi hàng đợi chính, tránh block các bài sau).
3. Gửi message đó vào **Dead Letter Queue (DLQ)**: `analytics.errors`.
4. Gọi `mark_item_done(project_id, is_error=True)` để Project vẫn có thể về đích 100%.

### 4.3. Xử lý trùng lặp (Idempotency)

**Tình huống:** RabbitMQ gửi 1 bài viết 2 lần.

**Cơ chế:**

- Analytics Service kiểm tra DB trước khi Insert.
- Sử dụng `INSERT ... ON CONFLICT DO UPDATE`.
- Redis `INCR` vẫn có thể bị tăng 2 lần. Chấp nhận sai số nhỏ này hoặc dùng `SET` trong Redis để lưu list ID đã làm (tốn RAM hơn).

### 4.4. Analytics Service bị sập (Crash)

**Hiện tượng:** Hàng nghìn message `data.collected` dồn ứ trong Queue RabbitMQ.

**Hậu quả:** Không sao cả. MinIO vẫn giữ file. Redis vẫn giữ số đếm cũ.

**Khắc phục:** Khi Analytics bật lại, nó tiếp tục consume message từ Queue và chạy tiếp. Không mất dữ liệu.

---

## 5. Tổng kết Kiến trúc

Với thiết kế này, hệ thống SMAP đạt được:

1. **DB 1 riêng biệt:** Đảm bảo dữ liệu tracking an toàn, không bị eviction.
2. **Redis Hash:** Cấu trúc dữ liệu gọn gàng, dễ quản lý TTL.
3. **Atomic Increment:** Đảm bảo chính xác 100% dù chạy đa luồng.
4. **Finish Check tại chỗ:** Logic kiểm tra đích ngay sau khi tăng biến đếm là cách hiệu quả nhất.
5. **High Throughput:** Collector cứ việc đẩy hàng nghìn bài vào kho mà không sợ Analytics bị nghẹn.
6. **Decoupling:** Team làm Crawler không cần quan tâm Team AI code gì, chỉ cần thống nhất format JSON.
7. **Visibility:** User vẫn thấy được thanh tiến độ chạy vù vù nhờ Redis, dù bên dưới là hàng chục service đang chạy tán loạn.

Đây chính là **Event-Driven Choreography** chuẩn mực cho hệ thống xử lý dữ liệu lớn.
