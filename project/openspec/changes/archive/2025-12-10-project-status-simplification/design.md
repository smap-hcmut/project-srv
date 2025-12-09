# Design Document: Project Status Simplification

## Overview

This design document outlines the refactoring of the project status system from a five-status model (draft, active, completed, archived, cancelled) to a simplified three-status model (draft, process, completed). The current system has unnecessary complexity with statuses that are not actively used or don't align with the actual project lifecycle. The new model will:

- Reduce cognitive overhead for developers and users
- Align status values with the actual execution flow
- Maintain clear separation between persistent PostgreSQL status and runtime Redis execution state
- Ensure backward compatibility for existing projects

The key insight is that the system already has two separate state tracking mechanisms:
1. **PostgreSQL Status**: Persistent project lifecycle state (draft → process → completed)
2. **Redis Execution State**: Runtime processing state (INITIALIZING → CRAWLING → PROCESSING → DONE/FAILED)

The refactoring will clarify this separation and remove unused status values.

## Architecture

### Current Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PROJECT SERVICE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  PostgreSQL Status (5 values)        Redis State           │
│  ┌──────────────────┐               ┌──────────────────┐   │
│  │ - draft          │               │ INITIALIZING     │   │
│  │ - active         │               │ CRAWLING         │   │
│  │ - completed      │               │ PROCESSING       │   │
│  │ - archived       │               │ DONE             │   │
│  │ - cancelled      │               │ FAILED           │   │
│  └──────────────────┘               └──────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Target Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PROJECT SERVICE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  PostgreSQL Status (3 values)        Redis State           │
│  ┌──────────────────┐               ┌──────────────────┐   │
│  │ - draft          │               │ INITIALIZING     │   │
│  │ - process        │               │ CRAWLING         │   │
│  │ - completed      │               │ PROCESSING       │   │
│  │                  │               │ DONE             │   │
│  │                  │               │ FAILED           │   │
│  └──────────────────┘               └──────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Status Lifecycle Flow

```
User Creates Project
        │
        ▼
   ┌─────────┐
   │  draft  │ ◄─── Can edit all fields
   └────┬────┘
        │
        │ User calls /execute
        ▼
   ┌─────────┐
   │ process │ ◄─── Redis state tracks detailed progress
   └────┬────┘      (INITIALIZING → CRAWLING → PROCESSING)
        │
        │ All tasks complete
        ▼
   ┌───────────┐
   │ completed │ ◄─── Results available, no re-execution
   └───────────┘
```

## Components and Interfaces

### 1. Model Layer (`internal/model/project.go`)

**Changes Required:**

```go
// BEFORE (5 statuses)
const (
    ProjectStatusDraft     = "draft"
    ProjectStatusActive    = "active"      // REMOVE
    ProjectStatusCompleted = "completed"
    ProjectStatusArchived  = "archived"    // REMOVE
    ProjectStatusCancelled = "cancelled"   // REMOVE
)

// AFTER (3 statuses)
const (
    ProjectStatusDraft     = "draft"
    ProjectStatusProcess   = "process"     // NEW
    ProjectStatusCompleted = "completed"
)

// Updated validation function
func IsValidProjectStatus(status string) bool {
    validStatuses := []string{
        ProjectStatusDraft,
        ProjectStatusProcess,
        ProjectStatusCompleted,
    }
    return slices.Contains(validStatuses, status)
}
```

**Interface:**
- `IsValidProjectStatus(status string) bool`: Validates status against the three allowed values
- `Project` struct: No changes to structure, only valid status values change

### 2. Repository Layer (`internal/project/repository/postgre/project.go`)

**Changes Required:**

The repository layer requires minimal changes since it doesn't enforce status validation - that's handled at the usecase layer. However, we need to ensure:

1. Migration handling for existing projects with old status values
2. Query filters work correctly with new status values

**Interface:**
- `Create(ctx, sc, opts)`: Sets initial status to "draft"
- `Update(ctx, sc, opts)`: Accepts new status values
- `List/Get(ctx, sc, opts)`: Filters by new status values

**Migration Strategy:**

```sql
-- Migration to update existing projects
UPDATE projects 
SET status = CASE 
    WHEN status = 'active' THEN 'process'
    WHEN status = 'archived' THEN 'completed'
    WHEN status = 'cancelled' THEN 'draft'
    ELSE status
END
WHERE status IN ('active', 'archived', 'cancelled');
```

