# Update Webhook Phase-Based Progress Proposal

**Change ID**: `update-webhook-phase-progress`  
**Type**: Feature Enhancement  
**Status**: 🔄 PROPOSED  
**Created**: 2025-12-15

## Summary

Update Project Service webhook format to support phase-based progress tracking (crawl + analyze phases), replacing the current flat progress structure with a more detailed two-phase model that provides granular visibility into project execution.

## Background

Currently, the webhook callback from Collector Service uses a flat progress structure:

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "CRAWLING",
  "total": 100,
  "done": 50,
  "errors": 2
}
```

This structure has limitations:

1. **Single phase visibility**: Cannot distinguish between crawl and analyze progress
2. **Unclear status mapping**: Status like "CRAWLING" doesn't reflect the full pipeline
3. **No overall progress**: Clients must calculate overall progress manually
4. **Limited error tracking**: Single error count doesn't show which phase has issues

## Goals

- **Phase-Based Tracking**: Separate progress for crawl and analyze phases
- **Overall Progress**: Provide calculated overall progress percentage
- **Better Status Model**: Unified "PROCESSING" status with phase details
- **Enhanced Error Visibility**: Per-phase error tracking
- **Backward Compatibility**: Support old format during transition period

## Proposed Changes

### Webhook Format Changes

**Current Format (deprecated)**:

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "CRAWLING",
  "total": 100,
  "done": 50,
  "errors": 2
}
```

**New Format**:

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
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

### State Model Changes

Update `ProjectState` to track both phases:

- `crawl_total`, `crawl_done`, `crawl_errors`
- `analyze_total`, `analyze_done`, `analyze_errors`
- Progress calculation methods for each phase and overall

### API Response Changes

Update `GET /projects/:id/progress` to return phase-based progress matching the new webhook format.

### WebSocket Message Changes

Update message payload to include phase-based progress for both `project_progress` and `project_completed` message types.

## Impact Analysis

### Low Risk

- **Internal changes only**: Webhook format is internal between Collector and Project Service
- **Backward compatible**: Handler supports both old and new formats
- **No breaking changes**: External clients receive enhanced data

### Moderate Effort

- **Struct updates**: New PhaseProgress struct and updated request/response types
- **Handler logic**: Backward compatibility detection and handling
- **State model**: Extended fields for phase tracking
- **Testing**: Unit tests for both formats

## Success Criteria

1. **Backward Compatibility**: Old webhook format continues to work
2. **Phase Visibility**: Clients can see crawl and analyze progress separately
3. **Accurate Progress**: Overall progress correctly calculated from phases
4. **Test Coverage**: 100% coverage for new transformation logic
5. **Zero Downtime**: Seamless transition without service interruption

## Dependencies

- **Collector Service**: Must be updated to send new webhook format
- **WebSocket Service**: No changes needed (passes through messages)
- **Frontend**: Must be updated to display phase-based progress

## Risk Assessment

**Technical Risk**: Low

- Simple struct additions and handler logic
- Backward compatibility ensures safe rollout

**Business Risk**: Low

- Enhanced visibility for users
- No functionality removed

**Mitigation**:

- Backward compatibility during transition
- Comprehensive testing before deployment
- Gradual rollout with monitoring

## Files to Modify

| File                                        | Change                                            |
| ------------------------------------------- | ------------------------------------------------- |
| `internal/webhook/types.go`                 | Add PhaseProgress, update ProgressCallbackRequest |
| `internal/webhook/usecase/webhook.go`       | Update handler with backward compat               |
| `internal/model/state.go`                   | Update ProjectState struct                        |
| `internal/state/repository/redis/redis.go`  | Update field names                                |
| `internal/project/delivery/http/types.go`   | Add response structs                              |
| `internal/project/delivery/http/handler.go` | Update GetProgress handler                        |
