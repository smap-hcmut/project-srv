# Redis Pub/Sub Pattern Refactor Proposal

**Change ID**: `refactor-redis-pubsub`  
**Type**: Architecture Refactor  
**Status**: ✅ COMPLETED  
**Created**: 2025-12-13  
**Completed**: 2025-12-14

## Summary

Refactor the Project Service Redis pub/sub pattern from a single generic topic to multiple topic-specific patterns, improving message routing efficiency, type safety, and maintainability.

## Background

Currently, all Redis messages (dry-run results and project progress) are published to a single generic topic pattern `user_noti:{userID}` with type discrimination through message payload. This creates several issues:

1. **Single topic overload**: All message types flow through one channel
2. **Type discrimination complexity**: WebSocket service must parse `type` field to route messages
3. **Generic structure**: No type safety, difficult to maintain
4. **Mixed payload formats**: Inconsistent data structures across message types

## Goals

- **Improved Message Routing**: Topic-specific subscriptions for better scalability
- **Type Safety**: Replace generic `map[string]interface{}` with structured Go types
- **Better Performance**: Eliminate type discrimination overhead
- **Enhanced Maintainability**: Self-documenting message structures

## Proposed Changes

### Topic Pattern Changes

**Current (Generic)**:

- All messages: `user_noti:{userID}`
- Message type determined by `payload.type` field

**Proposed (Topic-Specific)**:

- Dry-run jobs: `job:{jobID}:{userID}`
- Project progress: `project:{projectID}:{userID}`
- No type discrimination needed

### Message Structure Changes

**Current**: Generic structure with type field

```go
message := map[string]interface{}{
    "type": "dryrun_result|project_progress",
    "payload": map[string]interface{}{...}
}
```

**Proposed**: Topic-specific structured messages

```go
// For job:{jobID}:{userID}
type JobMessage struct {
    Platform Platform   `json:"platform"`
    Status   Status     `json:"status"`
    Batch    *BatchData `json:"batch,omitempty"`
    Progress *Progress  `json:"progress,omitempty"`
}

// For project:{projectID}:{userID}
type ProjectMessage struct {
    Status   Status    `json:"status"`
    Progress *Progress `json:"progress,omitempty"`
}
```

## Impact Analysis

### Low Risk

- **No external API changes**: Only internal Redis pub/sub refactor
- **Backward compatible**: WebSocket service can handle both formats during transition
- **No breaking changes**: External services (crawler, collector) unchanged

### Moderate Effort

- **Message transformation**: Content and error structure mapping required
- **Type system**: New enums and structured types needed
- **Testing**: Comprehensive unit and integration tests required

## Success Criteria

1. **Zero message loss** during transition period
2. **Improved performance**: 20% faster message processing target
3. **Type safety**: 100% structured types, no `map[string]interface{}`
4. **Test coverage**: 100% coverage for transformation logic
5. **Successful deployment**: Zero downtime migration

## Rollback Plan

If issues arise:

1. **Immediate**: Revert to previous deployment
2. **Gradual**: WebSocket service supports both old and new formats
3. **Fallback**: Can disable new logic via feature flag (if implemented)

## Dependencies

- **WebSocket Service**: Must be updated to handle new topic patterns and message structures
- **Redis Infrastructure**: No changes required
- **External Services**: No changes required (crawler, collector keep existing callback format)

## Risk Assessment

**Technical Risk**: Medium

- Message structure transformation complexity
- Potential data loss during transition

**Business Risk**: Low

- No user-facing changes
- Internal architecture improvement only

**Mitigation**:

- Comprehensive testing strategy
- Gradual rollout with monitoring
- Backward compatibility during transition

## Implementation Status

All coding tasks have been completed:

1. ✅ Created detailed technical specifications (design.md)
2. ✅ Implemented transformation functions (`transformers.go`)
3. ✅ Implemented new message structures (`redis_types.go`)
4. ✅ Updated webhook handlers (`webhook.go`)
5. ✅ Developed comprehensive test suite (`e2e_test.go`, `transformers_test.go`, etc.)
6. ✅ Created deployment documentation

### Files Created/Modified

| File                                            | Description                              |
| ----------------------------------------------- | ---------------------------------------- |
| `internal/webhook/redis_types.go`               | JobMessage, ProjectMessage, enums        |
| `internal/webhook/usecase/webhook.go`           | Updated handlers with new topic patterns |
| `internal/webhook/usecase/transformers.go`      | Transformation functions                 |
| `internal/webhook/usecase/job_mapping.go`       | Job mapping storage                      |
| `internal/webhook/usecase/e2e_test.go`          | End-to-end tests                         |
| `document/redis_pubsub_deployment_checklist.md` | Deployment guide                         |
| `document/redis_pubsub_rollback_procedures.md`  | Rollback procedures                      |
| `document/redis_pubsub_troubleshooting.md`      | Troubleshooting guide                    |
| `document/redis_pubsub_legacy_archive.md`       | Legacy implementation archive            |
| `document/architecture.md`                      | Updated with webhook module section      |

### Remaining Operational Tasks

The following tasks require manual action in staging/production:

- Verify staging environment Redis configuration
- Coordinate with WebSocket service team
- Deploy and monitor in production