### 3. UseCase Layer (`internal/project/usecase/project.go`)

**Changes Required:**

This is where the main business logic changes occur:

**Create Operation:**
```go
func (uc *usecase) Create(ctx, sc, ip) (ProjectOutput, error) {
    // ... validation ...
    
    // Create project with "draft" status (implicit)
    p, err := uc.repo.Create(ctx, sc, repository.CreateOptions{
        // Status is implicitly "draft" - set by database default
        // or explicitly in repository layer
    })
    
    return ProjectOutput{Project: p}, nil
}
```

**Execute Operation:**
```go
func (uc *usecase) Execute(ctx, sc, projectID) error {
    // 1. Get project and verify ownership
    p, err := uc.repo.Detail(ctx, sc, projectID)
    
    // 2. Verify project is in "draft" status
    if p.Status != model.ProjectStatusDraft {
        return project.ErrInvalidStatusTransition
    }
    
    // 3. Check for duplicate execution
    existingState, err := uc.stateUC.GetProjectState(ctx, projectID)
    if err == nil && existingState != nil {
        return project.ErrProjectAlreadyExecuting
    }
    
    // 4. Update PostgreSQL status to "process"
    _, err = uc.repo.Update(ctx, sc, repository.UpdateOptions{
        ID:     projectID,
        Status: &model.ProjectStatusProcess,
    })
    
    // 5. Initialize Redis state
    if err := uc.stateUC.InitProjectState(ctx, projectID); err != nil {
        // Rollback PostgreSQL status to "draft"
        rollbackStatus := model.ProjectStatusDraft
        uc.repo.Update(ctx, sc, repository.UpdateOptions{
            ID:     projectID,
            Status: &rollbackStatus,
        })
        return err
    }
    
    // 6. Publish event
    event := rabbitmq.ToProjectCreatedEvent(p)
    if err := uc.producer.PublishProjectCreated(ctx, event); err != nil {
        // Rollback both Redis and PostgreSQL
        uc.stateUC.DeleteProjectState(ctx, projectID)
        rollbackStatus := model.ProjectStatusDraft
        uc.repo.Update(ctx, sc, repository.UpdateOptions{
            ID:     projectID,
            Status: &rollbackStatus,
        })
        return err
    }
    
    return nil
}
```

**Patch Operation:**
```go
func (uc *usecase) Patch(ctx, sc, ip) (ProjectOutput, error) {
    // ... existing validation ...
    
    // Validate status transition if status is being updated
    if ip.Status != nil {
        if !model.IsValidProjectStatus(*ip.Status) {
            return ProjectOutput{}, project.ErrInvalidStatus
        }
        
        // Enforce valid transitions
        if err := uc.validateStatusTransition(p.Status, *ip.Status); err != nil {
            return ProjectOutput{}, err
        }
    }
    
    // ... rest of update logic ...
}

func (uc *usecase) validateStatusTransition(from, to string) error {
    // Valid transitions:
    // draft -> process (via Execute, not Patch)
    // process -> completed (via system, not user)
    // Any status can stay the same
    
    if from == to {
        return nil
    }
    
    // Users should not manually change status via Patch
    // Status changes happen through Execute or system events
    return project.ErrInvalidStatusTransition
}
```

**GetProgress Operation:**
```go
func (uc *usecase) GetProgress(ctx, sc, projectID) (ProgressOutput, error) {
    // ... ownership verification ...
    
    // Get Redis state for detailed progress
    state, err := uc.stateUC.GetProjectState(ctx, projectID)
    if err != nil || state == nil {
        // Fallback to PostgreSQL status
        return ProgressOutput{
            Project:         p,
            Status:          p.Status,  // Will be "draft", "process", or "completed"
            TotalItems:      0,
            ProcessedItems:  0,
            FailedItems:     0,
            ProgressPercent: 0,
        }, nil
    }
    
    // Return detailed progress from Redis
    var progressPercent float64
    if state.Total > 0 {
        progressPercent = float64(state.Done) / float64(state.Total) * 100
    }
    
    return ProgressOutput{
        Project:         p,
        Status:          string(state.Status),  // Redis execution status
        TotalItems:      state.Total,
        ProcessedItems:  state.Done,
        FailedItems:     state.Errors,
        ProgressPercent: progressPercent,
    }, nil
}
```

