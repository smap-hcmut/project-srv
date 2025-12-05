# Design Document: Event-Driven Choreography Alignment

## Architecture Overview

This design implements the event-driven choreography pattern for the SMAP Project Service, enabling autonomous data flow across microservices (Project → Collector → Analytics → Dashboard) without centralized orchestration.

### Key Architectural Principles

1. **Choreography over Orchestration**: Services react to events independently without central coordinator
2. **Distributed State Management**: Redis acts as "The Arbitrator" for cross-service state visibility
3. **Event Sourcing**: Project lifecycle changes trigger domain events
4. **Eventual Consistency**: Services converge to consistent state through event processing

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Project Service (This)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   HTTP API   │    │ RabbitMQ Pub │    │  Redis State │      │
│  │  (Create)    │───▶│  Producer    │    │  Manager     │      │
│  └──────────────┘    └──────┬───────┘    └──────┬───────┘      │
│                             │                     │              │
│                             ▼                     ▼              │
│                      ┌─────────────┐      ┌─────────────┐       │
│                      │  RabbitMQ   │      │   Redis DB  │       │
│                      │ smap.events │      │   (DB 1)    │       │
│                      │  Exchange   │      │  State Hash │       │
│                      └─────┬───────┘      └─────────────┘       │
└────────────────────────────┼─────────────────────────────────────┘
                             │
                             │ project.created
                             ▼
                    ┌─────────────────┐
                    │ Collector Service│
                    │  (Consumes)      │
                    └──────────────────┘
```

## Component Design

### 0. Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CLEAN ARCHITECTURE                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ internal/project/usecase/                                             │  │
│  │ ORCHESTRATION LAYER                                                   │  │
│  │ - Coordinates flow: PostgreSQL → Redis State → RabbitMQ               │  │
│  │ - Decides WHEN to init state, WHEN to publish events                  │  │
│  │ - Calls state.UseCase for state operations                            │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                    │                                         │
│                                    ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ internal/state/usecase/                                               │  │
│  │ BUSINESS LOGIC LAYER                                                  │  │
│  │ - Completion checks (done >= total && total > 0)                      │  │
│  │ - Status transitions (INITIALIZING → CRAWLING → PROCESSING → DONE)   │  │
│  │ - Progress calculation (done/total * 100)                             │  │
│  │ - Duplicate completion prevention                                     │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                    │                                         │
│                                    ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ internal/state/repository/                                            │  │
│  │ DATA ACCESS LAYER                                                     │  │
│  │ - Pure Redis CRUD operations                                          │  │
│  │ - Key schema: smap:proj:{projectID}                                   │  │
│  │ - TTL management (7 days)                                             │  │
│  │ - NO business logic                                                   │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                    │                                         │
│                                    ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ pkg/redis/                                                            │  │
│  │ INFRASTRUCTURE LAYER                                                  │  │
│  │ - Generic Redis operations (HSet, HGet, HIncrBy, Pipeline)            │  │
│  │ - Connection management                                               │  │
│  │ - NO domain knowledge                                                 │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ internal/model/state.go                                               │  │
│  │ DOMAIN TYPES                                                          │  │
│  │ - ProjectState struct (Status, Total, Done, Errors)                   │  │
│  │ - ProjectStatus constants (INITIALIZING, CRAWLING, etc.)              │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Principles:**
1. **Repository = Data Access Only**: No business logic, just Redis CRUD
2. **UseCase = Business Logic**: Completion checks, status transitions, calculations
3. **Orchestration = Coordination**: When to call what, in what order
4. **Domain Types in Model**: Follow project convention (`internal/model/`)

### 1. Redis State Management

#### 1.1 Database Separation Strategy

**Problem**: Redis cache eviction policies (LRU, LFU) can inadvertently delete state tracking keys when memory is full, causing loss of project progress.

**Solution**: Dedicated Redis DB for state management with different eviction policies.

- **DB 0**: Cache (sessions, API responses) - `volatile-lru` eviction
- **DB 1**: SMAP State - `noeviction` policy (fail writes when full, never evict)

**Configuration**:
```go
type RedisConfig struct {
    // Existing cache connection (DB 0)
    RedisAddr    []string
    RedisDB      int  // 0 for cache

    // New state connection (DB 1)
    StateDB      int  // 1 for state
}
```

#### 1.2 State Data Structure

**Design Choice**: Redis Hash over separate keys

**Options Considered**:
1. Separate keys: `proj:{id}:status`, `proj:{id}:total`, `proj:{id}:done`
   - ❌ Harder to manage TTL (need to set on each key)
   - ❌ Not atomic when reading multiple fields
   - ✅ Slightly faster for single field access

2. Redis Hash: `smap:proj:{id}` with fields {status, total, done, errors}
   - ✅ Single TTL for entire project state
   - ✅ Atomic multi-field operations with HGETALL
   - ✅ Cleaner key namespace
   - ❌ Slightly more overhead for single field

**Decision**: Use Redis Hash for better atomicity and TTL management.

**State Schema** (in `internal/model/state.go`):
```go
// ProjectStatus represents the current state of a project in the processing pipeline.
type ProjectStatus string

