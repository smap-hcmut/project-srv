# Proposal: Align Project Service with Event-Driven Choreography

## Change ID
`2025-12-05-event-driven-alignment`

## Problem Statement

The Project Service currently lacks full compliance with the event-driven choreography architecture defined in `event-drivent.md`. While RabbitMQ and Redis infrastructure are initialized in `main.go`, critical components for distributed state management and event orchestration are missing:

1. **Redis State Management Missing**: No dedicated Redis DB for SMAP state tracking (as specified: DB 1 for state, DB 0 for cache)
2. **Incomplete Event Publishing**: Only dry-run events implemented; missing `project.created` event publishing
3. **No State Tracking Implementation**: Redis Hash structure (`smap:proj:{id}`) for tracking project lifecycle (status, total, done, errors) not implemented
4. **No Progress Monitoring**: No API endpoints for real-time project progress tracking
5. **Event Architecture Not Standardized**: Current RabbitMQ setup uses `collector.inbound` exchange instead of the standardized `smap.events` topic exchange

## Proposed Solution

Implement complete event-driven choreography compliance by:

1. **Redis State Management**: Add dedicated state DB configuration and Redis Hash operations for project lifecycle tracking
2. **Event Publishing Layer**: Implement `project.created` event publishing using standardized `smap.events` exchange
3. **State Tracking Module**: Create state management usecase for atomic operations on project state (initialize, update status, mark complete)
4. **Progress API**: Add HTTP endpoints for real-time project progress monitoring
5. **Standardize Event Architecture**: Migrate to `smap.events` topic exchange with proper routing keys

## Scope

### In Scope
- Redis state DB configuration (DB 1 for state management)
- Redis Hash operations for project state tracking
- Project state management usecase (initialization, status updates, completion tracking)
- `project.created` event publishing on project creation
- HTTP endpoints for project progress monitoring (`GET /projects/{id}/progress`)
- Migration of RabbitMQ exchanges to standardized `smap.events`
- Health check endpoint for Redis state connection

### Out of Scope
- Identity Service integration (being implemented in parallel)
- Collector Service event consumption (handled by Collector Service)
- Analytics Service implementation (separate service)
- Event retry mechanisms and dead letter queues (future enhancement)
- Multi-project progress aggregation (future feature)

## Dependencies

### Services Being Implemented in Parallel
- **Identity Service**: Authentication and user management (tasks 1.1 Redis Integration already completed by user)

### Required Infrastructure
- Redis server (local Docker container confirmed available with password: 21042004)
- RabbitMQ server (already initialized in main.go)
- PostgreSQL (already connected)

### Assumptions
- User has completed Identity Service tasks 1.1.1-1.1.6 (Redis integration)
- Redis state DB (DB 1) will be dedicated for SMAP state management only
- RabbitMQ `smap.events` exchange will be created declaratively by Project Service

## Impact Analysis

### Code Changes
- **New Files**: ~8 files (state management module, progress endpoints, Redis state client wrapper)
- **Modified Files**: ~5 files (main.go, config.go, template.env, project usecase, RabbitMQ constants)
- **Deleted Files**: None

### Breaking Changes
- **RabbitMQ Exchange Migration**: Existing dry-run events use `collector.inbound`, need migration path to `smap.events`
  - **Mitigation**: Publish to both exchanges during transition period (feature flag controlled)

### Performance Impact
- **Redis Operations**: +2-3 Redis operations per project creation (HSET for initialization, EXPIRE for TTL)
- **Event Publishing**: +1 RabbitMQ message per project creation
- **Expected Overhead**: <50ms per project creation (negligible)

## Success Criteria

### Functional Requirements
- [ ] Project creation triggers `project.created` event to `smap.events` exchange
- [ ] Redis Hash `smap:proj:{id}` initialized with status=INITIALIZING, total=0, done=0, errors=0
- [ ] GET `/projects/{id}/progress` returns real-time state from Redis
- [ ] Redis state DB (DB 1) separate from cache DB (DB 0)
- [ ] Health check endpoint `/health/redis` validates state DB connection

### Non-Functional Requirements
- [ ] Event publishing succeeds even if Redis state update fails (eventual consistency)
- [ ] Progress API responds within 100ms (Redis reads are fast)
- [ ] State keys have 7-day TTL to prevent memory leaks
- [ ] Atomic operations for state updates (using Redis pipelines)

### Testing Requirements
- [ ] Unit tests for state management operations
- [ ] Integration tests for event publishing flow
- [ ] Health check endpoint returns correct Redis state

## Open Questions

1. **Event Publishing Failure Handling**: If RabbitMQ publish fails after project is created in PostgreSQL, should we:
   - Option A: Rollback PostgreSQL transaction (ensures consistency but reduces availability)
   - Option B: Log error and rely on eventual consistency / manual retry (higher availability)
   - **Recommendation**: Option B with dead letter table for failed events (implement in future enhancement)

2. **State Initialization Timing**: Should Redis state be initialized:
   - Option A: Synchronously during project creation (ensures state exists before event published)
   - Option B: Asynchronously after event published (better performance, eventual consistency)
   - **Recommendation**: Option A to ensure Collector Service can immediately query state

3. **Migration Strategy for Existing Events**: For dry-run events currently using `collector.inbound`:
   - Option A: Immediate cutover (risky if Collector Service not ready)
   - Option B: Dual publishing to both exchanges with feature flag (safer transition)
   - **Recommendation**: Option B with 2-week transition period

## Related Changes

This change is part of the broader event-driven architecture initiative:
- **Identity Service**: Redis integration (completed in parallel)
- **Collector Service**: Will consume `project.created` events (future)
- **Analytics Service**: Will consume `data.collected` and publish `job.completed` (future)

## Timeline Estimate

**Note**: This is a planning estimate only, not a commitment.

- Proposal & Design: Completed in this phase
- Implementation: ~3-4 focused work sessions
- Testing & Validation: ~1-2 work sessions
- Total: ~4-6 work sessions

## References

- Event-Driven Architecture Guide: `project/event-drivent.md`
- Current Project Module: `internal/project/`
- RabbitMQ Implementation: `internal/project/delivery/rabbitmq/`
- Redis Package: `pkg/redis/`