**Interface:**
- `Create(ctx, sc, ip) (ProjectOutput, error)`: Creates project with "draft" status
- `Execute(ctx, sc, projectID) error`: Transitions "draft" → "process", initializes Redis
- `Patch(ctx, sc, ip) (ProjectOutput, error)`: Validates status transitions
- `GetProgress(ctx, sc, projectID) (ProgressOutput, error)`: Returns status and progress

### 4. Delivery Layer (`internal/project/delivery/http/`)

**Changes Required:**

Minimal changes needed - primarily documentation and validation:

**presenter.go:**
```go
// PatchReq - status validation happens at usecase layer
type PatchReq struct {
    ID                 string           `uri:"id" binding:"required"`
    Description        *string          `json:"description"`
    Status             *string          `json:"status"`  // Must be "draft", "process", or "completed"
    BrandKeywords      []string         `json:"brand_keywords"`
    CompetitorKeywords []CompetitorsReq `json:"competitor_keywords"`
}

// GetReq - filter by new status values
type GetReq struct {
    IDs        []string `form:"ids"`
    Statuses   []string `form:"statuses"`  // Must be "draft", "process", or "completed"
    SearchName *string  `form:"search_name"`
    Paginate   paginator.PaginateQuery
}
```

**Interface:**
- HTTP endpoints remain unchanged
- Request/response structures remain unchanged
- Status field values change to new set

### 5. Error Handling

**New Error:**
```go
// internal/project/error.go
var (
    // ... existing errors ...
    ErrInvalidStatusTransition = errors.New("invalid status transition")
)

// Map to HTTP error
// internal/project/delivery/http/error.go
case project.ErrInvalidStatusTransition:
    return errInvalidStatusTransition

var errInvalidStatusTransition = errors.HTTPError{
    Code:    30009,
    Message: "invalid status transition",
    Status:  http.StatusBadRequest,
}
```

## Data Models

### Project Model

No structural changes to the Project model itself - only the valid values for the Status field change:

```go
type Project struct {
    ID                 string
    Name               string
    Description        *string
    Status             string  // Valid values: "draft", "process", "completed"
    FromDate           time.Time
    ToDate             time.Time
    BrandName          string
    CompetitorNames    []string
    BrandKeywords      []string
    CompetitorKeywords []CompetitorKeyword
    CreatedBy          string
    CreatedAt          time.Time
    UpdatedAt          time.Time
    DeletedAt          *time.Time
}
```

### Status Mapping

| Old Status  | New Status  | Rationale                                      |
| ----------- | ----------- | ---------------------------------------------- |
| draft       | draft       | Unchanged - initial state                      |
| active      | process     | Renamed to match execution semantics           |
| completed   | completed   | Unchanged - final state                        |
| archived    | completed   | Merged - archiving is a UI concern, not status |
| cancelled   | draft       | Merged - cancelled projects revert to draft    |

## 
Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*


### Property Reflection

After analyzing all acceptance criteria, several redundancies were identified:

**Redundant Properties:**
- Properties 1.1, 1.3, 5.3, and 6.2 all test the same thing: status validation accepts only the three valid values
- Properties 3.1 and 3.5 both test that PostgreSQL status is updated to "process" during execution
- These will be consolidated into single, comprehensive properties

**Combined Properties:**
- Status validation (1.1, 1.3, 5.3, 6.2) → Single property testing validation across all inputs
- Execute status transition (3.1, 3.5) → Single property testing database update during execution

**Non-testable Criteria:**
- 1.2, 1.4: Code structure concerns (code review)
- 5.4: Logging behavior (observability)
- 6.4: Documentation (manual review)

### Correctness Properties

Property 1: Status validation accepts only valid values
*For any* string value, the status validation function should return true if and only if the value is "draft", "process", or "completed"
**Validates: Requirements 1.1, 1.3, 5.3, 6.2**

Property 2: New projects start in draft status
*For any* valid project creation request, the created project should have status "draft"
**Validates: Requirements 2.1**

Property 3: Draft projects have no Redis state
*For any* project with "draft" status, querying Redis for execution state should return no data
**Validates: Requirements 2.3**