const (
    ProjectStatusInitializing ProjectStatus = "INITIALIZING"
    ProjectStatusCrawling     ProjectStatus = "CRAWLING"
    ProjectStatusProcessing   ProjectStatus = "PROCESSING"
    ProjectStatusDone         ProjectStatus = "DONE"
    ProjectStatusFailed       ProjectStatus = "FAILED"
)

// ProjectState represents the current processing state of a project.
type ProjectState struct {
    Status ProjectStatus `json:"status"`
    Total  int64         `json:"total"`   // Total items to process (set by Collector)
    Done   int64         `json:"done"`    // Completed items (atomic counter)
    Errors int64         `json:"errors"`  // Failed items (atomic counter)
}
```

**Key Pattern**: `smap:proj:{project_id}`

**TTL Strategy**: 7 days (604800 seconds) to handle long-running projects while preventing indefinite memory growth.

#### 1.3 Atomic Operations Design

**Critical Operation**: Increment `done` counter and check completion atomically.

**Challenge**: In event-driven choreography, Analytics Service workers concurrently increment `done`. Race conditions can cause:
- Multiple `job.completed` events
- Missed completion detection (done=999/1000 forever)

**Solution Pattern** (Clean Architecture - Business Logic in UseCase):

**Repository Layer** (`internal/state/repository/redis/state_repo.go`):
```go
// IncrementDone atomically increments the done counter.
// Returns the new done count after increment.
// NO completion check here - that's UseCase responsibility.
func (r *redisStateRepository) IncrementDone(ctx context.Context, projectID string) (int64, error) {
    key := buildKey(projectID)
    return r.client.HIncrBy(ctx, key, fieldDone, 1)
}
```

**UseCase Layer** (`internal/state/usecase/state.go`):
```go
// IncrementDone increments the done counter and checks for completion.
// Business logic: if done >= total && total > 0, marks status as DONE.
func (uc *stateUseCase) IncrementDone(ctx context.Context, projectID string) (state.IncrementResult, error) {
    // Step 1: Atomically increment done counter (repository layer)
    newDone, err := uc.repo.IncrementDone(ctx, projectID)
    if err != nil {
        return state.IncrementResult{}, err
    }

    // Step 2: Get current state to check completion (business logic)
    currentState, err := uc.repo.GetState(ctx, projectID)
    if err != nil {
        return state.IncrementResult{NewDoneCount: newDone, IsComplete: false}, nil
    }

    // Step 3: Business logic - check completion
    isComplete := currentState.Total > 0 && newDone >= currentState.Total

    if isComplete {
        // Only mark as DONE if not already DONE (prevents duplicate completion events)
        if currentState.Status != model.ProjectStatusDone {
            uc.repo.SetStatus(ctx, projectID, model.ProjectStatusDone)
        } else {
            isComplete = false // Already DONE, don't trigger completion again
        }
    }

    return state.IncrementResult{
        NewDoneCount: newDone,
        Total:        currentState.Total,
        IsComplete:   isComplete,
    }, nil
}
```

**Why This Works**:
- `HINCRBY` is atomic and returns post-increment value (Repository)
- Completion check is business logic (UseCase)
- Status check prevents duplicate completion events (UseCase)
- Clear separation of concerns

### 2. Event Publishing Architecture

#### 2.1 Exchange Design

**Current State**: Uses `collector.inbound` topic exchange for dry-run events.

**Target State**: Standardized `smap.events` topic exchange for all SMAP events.

**Exchange Configuration**:
```go
const SMAPEventsExchangeName = "smap.events"

