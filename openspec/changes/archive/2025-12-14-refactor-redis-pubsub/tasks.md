# Redis Pub/Sub Refactor Tasks

## Phase 1: Foundation (Day 1-2)

### Task 1.1: Create Message Type System

- [x] Define Platform, Status, MediaType enums in `webhook/redis_types.go`
- [x] Create JobMessage and ProjectMessage structs with proper JSON tags
- [x] Define shared data structures: Progress, BatchData, ContentItem, AuthorInfo, MetricsInfo, MediaInfo
- [x] Add comprehensive struct documentation and validation tags

### Task 1.2: Implement Transformation Functions

- [x] Create `webhook/transformers.go` with transformation logic
- [x] Implement `TransformDryRunCallback()` for dry-run job messages
- [x] Implement `TransformProjectCallback()` for project progress messages
- [x] Add helper functions for content and error transformations
- [x] Add mapping functions for platforms, statuses, and media types

### Task 1.3: Write Transformation Tests

- [x] Unit tests for `TransformDryRunCallback()` with various inputs
- [x] Unit tests for `TransformProjectCallback()` with edge cases
- [x] Test error transformation and edge case handling
- [x] Test content structure transformations
- [x] Achieve 100% test coverage for transformation logic

## Phase 2: Handler Updates (Day 2-3)

### Task 2.1: Update Dry-Run Handler

- [x] Modify `HandleDryRunCallback()` in `webhook/usecase/webhook.go`
- [x] Change topic pattern from `user_noti:{userID}` to `job:{jobID}:{userID}`
- [x] Replace generic message with `JobMessage` struct
- [x] Update error logging to include new message format context
- [x] Maintain backward compatibility during transition

### Task 2.2: Update Project Progress Handler

- [x] Modify `HandleProgressCallback()` in `webhook/usecase/webhook.go`
- [x] Change topic pattern from `user_noti:{userID}` to `project:{projectID}:{userID}`
- [x] Replace generic message with `ProjectMessage` struct
- [x] Ensure proper error handling and logging
- [x] Validate required fields before publishing

### Task 2.3: Integration Testing

- [x] Create integration tests with mock Redis client
- [x] Test new topic patterns are correctly generated
- [x] Verify message structures match specifications
- [x] Test with real crawler callback payloads
- [x] Test with real collector progress callbacks

## Phase 3: Testing & Quality (Day 3-4)

### Task 3.1: Comprehensive Test Suite

- [x] End-to-end tests from webhook callback to Redis publish
- [x] Load testing with high message volume
- [x] Error case testing (malformed inputs, Redis failures)
- [x] Boundary condition testing (empty content, zero progress)
- [x] Performance benchmarking vs. current implementation

### Task 3.2: Code Quality & Documentation

- [x] Code review for transformation logic and handlers
- [x] Update inline documentation and comments
- [x] Ensure Go code follows project conventions
- [x] Verify error handling patterns match project standards
- [x] Update relevant API documentation if needed

### Task 3.3: Monitoring & Observability

- [x] Add structured logging for new message formats
- [x] Include topic patterns in log messages for debugging
- [x] Add metrics for message transformation performance
- [x] Ensure error cases are properly logged with context

## Phase 4: Deployment Preparation (Day 4-5)

### Task 4.1: Environment Preparation

- [ ] Verify staging environment Redis configuration
- [ ] Test new message patterns in staging environment
- [ ] Coordinate with WebSocket service team for subscriber updates
- [x] Prepare rollback procedures and documentation

### Task 4.2: Deployment Scripts & Procedures

- [x] Create deployment checklist with verification steps
- [ ] Prepare monitoring dashboards for message flow tracking
- [x] Document topic pattern changes for operations team
- [x] Create troubleshooting guide for new message formats

### Task 4.3: Production Deployment

- [ ] Deploy to staging and verify end-to-end functionality
- [ ] Coordinate deployment timing with WebSocket service
- [ ] Deploy to production with careful monitoring
- [ ] Verify message delivery and format correctness
- [ ] Monitor for any performance or functional issues

## Phase 5: Cleanup & Optimization (Day 5-6)

### Task 5.1: Legacy Code Cleanup

- [x] Remove old generic message construction logic (if safe)
- [x] Clean up unused imports and dead code
- [x] Update any remaining generic `map[string]interface{}` usage
- [x] Ensure consistent error handling patterns

### Task 5.2: Performance Monitoring

- [ ] Monitor message processing performance improvements
- [ ] Track Redis pub/sub throughput metrics
- [ ] Verify memory usage improvements from structured types
- [ ] Document actual performance gains vs. targets

### Task 5.3: Documentation Updates

- [x] Update internal architecture documentation
- [x] Document new topic patterns for future developers
- [x] Update troubleshooting guides with new message formats
- [x] Archive old implementation documentation

## Validation Criteria

Each task must meet:

- [ ] **Functionality**: New implementation works correctly with real data
- [ ] **Performance**: No regression, ideally 20% improvement
- [ ] **Reliability**: Zero message loss during transition
- [ ] **Observability**: Proper logging and error tracking
- [ ] **Maintainability**: Code follows project conventions and is well-documented

## Dependencies & Coordination

- **WebSocket Service**: Must be ready to handle new topic patterns
- **Operations Team**: Needs updated monitoring and alerting for new patterns
- **Testing Team**: May need updated test scenarios for new message formats

## Risk Mitigation

- All changes are backward compatible during transition
- Comprehensive test coverage before deployment
- Gradual rollout with monitoring at each step
- Rollback procedures documented and tested
