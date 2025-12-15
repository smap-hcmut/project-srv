# Phase-Based Progress Tasks

## Phase 1: Type System Updates (Day 1)

### Task 1.1: Create PhaseProgress Struct

- [x] Define `PhaseProgress` struct in `internal/webhook/type.go`
- [x] Add JSON tags for serialization
- [x] Add validation tags if needed
- [x] Document struct fields

### Task 1.2: Update ProgressCallbackRequest

- [x] Add `Crawl` field with PhaseProgress type
- [x] Add `Analyze` field with PhaseProgress type
- [x] Add `OverallProgressPercent` field
- [x] Keep old fields (Total, Done, Errors) for backward compat
- [x] Add `omitempty` tags to old fields

### Task 1.3: Create Response Structs

- [x] Define `PhaseProgressResp` in `internal/project/delivery/http/presenter.go`
- [x] Define `ProjectProgressResp` struct
- [x] Add proper JSON tags
- [x] Document response format

## Phase 2: State Model Updates (Day 1-2)

### Task 2.1: Update ProjectState Struct

- [x] Add `CrawlTotal`, `CrawlDone`, `CrawlErrors` fields
- [x] Add `AnalyzeTotal`, `AnalyzeDone`, `AnalyzeErrors` fields
- [x] Keep old flat fields for backward compatibility
- [x] Update JSON tags

### Task 2.2: Add Progress Calculation Methods

- [x] Implement `CrawlProgressPercent()` method
- [x] Implement `AnalyzeProgressPercent()` method
- [x] Implement `OverallProgressPercent()` method
- [x] Handle edge cases (zero total, all errors)

### Task 2.3: Update Redis Field Constants

- [x] Add `fieldCrawlTotal`, `fieldCrawlDone`, `fieldCrawlErrors`
- [x] Add `fieldAnalyzeTotal`, `fieldAnalyzeDone`, `fieldAnalyzeErrors`
- [x] Update repository to use new fields (InitState, GetState)

### Task 2.4: Write State Model Tests

- [x] Test progress calculation with normal data
- [x] Test edge cases (zero total, all errors)
- [x] Test JSON serialization/deserialization

## Phase 3: Handler Updates (Day 2)

### Task 3.1: Implement Format Detection

- [x] Add logic to detect new vs old format (`isNewProgressFormat()`)
- [x] Check for Crawl/Analyze data presence
- [x] Route to appropriate handler (`transformNewProgressFormat()` / `transformOldProgressFormat()`)

### Task 3.2: Implement New Format Handler

- [x] Create `transformNewProgressFormat()` function
- [x] Extract phase data from request
- [x] Build WebSocket message with phase structure (`buildProgressWebSocketMessage()`)
- [x] Publish to Redis with new format

### Task 3.3: Update Old Format Handler

- [x] Keep existing `transformOldProgressFormat()` logic
- [x] Map old status to phase via `mapProjectStatus()`
- [x] Convert to new internal format
- [x] Log deprecation warning (`Warnf`)

### Task 3.4: Update WebSocket Message Building

- [x] Include `crawl` object in payload
- [x] Include `analyze` object in payload
- [x] Include `overall_progress_percent`
- [x] Determine message type (progress vs completed)

### Task 3.5: Write Handler Tests

- [x] Test new format handling (`TestTransformProjectCallback_NewFormat`)
- [x] Test old format backward compatibility (`TestTransformProjectCallback`)
- [x] Test format detection logic (`TestIsNewProgressFormat`)
- [x] Test phase errors building (`TestBuildPhaseErrors`)

## Phase 4: API Updates (Day 2-3)

### Task 4.1: Add GetPhaseProgress Handler

- [x] Add `GetPhaseProgress` method to UseCase interface
- [x] Implement `GetPhaseProgress` in usecase (fetch state from Redis)
- [x] Add `ProjectProgressOutput` and `PhaseProgressOutput` types
- [x] Add `GetPhaseProgress` handler in HTTP delivery
- [x] Add `newProjectProgressResp` presenter function
- [x] Add route `/projects/:id/phase-progress`