var SMAPEventsExchange = rabbitmq.ExchangeArgs{
    Name:       "smap.events",
    Type:       "topic",
    Durable:    true,
    AutoDelete: false,
}
```

**Routing Keys**:
| Event | Routing Key | Producer | Consumers |
|-------|-------------|----------|-----------|
| Project Created | `project.created` | Project Service | Collector Service |
| Data Collected | `data.collected` | Collector Service | Analytics Service |
| Analysis Finished | `analysis.finished` | Analytics Service | Insight Service |
| Job Completed | `job.completed` | Analytics Service | Notification Service |

#### 2.2 Event Message Schema

**`project.created` Event**:
```json
{
  "event_id": "evt_uuid_v4",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "proj_uuid",
    "brand_name": "VinFast",
    "brand_keywords": ["VinFast", "VF3", "VF5"],
    "competitor_names": ["Tesla", "BYD"],
    "competitor_keywords_map": {
      "Tesla": ["Model 3", "Model Y"],
      "BYD": ["Seagull", "Dolphin"]
    },
    "date_range": {
      "from": "2025-01-01",
      "to": "2025-02-01"
    }
  }
}
```

**Design Rationale**:
- `event_id`: Enables event deduplication and tracing
- `timestamp`: Event ordering and audit trail
- `payload`: Contains all information Collector needs to start crawling

#### 2.3 Migration Strategy

**Problem**: Existing dry-run events use `collector.inbound`. Immediate cutover risks breaking Collector Service.

**Solution**: Dual publishing with feature flag.

**Implementation**:
```go
type PublisherConfig struct {
    UseLegacyExchange bool  // Feature flag
}

func (p *Producer) PublishProjectCreated(ctx context.Context, event ProjectCreatedEvent) error {
    // Publish to new standardized exchange
    if err := p.publishTo(ctx, "smap.events", "project.created", event); err != nil {
        return err
    }

    // Also publish to legacy exchange during transition
    if p.config.UseLegacyExchange {
        p.publishTo(ctx, "collector.inbound", "project.created", event)
    }

    return nil
}
```

**Migration Timeline**:
1. Week 1-2: Dual publishing enabled, Collector Service updated to consume from `smap.events`
2. Week 3: Monitor metrics, verify no messages on `collector.inbound`
3. Week 4: Disable `UseLegacyExchange` flag, deprecate `collector.inbound`

### 3. Project Lifecycle State Machine

```
┌──────────────┐
│ PostgreSQL   │  User creates project
│ Insert       │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ Redis Init       │  status: INITIALIZING
│ smap:proj:{id}   │  total: 0, done: 0, errors: 0
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ RabbitMQ Publish │  project.created → smap.events
└──────┬───────────┘
       │
       │  Collector Service consumes event
       ▼
┌──────────────────┐
│ Collector Updates│  status: CRAWLING
│ total: 1000      │  (found 1000 posts to crawl)
└──────┬───────────┘
       │
       │  Collector publishes data.collected events
       ▼
┌──────────────────┐
│ Analytics Updates│  done++, check if done == total
│ Atomic Increment │
└──────┬───────────┘
       │
       ▼
    done == total?
       │
       ├── No  → Continue processing
       │
       └── Yes → status: DONE
                 ▼
          ┌──────────────────┐
          │ Analytics Publish│  job.completed → smap.events
          └──────────────────┘