Property 4: Draft status persists immediately
*For any* newly created project, querying the database immediately should return the project with "draft" status
**Validates: Requirements 2.4**

Property 5: Execute transitions draft to process
*For any* project in "draft" status, executing it should result in the project status changing to "process" in PostgreSQL
**Validates: Requirements 3.1, 3.5**

Property 6: Execute initializes Redis state
*For any* project transitioning to "process" status via execute, Redis should contain execution state with status "INITIALIZING"
**Validates: Requirements 3.2**

Property 7: Execute publishes event
*For any* successful project execution, a project.created event should be published to RabbitMQ
**Validates: Requirements 3.3**

Property 8: Duplicate execution is rejected
*For any* project with "process" status, attempting to execute it should return an error
**Validates: Requirements 3.4**

Property 9: Completed projects retain Redis state
*For any* project with "completed" status, Redis execution state should still exist
**Validates: Requirements 4.2**

Property 10: Completed status persists
*For any* project that completes processing, the PostgreSQL status should be "completed"
**Validates: Requirements 4.3**

Property 11: Completed projects cannot be re-executed
*For any* project with "completed" status, attempting to execute it should return an error
**Validates: Requirements 4.4**

Property 12: Only draft projects can be executed
*For any* project with status other than "draft", attempting to execute it should return an error
**Validates: Requirements 5.1**

Property 13: Invalid transitions are rejected
*For any* status transition that skips intermediate states (e.g., draft → completed), the update should be rejected with an error
**Validates: Requirements 5.2**

Property 14: Invalid transitions return descriptive errors
*For any* invalid status transition attempt, the error message should clearly indicate the transition is invalid
**Validates: Requirements 5.5**

Property 15: HTTP responses contain valid status
*For any* project returned via HTTP API, the status field should be one of "draft", "process", or "completed"
**Validates: Requirements 6.1**

Property 16: Invalid status in requests returns 400
*For any* HTTP request containing an invalid status value, the response should be 400 Bad Request
**Validates: Requirements 6.3**

Property 17: Progress includes status and metrics
*For any* progress query response, both the status field and execution metrics (total, done, errors, percent) should be present
**Validates: Requirements 6.5**

Property 18: Draft projects allow full modification
*For any* project in "draft" status, update operations for all fields should succeed
**Validates: Requirements 2.2**

## Error Handling

### Error Types

The refactoring introduces one new error type and modifies validation for existing errors:

```go
// New error for invalid status transitions
var ErrInvalidStatusTransition = errors.New("invalid status transition")

// Existing errors that need updated validation
var ErrInvalidStatus = errors.New("invalid status")  // Now checks against 3 values
var ErrProjectAlreadyExecuting = errors.New("project already executing")  // Enhanced check
```

### Error Mapping

| Error                        | HTTP Status | Error Code | When It Occurs                                |
| ---------------------------- | ----------- | ---------- | --------------------------------------------- |
| ErrInvalidStatus             | 400         | 30006      | Status value not in {draft, process, completed} |
| ErrInvalidStatusTransition   | 400         | 30009      | Attempting invalid state transition           |
| ErrProjectAlreadyExecuting   | 409         | 30008      | Executing project already in process          |
| ErrUnauthorized              | 403         | 30005      | User doesn't own project                      |
| ErrProjectNotFound           | 404         | 30004      | Project doesn't exist                         |

### Error Handling Strategy

**Validation Errors:**
- Status validation happens at multiple layers:
  - Model layer: `IsValidProjectStatus()` function
  - UseCase layer: Business logic validation
  - Delivery layer: HTTP request validation

**Transaction Rollback:**
- If Redis initialization fails during Execute, PostgreSQL status rolls back to "draft"
- If event publishing fails during Execute, both Redis state and PostgreSQL status roll back
- Rollback ensures consistency between PostgreSQL and Redis

**Error Propagation:**
```
Repository Error → UseCase (wrap with context) → Handler (map to HTTP error)
```

## Testing Strategy

### Unit Testing Approach

Unit tests will focus on specific examples and edge cases:

**Model Layer Tests:**
- Test `IsValidProjectStatus()` with each valid status
- Test `IsValidProjectStatus()` with invalid statuses
- Test status constant values are correct strings

