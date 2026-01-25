# Implementation Plan

- [x] 1. Update model layer with new status constants





  - Update `internal/model/project.go` to define only three status constants: "draft", "process", "completed"
  - Remove constants for "active", "archived", and "cancelled"
  - Update `IsValidProjectStatus()` function to validate against three values only
  - Use `slices.Contains()` for cleaner validation logic
  - _Requirements: 1.1, 1.3_

- [ ]* 1.1 Write property test for status validation
  - **Property 1: Status validation accepts only valid values**
  - **Validates: Requirements 1.1, 1.3, 5.3, 6.2**

- [x] 2. Create database migration for existing projects





  - Create migration file `migration/03_simplify_project_status.sql`
  - Write UPDATE statement to map old statuses to new ones (active→process, archived→completed, cancelled→draft)
  - Add check constraint to enforce valid status values
  - Test migration on sample data
  - _Requirements: 1.5_

- [x] 3. Update error handling for status transitions





  - Add `ErrInvalidStatusTransition` error to `internal/project/error.go`
  - Map error to HTTP 400 in `internal/project/delivery/http/error.go` with code 30009
  - Update error documentation
  - _Requirements: 5.5_

- [ ]* 3.1 Write unit tests for new error type
  - Test error message formatting
  - Test HTTP error mapping
  - _Requirements: 5.5_

- [x] 4. Update usecase layer - Create operation




  - Ensure `Create()` in `internal/project/usecase/project.go` sets initial status to "draft"
  - Verify status is persisted to PostgreSQL immediately
  - _Requirements: 2.1, 2.4_

- [ ]* 4.1 Write property test for create operation
  - **Property 2: New projects start in draft status**
  - **Validates: Requirements 2.1**

- [ ]* 4.2 Write property test for draft persistence
  - **Property 4: Draft status persists immediately**
  - **Validates: Requirements 2.4**

- [x] 5. Update usecase layer - Execute operation





  - Modify `Execute()` to verify project is in "draft" status before execution
  - Add status transition from "draft" to "process" in PostgreSQL
  - Implement rollback to "draft" if Redis initialization fails
  - Implement rollback to "draft" if event publishing fails
  - Ensure Redis state is initialized after PostgreSQL update
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 5.1_

- [ ]* 5.1 Write property test for execute status transition
  - **Property 5: Execute transitions draft to process**
  - **Validates: Requirements 3.1, 3.5**

- [ ]* 5.2 Write property test for Redis initialization
  - **Property 6: Execute initializes Redis state**
  - **Validates: Requirements 3.2**

- [ ]* 5.3 Write property test for event publishing
  - **Property 7: Execute publishes event**
  - **Validates: Requirements 3.3**

- [ ]* 5.4 Write property test for duplicate execution rejection
  - **Property 8: Duplicate execution is rejected**
  - **Validates: Requirements 3.4**

- [ ]* 5.5 Write property test for draft-only execution
  - **Property 12: Only draft projects can be executed**
  - **Validates: Requirements 5.1**

- [ ]* 5.6 Write unit tests for rollback scenarios
  - Test rollback when Redis initialization fails
  - Test rollback when event publishing fails
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 6. Update usecase layer - Patch operation





  - Add `validateStatusTransition()` helper function
  - Validate status value is one of three allowed values
  - Enforce valid status transitions (prevent skipping states)
  - Return `ErrInvalidStatusTransition` for invalid transitions
  - _Requirements: 5.2, 5.3, 5.5_

- [ ]* 6.1 Write property test for status transition validation
  - **Property 13: Invalid transitions are rejected**
  - **Validates: Requirements 5.2**

- [ ]* 6.2 Write property test for error messages
  - **Property 14: Invalid transitions return descriptive errors**
  - **Validates: Requirements 5.5**

- [ ]* 6.3 Write property test for draft modification
  - **Property 18: Draft projects allow full modification**
  - **Validates: Requirements 2.2**

- [x] 7. Update usecase layer - GetProgress operation





  - Ensure fallback to PostgreSQL status when Redis state is unavailable
  - Return status as one of three valid values
  - Include execution metrics in response
  - _Requirements: 6.5_

- [ ]* 7.1 Write property test for draft projects with no Redis state
  - **Property 3: Draft projects have no Redis state**
  - **Validates: Requirements 2.3**

- [ ]* 7.2 Write property test for progress response completeness
  - **Property 17: Progress includes status and metrics**
  - **Validates: Requirements 6.5**



- [x] 8. Update delivery layer - HTTP handlers



  - Update status validation in request processing
  - Ensure API responses include only valid status values
  - Return 400 Bad Request for invalid status values in requests
  - _Requirements: 6.1, 6.3_

- [ ]* 8.1 Write property test for HTTP response status
  - **Property 15: HTTP responses contain valid status**
  - **Validates: Requirements 6.1**

- [ ]* 8.2 Write property test for invalid status rejection
  - **Property 16: Invalid status in requests returns 400**
  - **Validates: Requirements 6.3**

- [x] 9. Update documentation






  - Update `document/project-behavior.md` with new status model
  - Update API documentation (Swagger/OpenAPI) with new status enum
  - Update status descriptions and examples
  - Document status transition rules
  - Add state machine diagram
  - _Requirements: 6.4_

- [x] 10. Checkpoint - Ensure all tests pass





  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. Run database migration





  - Execute migration on development database
  - Verify all projects have valid status values
  - Check for any projects that weren't migrated correctly
  - _Requirements: 1.5_

- [ ]* 11.1 Write integration test for migration
  - Create projects with old status values
  - Run migration
  - Verify status values are updated correctly
  - _Requirements: 1.5_

- [x] 12. Update completed project handling




  - Verify completed projects retain Redis state
  - Ensure completed projects cannot be re-executed
  - Test that completed status persists correctly
  - _Requirements: 4.2, 4.3, 4.4_

- [ ]* 12.1 Write property test for completed project Redis state
  - **Property 9: Completed projects retain Redis state**
  - **Validates: Requirements 4.2**

- [ ]* 12.2 Write property test for completed status persistence
  - **Property 10: Completed status persists**
  - **Validates: Requirements 4.3**

- [ ]* 12.3 Write property test for completed project re-execution
  - **Property 11: Completed projects cannot be re-executed**
  - **Validates: Requirements 4.4**

- [ ] 13. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.
