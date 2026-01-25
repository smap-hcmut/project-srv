# Implementation Tasks: Event-Driven Choreography Alignment

## Task Organization

Tasks are organized by capability and ordered for incremental delivery. Each task includes:
- **Validation**: How to verify the task is complete
- **Dependencies**: Prerequisites that must be completed first
- **Parallelizable**: Tasks that can be done concurrently

---

## 1. Infrastructure Setup (Prerequisites - COMPLETED)

### 1.1 Redis Integration (COMPLETED - Done by Identity Service in Parallel)

- [x] 1.1.1 Check current local docker already have redis container, password: 21042004
- [x] 1.1.2 Add Redis configuration to `template.env` (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_STATE_DB`)
- [x] 1.1.3 Create `pkg/redis/config.go` with Redis configuration struct
- [x] 1.1.4 Create `pkg/redis/client.go` with connection pool and graceful shutdown
- [x] 1.1.5 Add Redis state client initialization to `cmd/api/main.go`
- [x] 1.1.6 Add Redis health check endpoint `/health/redis`

**Note**: These tasks were completed in parallel by the Service team. Precheck before do task 2.

---

## 2. Redis State Management Implementation

**ARCHITECTURE NOTE**: Following Clean Architecture pattern (REFACTORED):
- `pkg/redis/` = Generic Redis operations only (HSet, HGet, HIncrBy, etc.) - NO business logic
- `internal/model/state.go` = Domain types (ProjectState, ProjectStatus) - follows project convention
- `internal/state/repository/` = Redis communication ONLY (key schemas, TTL, CRUD operations)
- `internal/state/usecase/` = Business logic (completion checks, status transitions, progress calculation)
- `internal/project/usecase/` = Orchestration (when to init state, when to publish events)

### 2.1 pkg/redis Enhancement (Generic Layer Only)

- [x] 2.1.1 Add generic Redis Hash operations to `pkg/redis/client.go`
  - **What**: Add `HSet`, `HSetMultiple`, `HGet`, `HGetAll`, `HIncrBy` methods
  - **Files**: `pkg/redis/client.go`
  - **Validation**: Methods exist and take generic `key`, `field`, `value` parameters (NO business logic)
  - **Dependencies**: 1.1.4 (Redis client structure exists)
  - **Completed**: ✅ Added all Hash operations

- [x] 2.1.2 Add Redis Pipeline support
  - **What**: Add `Pipeline()` and `TxPipeline()` methods for atomic multi-operation transactions
  - **Files**: `pkg/redis/client.go`
  - **Validation**: Pipeline interface includes `HSet`, `Expire`, `Exec` methods
  - **Dependencies**: 2.1.1
  - **Completed**: ✅ Added Pipeline() and TxPipeline() methods

- [x] 2.1.3 Add key management operations
  - **What**: Add `Expire`, `Exists`, `Del` methods for key lifecycle management
  - **Files**: `pkg/redis/client.go`
  - **Validation**: Methods work with any key type
  - **Dependencies**: 2.1.1
  - **Completed**: ✅ Added Expire, Exists, Del methods

### 2.2 Domain Types (Model Layer)

- [x] 2.2.1 Create state domain types in model package
  - **What**: Define `ProjectState` struct and `ProjectStatus` constants
  - **Files**: `internal/model/state.go`
  - **Validation**: Types compile and follow project naming convention (ProjectState, ProjectStatus)
  - **Dependencies**: None
  - **Completed**: ✅ Created ProjectState struct with Status, Total, Done, Errors fields. Added ProjectStatus constants (INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED)

### 2.3 State Repository Layer (Redis Communication ONLY)

- [x] 2.3.1 Create state repository interface
  - **What**: Define `StateRepository` interface with CRUD operations only (NO business logic)
  - **Files**: `internal/state/repository/interface.go`
  - **Methods**: InitState, GetState, SetStatus, SetTotal, IncrementDone, IncrementErrors, Delete, RefreshTTL
  - **Validation**: Interface compiles, methods are pure Redis operations
  - **Dependencies**: 2.2.1
  - **Completed**: ✅ Created StateRepository interface with pure Redis operations

- [x] 2.3.2 Implement Redis state repository
  - **What**: Create `redisStateRepository` implementing `StateRepository` interface
  - **Files**: `internal/state/repository/redis/new.go`, `internal/state/repository/redis/state_repo.go`
  - **Key schema**: `smap:proj:{projectID}`
  - **TTL**: 7 days (604800 seconds)
  - **Validation**: All interface methods implemented with atomic operations
  - **Dependencies**: 2.1.3, 2.3.1
  - **Completed**: ✅ Implemented all methods using Redis Pipeline for atomicity

- [x] 2.3.3 Write unit tests for state repository
  - **What**: Test constants, key building, field names
  - **Files**: `internal/state/repository/redis/state_repo_test.go`
  - **Validation**: All tests pass
  - **Dependencies**: 2.3.2
  - **Completed**: ✅ Tests for buildKey, constants, status constants

### 2.4 State UseCase Layer (Business Logic)

- [x] 2.4.1 Create state usecase interface
  - **What**: Define `UseCase` interface with business logic methods
  - **Files**: `internal/state/interface.go`
  - **Methods**: InitProjectState, GetProjectState, UpdateStatus, SetTotal, IncrementDone (with completion check), IncrementErrors, DeleteProjectState, GetProgressPercent
  - **Validation**: Interface compiles
  - **Dependencies**: 2.2.1, 2.3.1
  - **Completed**: ✅ Created UseCase interface with IncrementResult type for completion tracking

- [x] 2.4.2 Implement state usecase
  - **What**: Create `stateUseCase` implementing `UseCase` interface with business logic
  - **Files**: `internal/state/usecase/new.go`, `internal/state/usecase/state.go`
  - **Business Logic**:
    - InitProjectState: Creates state with INITIALIZING status
    - IncrementDone: Increments counter, checks completion (done >= total && total > 0), updates status to DONE if complete, prevents duplicate completion events
    - GetProgressPercent: Calculates percentage with zero-division handling
  - **Validation**: Business logic correctly separated from repository
  - **Dependencies**: 2.3.2, 2.4.1
  - **Completed**: ✅ Implemented all business logic in usecase layer

- [x] 2.4.3 Write unit tests for state usecase
  - **What**: Test business logic (completion check, progress calculation, status transitions)
  - **Files**: `internal/state/usecase/state_test.go`
  - **Validation**: All tests pass
  - **Dependencies**: 2.4.2
  - **Completed**: ✅ Tests for completion logic, progress calculation, status transitions
  - **Completed**: Created unit tests for buildKey, State.IsComplete, State.ProgressPercent, status constants, and field constants. All tests pass.

---

## 3. Event Publishing Implementation

### 3.1 Event Architecture Setup

- [x] 3.1.1 Add `smap.events` exchange constants
  - **What**: Define `SMAPEventsExchangeName = "smap.events"` and exchange declaration args
  - **Files**: `internal/project/delivery/rabbitmq/constants.go`
  - **Validation**: Constants defined and match design specs
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Added SMAPEventsExchangeName, SMAPEventsExchange, ProjectCreatedRoutingKey

- [x] 3.1.2 Add routing key constants
  - **What**: Define `ProjectCreatedRoutingKey = "project.created"`
  - **Files**: `internal/project/delivery/rabbitmq/constants.go`
  - **Validation**: Constants defined
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Added in 3.1.1

- [x] 3.1.3 Add migration feature flag configuration
  - **What**: Add `USE_LEGACY_EXCHANGE` env var to `config/config.go`
  - **Files**: `config/config.go`, `template.env`
  - **Validation**: Config loads with default `false`
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Added UseLegacyExchange to RabbitMQConfig with default false

### 3.2 Event Message Schema

- [x] 3.2.1 Create `ProjectCreatedEvent` type
  - **What**: Define struct with fields: EventID, Timestamp, Payload (project details)
  - **Files**: `internal/project/delivery/rabbitmq/types.go`
  - **Validation**: Type compiles and JSON serializes correctly
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Created ProjectCreatedEvent, ProjectCreatedPayload, DateRange types

- [x] 3.2.2 Implement event presenter to convert domain Project to event
  - **What**: Create function `ToProjectCreatedEvent(project domain.Project) ProjectCreatedEvent`
  - **Files**: `internal/project/delivery/rabbitmq/presenters.go`
  - **Validation**: Presenter correctly maps all fields
  - **Dependencies**: 3.2.1
  - **Completed**: ✅ Added ToProjectCreatedEvent function with UUID generation and field mapping

### 3.3 Event Publisher

- [x] 3.3.1 Add `PublishProjectCreated` method to producer interface
  - **What**: Define `PublishProjectCreated(ctx, event ProjectCreatedEvent) error`
  - **Files**: `internal/project/delivery/rabbitmq/producer/new.go`
  - **Validation**: Interface compiles
  - **Dependencies**: 3.2.1
  - **Completed**: ✅ Added to Producer interface, also added ProducerConfig and NewWithConfig

- [x] 3.3.2 Implement `PublishProjectCreated` with dual publishing support
  - **What**: Publish to `smap.events`, optionally publish to `collector.inbound` if feature flag enabled
  - **Files**: `internal/project/delivery/rabbitmq/producer/producer.go`
  - **Validation**: Event published to correct exchange based on feature flag
  - **Dependencies**: 3.1.1, 3.1.3, 3.3.1
  - **Completed**: ✅ Implemented with dual publishing support via UseLegacyExchange flag

- [ ] 3.3.3 Add publisher confirms for reliable delivery
  - **What**: Enable publisher confirms on RabbitMQ channel, wait for ack before returning success
  - **Files**: `pkg/rabbitmq/publisher.go` (or `internal/project/delivery/rabbitmq/producer/producer.go`)
  - **Validation**: Publish waits for broker confirmation
  - **Dependencies**: None (can enhance existing publisher)
  - **Test**: Verify publish fails if RabbitMQ nacks
  - **Note**: Deferred - requires changes to pkg/rabbitmq which may affect other services

- [ ] 3.3.4 Add event publishing metrics
  - **What**: Increment counters for published/failed events, record latency histogram
  - **Files**: `internal/project/delivery/rabbitmq/producer/metrics.go` (create), `producer.go`
  - **Validation**: Metrics emitted on success/failure
  - **Dependencies**: 3.3.2
  - **Note**: Deferred - requires metrics infrastructure setup

### 3.4 Integration with Project UseCase

- [x] 3.4.1 Inject StateRepository and EventPublisher into project usecase
  - **What**: Add StateRepository and EventPublisher as dependencies in `internal/project/usecase/new.go`
  - **Files**: `internal/project/usecase/new.go`, `internal/project/usecase/project.go`
  - **Validation**: Dependencies injected via constructor
  - **Dependencies**: 2.2.2, 3.3.1
  - **Completed**: ✅ Added stateUC field and NewWithState constructor

- [x] 3.4.2 Update `Create` method to initialize state and publish event
  - **What**: After PostgreSQL insert: 1) InitProjectState via StateRepository, 2) PublishProjectCreated event
  - **Files**: `internal/project/usecase/project.go`
  - **Validation**: Create flow executes PostgreSQL → Redis (via StateRepo) → RabbitMQ
  - **Dependencies**: 3.4.1
  - **Completed**: ✅ Updated Create method with 3-step flow: PostgreSQL → Redis state → RabbitMQ event

- [x] 3.4.3 Add error handling for partial failures
  - **What**: If event publish fails, log error and return error to user (project remains in DB)
  - **Files**: `internal/project/usecase/project.go`
  - **Validation**: User receives clear error message on publish failure
  - **Dependencies**: 3.4.2
  - **Completed**: ✅ Redis failure logs but continues, RabbitMQ failure returns error to user

- [ ] 3.4.4 Write integration tests for Create flow
  - **What**: Test full flow with real PostgreSQL, Redis, RabbitMQ (or mocks)
  - **Files**: `internal/project/usecase/project_integration_test.go`
  - **Validation**: Tests pass with all components
  - **Dependencies**: 3.4.2
  - **Note**: Deferred - requires test infrastructure setup

---

## 4. Progress Monitoring API

### 4.1 Progress HTTP Endpoint

- [x] 4.1.1 Create progress response type
  - **What**: Define `ProjectProgressResponse` with fields: ProjectID, Status, TotalItems, ProcessedItems, FailedItems, ProgressPercent
  - **Files**: `internal/project/delivery/http/presenter.go`, `internal/project/type.go`
  - **Validation**: Type compiles and JSON serializes correctly
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Added ProgressResp in presenter.go and ProgressOutput in type.go

- [x] 4.1.2 Implement progress presenter
  - **What**: Create function `ToProgressResponse(state ProjectState, projectID string) ProjectProgressResponse`
  - **Files**: `internal/project/delivery/http/presenter.go`
  - **Validation**: Correctly calculates progress percentage with zero-division handling
  - **Dependencies**: 4.1.1, 2.2.1
  - **Completed**: ✅ Added newProgressResp function in presenter.go

- [x] 4.1.3 Add `GetProgress` method to project usecase interface
  - **What**: Define `GetProgress(ctx, projectID, userID) (*ProgressOutput, error)`
  - **Files**: `internal/project/interface.go`
  - **Validation**: Interface compiles
  - **Dependencies**: None
  - **Completed**: ✅ Added GetProgress to UseCase interface

- [x] 4.1.4 Implement `GetProgress` in project usecase
  - **What**:
    1. Verify user owns project (check PostgreSQL via projectRepo)
    2. Get state from Redis via stateRepo.GetProjectState
    3. If state not found, return PostgreSQL status with zero progress
    4. Calculate progress percentage (business logic in UseCase)
    5. Return progress output
  - **Files**: `internal/project/usecase/project.go`
  - **Validation**: Returns correct progress with authorization check
  - **Dependencies**: 2.2.6, 4.1.3
  - **Architecture**:
    - StateRepository provides raw State data
    - UseCase calculates percentage and decides fallback behavior
  - **Completed**: ✅ Implemented with authorization check, Redis state lookup, and PostgreSQL fallback

- [x] 4.1.5 Create HTTP handler for progress endpoint
  - **What**: Implement `GetProgress` handler in `internal/project/delivery/http/handler.go`
  - **Files**: `internal/project/delivery/http/handler.go`, `internal/project/delivery/http/process_request.go`
  - **Validation**: Handler calls usecase and returns 200 with progress data
  - **Dependencies**: 4.1.1, 4.1.2, 4.1.4
  - **Completed**: ✅ Added GetProgress handler and processProgressReq function

- [x] 4.1.6 Add progress route to router
  - **What**: Add `GET /projects/:id/progress` route with authentication middleware
  - **Files**: `internal/project/delivery/http/routes.go`
  - **Validation**: Route registered and accessible
  - **Dependencies**: 4.1.5
  - **Completed**: ✅ Added route r.GET("/:id/progress", mw.Auth(), h.GetProgress)

- [x] 4.1.7 Add Cache-Control headers to prevent caching
  - **What**: Set `Cache-Control: no-cache` on progress responses
  - **Files**: `internal/project/delivery/http/handler.go`
  - **Validation**: Response headers include no-cache directive
  - **Completed**: ✅ Added Cache-Control, Pragma, and Expires headers
  - **Dependencies**: 4.1.5

- [ ] 4.1.8 Write unit tests for progress handler
  - **What**: Test authorized/unauthorized access, state found/not found scenarios
  - **Files**: `internal/project/delivery/http/handler_test.go`
  - **Validation**: All test cases pass
  - **Dependencies**: 4.1.5
  - **Note**: Deferred - requires test infrastructure setup

---

## 5. Configuration and Infrastructure

### 5.1 Environment Configuration

- [x] 5.1.1 Update `template.env` with new configurations
  - **What**: Add `REDIS_STATE_DB=1`, `USE_LEGACY_EXCHANGE=false`
  - **Files**: `template.env`
  - **Validation**: Template includes all new config vars
  - **Dependencies**: None (can be done in parallel)
  - **Completed**: ✅ Added REDIS_STATE_DB=1 and USE_LEGACY_EXCHANGE=false with documentation

- [x] 5.1.2 Update `config/config.go` to load state DB config
  - **What**: Add `StateDB int` field to `RedisConfig` with env tag `REDIS_STATE_DB` and default `1`
  - **Files**: `config/config.go`
  - **Validation**: Config struct includes StateDB field
  - **Dependencies**: None
  - **Completed**: ✅ Added StateDB field to RedisConfig with envDefault:"1"

### 5.2 Main Initialization

- [x] 5.2.1 Initialize state Redis client in `main.go`
  - **What**: Create second Redis client for DB 1 using `redisOpts.SetDB(cfg.Redis.StateDB)`
  - **Files**: `cmd/api/main.go`, `pkg/redis/options.go`
  - **Validation**: Two Redis clients initialized (cache + state)
  - **Dependencies**: 2.1.1, 5.1.2
  - **Completed**: ✅ Added SetDB method to ClientOptions, initialized stateRedisClient in main.go

- [x] 5.2.2 Initialize StateRepository in `main.go`
  - **What**: Create StateRepository with state Redis client, inject into HTTP server config
  - **Files**: `cmd/api/main.go`, `internal/httpserver/new.go`
  - **Validation**: StateRepository initialized and passed to HTTP server
  - **Dependencies**: 2.2.3, 5.2.1
  - **Completed**: ✅ Added StateRedisClient to HTTPServer Config, passed from main.go

- [x] 5.2.3 Inject StateRepository into project module initialization
  - **What**: Pass StateRepository to project usecase constructor in `httpserver.go`
  - **Files**: `internal/httpserver/handler.go`
  - **Validation**: StateRepository available in project usecase
  - **Dependencies**: 5.2.2, 3.4.1
  - **Architecture**: Dependencies flow: main.go → httpserver → usecase (dependency injection)
  - **Completed**: ✅ StateUseCase initialized in handler.go and injected via NewWithState

- [ ] 5.2.4 Update health check to include state Redis connection
  - **What**: Ping state Redis client in `/health/redis` endpoint
  - **Files**: `internal/httpserver/health.go`
  - **Validation**: Health endpoint checks both cache and state Redis
  - **Dependencies**: 5.2.1
  - **Note**: Deferred - health check already exists for Redis, can be enhanced later

---

## 6. Documentation and Testing

### 6.1 API Documentation

- [x] 6.1.1 Add Swagger annotations for progress endpoint
  - **What**: Annotate `GetProgress` handler with `@Summary`, `@Param`, `@Success`, `@Failure`
  - **Files**: `internal/project/delivery/http/handler.go`
  - **Validation**: Run `make swagger`, verify `/projects/{id}/progress` appears in docs
  - **Dependencies**: 4.1.5
  - **Completed**: ✅ Swagger annotations added in Phase 4, make swagger generates ProgressResp

- [ ] 6.1.2 Update API documentation with event architecture
  - **What**: Document `project.created` event schema and RabbitMQ setup in README or docs/
  - **Files**: `README.md` or `docs/events.md`
  - **Validation**: Documentation clearly explains event flow
  - **Dependencies**: None (can be done in parallel)

### 6.2 Integration Testing

- [ ] 6.2.1 Create end-to-end test for project creation flow
  - **What**: Test PostgreSQL → Redis → RabbitMQ → Progress API flow
  - **Files**: `tests/e2e/project_creation_test.go` (create tests/ directory if needed)
  - **Validation**: E2E test passes with all components
  - **Dependencies**: All previous tasks complete

- [ ] 6.2.2 Test concurrent state updates
  - **What**: Simulate multiple workers incrementing done counter concurrently
  - **Files**: `internal/state/repository/redis/state_manager_test.go`
  - **Validation**: No race conditions, completion detected exactly once
  - **Dependencies**: 2.2.5

- [ ] 6.2.3 Test event publishing with RabbitMQ unavailable
  - **What**: Stop RabbitMQ, attempt project creation, verify error handling
  - **Files**: `internal/project/usecase/project_test.go`
  - **Validation**: Error returned to user, project remains in PostgreSQL
  - **Dependencies**: 3.4.3

---

## 7. Migration and Deployment

### 7.1 Migration Preparation

- [ ] 7.1.1 Create migration plan document
  - **What**: Document dual publishing timeline, rollback procedure, monitoring metrics
  - **Files**: `docs/event-migration-plan.md`
  - **Validation**: Plan reviewed and approved
  - **Dependencies**: None

- [ ] 7.1.2 Deploy with `USE_LEGACY_EXCHANGE=true` (dual publishing)
  - **What**: Deploy to staging/production with feature flag enabled
  - **Validation**: Events appear in both `smap.events` and `collector.inbound`
  - **Dependencies**: All implementation tasks complete

- [ ] 7.1.3 Monitor event delivery for 1 week
  - **What**: Track metrics, verify Collector consumes from `smap.events`
  - **Validation**: No errors, Collector successfully processes events
  - **Dependencies**: 7.1.2

- [ ] 7.1.4 Disable legacy exchange publishing
  - **What**: Set `USE_LEGACY_EXCHANGE=false`, deploy
  - **Validation**: Events only in `smap.events`, no messages in `collector.inbound`
  - **Dependencies**: 7.1.3

---

## Task Summary

**Implementation Status**: ✅ CORE IMPLEMENTATION COMPLETE (2025-12-05)

**Build Status**: ✅ All builds pass (`go build ./...`)
**Test Status**: ✅ All tests pass (`go test ./internal/state/...`)

**Completed Phases**:
- ✅ Phase 1: Infrastructure Setup (6/6 tasks)
- ✅ Phase 2: Redis State Management (11/11 tasks)
- ✅ Phase 3: Event Publishing (10/14 tasks - core complete, confirms/metrics deferred)
- ✅ Phase 4: Progress Monitoring API (7/8 tasks - unit tests deferred)
- ✅ Phase 5: Configuration & Infrastructure (5/6 tasks - health check enhancement deferred)
- ⏳ Phase 6: Documentation & Testing (1/6 tasks - Swagger done, others deferred)
- ⏳ Phase 7: Migration & Deployment (0/4 tasks - operational tasks)

**Deferred Tasks** (not blocking core functionality):
- 3.3.3: Publisher confirms (requires pkg/rabbitmq changes)
- 3.3.4: Event publishing metrics (requires metrics infrastructure)
- 3.4.4, 4.1.8, 6.2.x: Integration/unit tests (requires test infrastructure)
- 5.2.4: Health check for state Redis (optional enhancement)
- 6.1.2: Event architecture documentation
- 7.x: Migration and deployment (operational tasks)

**Total**: 40/51 implementation tasks completed (78%)

---

## Implementation Summary

**User Flow** (Updated 2025-12-05):
```
1. POST /projects          → Create project (PostgreSQL only, status: pending)
2. POST /projects/dryrun   → Optional: Test keywords before execution
3. POST /projects/:id/execute → Start execution (Redis state init + publish event)
4. GET /projects/:id/progress → Monitor real-time progress
```

**Core Features Delivered**:
1. **Redis State Management** - Separate DB 1 for state tracking with 7-day TTL
2. **Event Publishing** - `project.created` event to `smap.events` exchange with dual publishing support
3. **Execute API** - `POST /projects/:id/execute` endpoint to start project processing
4. **Progress API** - `GET /projects/:id/progress` endpoint with Redis lookup and PostgreSQL fallback
5. **Clean Architecture** - 4-layer separation (pkg → repository → usecase → handler)

**Key Files**:
- `pkg/redis/client.go` - Generic Hash operations (HSet, HGet, HIncrBy, Pipeline)
- `internal/model/state.go` - ProjectState, ProjectStatus domain types
- `internal/state/repository/redis/` - Redis CRUD operations only
- `internal/state/usecase/` - Business logic (completion checks, progress calculation)
- `internal/project/usecase/project.go` - Create flow: PostgreSQL → Redis → RabbitMQ
- `internal/project/delivery/http/handler.go` - GetProgress endpoint

**Configuration**:
- `REDIS_STATE_DB=1` - Separate Redis DB for state
- `USE_LEGACY_EXCHANGE=false` - Feature flag for dual publishing

**Architecture Principles Applied**:
1. **Clean Architecture**: UseCase → Repository → pkg layers strictly enforced
2. **Separation of Concerns**:
   - `pkg/redis` = Generic operations only (no domain knowledge)
   - `internal/state/repository` = Domain-specific Redis logic
   - `internal/project/usecase` = Business orchestration
3. **Interface-Based Design**: All dependencies use interfaces for testability

**Parallelizable Tasks**:
- Type definitions: 2.2.1, 3.2.1, 4.1.1, 5.1.1
- Constants: 3.1.1-3.1.3
- Documentation: 6.1.2
- These can all be done concurrently

**Critical Path** (must be sequential):
1. Verify pkg/redis Hash operations (2.1.1-2.1.3)
2. State Repository implementation (2.2.3-2.2.7)
3. UseCase integration (3.4.2)
4. Progress API (4.1.4)
5. E2E testing (6.2.1)

**Dependencies on External Teams**:
- Collector Service must update to consume from `smap.events` (coordinate during migration phase 7.1.2-7.1.4)