**UseCase Layer Tests:**
- Test Create sets status to "draft"
- Test Execute transitions "draft" → "process"
- Test Execute rejects non-draft projects
- Test Execute handles rollback on Redis failure
- Test Execute handles rollback on event publish failure
- Test Patch validates status values
- Test GetProgress returns correct status

**Repository Layer Tests:**
- Test Create persists "draft" status
- Test Update changes status correctly
- Test List/Get filters by new status values

### Property-Based Testing Approach

Property-based tests will verify universal properties across all inputs using **Go's testing/quick package** or **gopter library**:

**Property Test Configuration:**
- Minimum 100 iterations per property
- Use custom generators for project data
- Tag each test with the property number from design doc

**Test Generators:**

```go
// Generator for valid project status
func GenerateValidStatus() string {
    statuses := []string{"draft", "process", "completed"}
    return statuses[rand.Intn(len(statuses))]
}

// Generator for invalid project status
func GenerateInvalidStatus() string {
    invalid := []string{"active", "archived", "cancelled", "pending", "invalid", ""}
    return invalid[rand.Intn(len(invalid))]
}

// Generator for project data
func GenerateProject() model.Project {
    return model.Project{
        ID:          uuid.New().String(),
        Name:        randomString(10),
        Status:      GenerateValidStatus(),
        FromDate:    randomDate(),
        ToDate:      randomDate(),
        BrandName:   randomString(8),
        // ... other fields
    }
}
```

**Property Test Examples:**

```go
// Property 1: Status validation
func TestProperty1_StatusValidation(t *testing.T) {
    // Feature: project-status-simplification, Property 1: Status validation accepts only valid values
    property := func(status string) bool {
        isValid := model.IsValidProjectStatus(status)
        shouldBeValid := status == "draft" || status == "process" || status == "completed"
        return isValid == shouldBeValid
    }
    
    if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
        t.Error(err)
    }
}

// Property 2: New projects start in draft
func TestProperty2_NewProjectsDraft(t *testing.T) {
    // Feature: project-status-simplification, Property 2: New projects start in draft status
    property := func(name string, brandName string) bool {
        // Create project with random data
        input := project.CreateInput{
            Name:      name,
            BrandName: brandName,
            // ... other required fields
        }
        
        output, err := usecase.Create(ctx, scope, input)
        if err != nil {
            return false
        }
        
        return output.Project.Status == model.ProjectStatusDraft
    }
    
    if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
        t.Error(err)
    }
}

// Property 8: Duplicate execution rejected
func TestProperty8_DuplicateExecutionRejected(t *testing.T) {
    // Feature: project-status-simplification, Property 8: Duplicate execution is rejected
    property := func() bool {
        // Create and execute a project
        p := createTestProject(t)
        err := usecase.Execute(ctx, scope, p.ID)
        if err != nil {
            return false
        }
        
        // Try to execute again
        err = usecase.Execute(ctx, scope, p.ID)
        return err == project.ErrProjectAlreadyExecuting
    }
    
    if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
        t.Error(err)
    }
}
```

### Integration Testing

Integration tests will verify end-to-end flows:

- Create → Execute → GetProgress flow
- Status transitions with actual PostgreSQL and Redis
- Event publishing to actual RabbitMQ
- HTTP API with actual request/response cycle

### Test Coverage Goals

- Unit tests: 80%+ code coverage
- Property tests: All 18 properties implemented
- Integration tests: Critical paths (create, execute, progress)
- Manual testing: API documentation accuracy

## Migration Strategy

### Database Migration

**Migration File:** `migration/03_simplify_project_status.sql`

```sql
-- Update existing projects with old status values
UPDATE projects 
SET status = CASE 
    WHEN status = 'active' THEN 'process'
    WHEN status = 'archived' THEN 'completed'
    WHEN status = 'cancelled' THEN 'draft'
    ELSE status
END
WHERE status IN ('active', 'archived', 'cancelled');

-- Add check constraint for new status values (optional, for data integrity)
ALTER TABLE projects 
ADD CONSTRAINT check_project_status 
CHECK (status IN ('draft', 'process', 'completed'));
```

### Deployment Strategy

**Phase 1: Code Deployment**
1. Deploy new code with updated status constants
2. Keep backward compatibility - accept old status values in API but map them
3. Monitor logs for any old status values being used