### Task 4.2: Update API Documentation

- [x] Update Swagger annotations for new endpoint
- [x] Document new response format (`ProjectProgressResp`)
- [x] Run `make swagger` - generated docs include new types

### Task 4.3: Build Verification

- [x] All files pass diagnostics (no errors)
- [x] `go build ./...` succeeds
- [x] Swagger docs generated successfully

## Phase 5: Integration Testing (Day 3)

### Task 5.1: End-to-End Tests

- [x] Test webhook → Redis → WebSocket flow (`TestE2E_ProjectProgressToRedisPublish`)
- [x] Test with new format payload (`TestE2E_PhaseBasedProgressToRedisPublish`)
- [x] Test with old format payload (`TestE2E_ProjectProgressToRedisPublish`)
- [x] Verify message structure at each step (`TestE2E_PhaseBasedProgressCompleted`)

### Task 5.2: Backward Compatibility Tests

- [x] Send old format, verify handling (`TestBackwardCompatibility_OldFormatStillWorks`)
- [x] Send new format, verify handling (`TestE2E_PhaseBasedProgressToRedisPublish`)
- [x] Mixed format scenarios (`TestMixedFormatScenarios`)
- [x] Verify no data loss - all tests pass

### Task 5.3: Load Testing

- [x] Test with high message volume (`TestLoadTesting_HighMessageVolume` - 100 concurrent messages)
- [x] Verify performance is acceptable (benchmarks pass)
- [x] Check memory usage (`BenchmarkHandleProgressCallback` - 3363 B/op, 57 allocs/op)

## Phase 6: Documentation & Cleanup (Day 3-4)

### Task 6.1: Update Documentation

- [x] Update `document/integration-project-websocket.md`
- [x] Update architecture documentation
- [x] Add migration notes

### Task 6.2: Code Review

- [x] Review all changes
- [x] Ensure code follows project conventions
- [x] Check error handling patterns
- [x] Verify logging is appropriate

### Task 6.3: Prepare Deployment

- [x] Create deployment checklist (`document/phase-progress-deployment-checklist.md`)
- [x] Document rollback procedure
- [ ] Coordinate with Collector team (external dependency)
- [ ] Plan frontend updates (external dependency)

## Validation Criteria

Each task must meet:

- [x] **Functionality**: Works correctly with both formats
- [x] **Backward Compat**: Old format continues to work
- [x] **Test Coverage**: Unit tests for new code
- [x] **Documentation**: Code is well-documented
- [x] **Conventions**: Follows project coding standards

## Dependencies

- **Collector Service**: Will need to send new format (can be done after Project Service is ready)
- **Frontend**: Will need to display phase progress (can be done after API is ready)
- **WebSocket Service**: No changes needed (passes through messages)

## Phase 7: Configuration & Documentation Sync (Day 4)

### Task 7.1: Sync Config with Env Template

- [x] Review `template.env` for any new config needed
- [x] Review `k8s/configmap.yaml` for any new config needed
- [x] Verify no new environment variables required for phase progress (confirmed: no new config needed)

### Task 7.2: Update Manifest Files

- [x] Review `k8s/configmap.yaml` - no changes needed (no new config)
- [x] Review `k8s/secret.yaml` - no changes needed (no new secrets)
- [x] Verify deployment manifests are up to date (confirmed: no changes required)

### Task 7.3: Update Current Doc Behavior

- [x] Update `document/project-behavior.md` with phase-based progress
- [x] Add new webhook format documentation
- [x] Add new API endpoint documentation (`GET /projects/:id/phase-progress`)
- [x] Update state management section (Redis fields, state operations)

## Risk Mitigation

- Backward compatibility ensures safe rollout
- Old format support can remain indefinitely
- Each service can be updated independently
- Rollback is simple (revert Project Service deployment)