```

### 4. Progress Monitoring API

**Endpoint**: `GET /projects/{id}/progress`

**Design Considerations**:
1. **Data Source**: Redis state (fast, real-time) vs PostgreSQL (slower, authoritative)
   - **Decision**: Redis for real-time progress, fallback to PostgreSQL if key missing

2. **Cache Strategy**: No caching needed (Redis IS the cache layer, <10ms reads)

3. **Error Handling**: If Redis unavailable, return project status from PostgreSQL with estimated progress

**Response Schema**:
```json
{
  "project_id": "proj_uuid",
  "status": "PROCESSING",
  "total_items": 1000,
  "processed_items": 750,
  "failed_items": 5,
  "progress_percent": 75.0,
  "estimated_completion": "2025-12-05T15:30:00Z"
}
```

**Performance Target**: <100ms response time (Redis HGETALL + minimal computation)

## Error Handling & Resilience

### 1. Event Publishing Failures

**Scenario**: RabbitMQ unavailable during project creation.

**Options**:
1. **Rollback Transaction**: Delete project from PostgreSQL
   - ✅ Strong consistency
   - ❌ Reduced availability
   - ❌ Complex rollback logic (Redis + PostgreSQL)

2. **Store-and-Forward**: Save event to PostgreSQL "outbox" table, retry asynchronously
   - ✅ High availability
   - ✅ Eventual consistency
   - ❌ Additional complexity (outbox processor)

3. **Fail-Fast**: Return error to user, let them retry
   - ✅ Simple implementation
   - ❌ Poor user experience
   - ✅ Honest failure reporting

**Decision**: Option 3 for initial implementation (simplicity), Option 2 for future enhancement.

**Implementation**:
```go
func (u *projectUseCase) Create(ctx context.Context, input CreateInput) (*Output, error) {
    // 1. Save to PostgreSQL
    project, err := u.repo.Create(ctx, input)
    if err != nil {
        return nil, err
    }

    // 2. Initialize Redis state
    if err := u.stateManager.InitProject(ctx, project.ID); err != nil {
        u.logger.Errorf(ctx, "Failed to init Redis state: %v", err)
        // Continue - state can be initialized lazily
    }

    // 3. Publish event
    if err := u.eventPublisher.PublishProjectCreated(ctx, project); err != nil {
        u.logger.Errorf(ctx, "Failed to publish project.created: %v", err)
        return nil, fmt.Errorf("project created but event publishing failed: %w", err)
    }

    return project, nil
}
```

### 2. Redis State Initialization Failures

**Strategy**: Continue operation, log error, initialize lazily when Collector queries state.

**Rationale**: State is supplementary to source-of-truth (PostgreSQL). Missing state causes progress tracking failure but not data loss.

### 3. State Synchronization Issues

**Problem**: Redis state diverges from PostgreSQL (e.g., Redis key expired but project still active).

**Solution**: State Reconciliation API (admin-only):
```
POST /admin/projects/{id}/reconcile-state
```

Rebuilds Redis state from PostgreSQL + query Collector/Analytics for current counts.

## Testing Strategy

### 1. Unit Tests
- State manager operations (init, update, mark complete)
- Event message serialization
- Atomic increment logic

### 2. Integration Tests
- End-to-end project creation flow (PostgreSQL → Redis → RabbitMQ)
- Progress API with various state conditions
- State TTL expiration and cleanup

### 3. Load Tests
- Concurrent project creation (100 req/s)
- Redis connection pool under load
- RabbitMQ throughput (1000 events/s)

## Security Considerations

### 1. Redis State Access Control
- State DB (DB 1) should NOT be directly accessible from frontend
- Only backend services query state
- No sensitive data in Redis (user IDs are UUIDs from JWT)

### 2. Event Message Security
- No passwords or API keys in event payloads
- User-generated content (keywords, brand names) sanitized before publishing
- Event schema validation to prevent injection attacks

## Monitoring & Observability

### Metrics to Track
1. **Event Publishing**:
   - `project_created_events_published_total` (counter)
   - `project_created_events_failed_total` (counter)
   - `event_publish_duration_seconds` (histogram)

2. **State Management**:
   - `redis_state_operations_total` (counter, by operation type)
   - `redis_state_operation_duration_seconds` (histogram)
   - `redis_state_keys_total` (gauge)

3. **Progress API**:
   - `progress_api_requests_total` (counter)
   - `progress_api_latency_seconds` (histogram)
   - `progress_api_cache_hit_ratio` (gauge)

### Logging
- Log all event publications at INFO level
- Log Redis failures at ERROR level
- Include correlation IDs (project_id, event_id) in all logs

## Future Enhancements

1. **Outbox Pattern**: Transactional event publishing with PostgreSQL outbox table
2. **Event Replay**: Store events in event store for debugging and replay
3. **Dead Letter Queue**: Handle failed event processing with retry logic
4. **State Snapshots**: Periodic backups of Redis state to PostgreSQL
5. **Multi-Project Dashboards**: Aggregate progress across user's projects
6. **WebSocket Progress Updates**: Real-time push notifications instead of polling

## References

- Event-Driven Choreography Guide: `event-drivent.md`
- Redis Best Practices: https://redis.io/docs/manual/patterns/
- RabbitMQ Topic Exchanges: https://www.rabbitmq.com/tutorials/tutorial-five-go.html