**Phase 2: Data Migration**
1. Run database migration to update existing projects
2. Verify all projects have new status values
3. Monitor for any issues

**Phase 3: Cleanup**
1. Remove backward compatibility code
2. Update API documentation
3. Notify API consumers of changes

### Rollback Plan

If issues are discovered:
1. Revert code deployment
2. Rollback database migration:
```sql
UPDATE projects 
SET status = CASE 
    WHEN status = 'process' THEN 'active'
    ELSE status
END
WHERE status = 'process';
```

## Performance Considerations

### Impact Analysis

**Positive Impacts:**
- Simpler validation logic (3 values vs 5) - marginal performance improvement
- Clearer code paths - easier for compiler optimization
- Reduced cognitive load - faster development

**Neutral Impacts:**
- Database queries unchanged (still filtering by status)
- Redis operations unchanged
- API response times unchanged

**No Negative Impacts Expected:**
- No additional database queries
- No additional Redis operations
- No additional network calls

### Monitoring

Monitor these metrics after deployment:
- API response times (should remain stable)
- Error rates (should not increase)
- Database query performance (should remain stable)
- Redis operation latency (should remain stable)

## Security Considerations

### Authorization

No changes to authorization model:
- Users can only access their own projects
- JWT token required for all operations
- Ownership verified at usecase layer

### Data Integrity

Enhanced data integrity:
- Database check constraint ensures only valid status values
- Status transition validation prevents invalid state changes
- Rollback mechanisms ensure consistency

### Audit Trail

Status transitions are logged:
- All status changes logged at INFO level
- Failed transitions logged at WARN level
- Audit trail maintained in application logs

## Documentation Updates

### API Documentation

Update Swagger/OpenAPI documentation:
- Update status enum to {draft, process, completed}
- Update status descriptions
- Add examples with new status values
- Document error codes for invalid transitions

### Code Documentation

Update code comments:
- Update status constant documentation
- Document status transition rules
- Add examples of valid/invalid transitions

### Architecture Documentation

Update architecture docs:
- Update `project-behavior.md` with new status model
- Update sequence diagrams with new status values
- Update state machine diagrams

## Appendix: Status Transition State Machine

```
┌─────────────────────────────────────────────────────────────┐
│                   Project Status State Machine              │
└─────────────────────────────────────────────────────────────┘

                    ┌──────────────┐
                    │   [START]    │
                    └──────┬───────┘
                           │
                           │ Create Project
                           ▼
                    ┌──────────────┐
              ┌─────│    draft     │◄────┐
              │     └──────┬───────┘     │
              │            │             │
              │            │ Execute     │ Rollback on
              │            │             │ failure
              │            ▼             │
              │     ┌──────────────┐    │
              │     │   process    │────┘
              │     └──────┬───────┘
              │            │
              │            │ All tasks complete
              │            │ (via webhook)
              │            ▼
              │     ┌──────────────┐
              └────►│  completed   │
                    └──────────────┘

Valid Transitions:
- draft → process (via Execute API)
- process → completed (via system/webhook)
- process → draft (rollback on failure)

Invalid Transitions:
- draft → completed (must go through process)
- completed → draft (no going back)
- completed → process (no re-execution)
- Any transition via Patch API (status changes only through Execute or system)
```

## Appendix: Redis vs PostgreSQL Status

The system maintains two separate status tracking mechanisms:

**PostgreSQL Status (Persistent):**
- Lifecycle state: draft, process, completed
- Persisted permanently
- Visible to users via API
- Updated by Project Service

**Redis Execution State (Runtime):**
- Processing state: INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED
- Temporary (7-day TTL)
- Used for progress tracking
- Updated by Collector Service

**Relationship:**
```
PostgreSQL "process" status
    ↓
    Contains detailed Redis state:
    - INITIALIZING (just started)
    - CRAWLING (collecting data)
    - PROCESSING (analyzing data)
    - DONE (finished successfully)
    - FAILED (error occurred)
    ↓
PostgreSQL "completed" status (when Redis shows DONE)
```

This separation allows:
- Simple, stable lifecycle tracking in PostgreSQL
- Detailed, real-time progress tracking in Redis
- Clear ownership: Project Service owns PostgreSQL, Collector owns Redis
