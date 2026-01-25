# Spec: Progress Monitoring

## ADDED Requirements

### Requirement: Project Progress API Endpoint
**ID**: PM-001

The system MUST provide an HTTP API endpoint to retrieve real-time project progress from Redis state, enabling users to monitor processing status without querying the source database.

#### Scenario: Retrieve Project Progress
**Given** a project with ID exists
**And** the project state exists in Redis
**When** a GET request is made to `/projects/{id}/progress`
**Then** the response status is 200 OK
**And** the response contains current project progress data

#### Scenario: Progress Response Structure
**Given** a project progress is retrieved
**When** the response is returned
**Then** the response JSON includes field `project_id`
**And** the response includes field `status` with current lifecycle status
**And** the response includes field `total_items` with total count
**And** the response includes field `processed_items` with completed count
**And** the response includes field `failed_items` with error count
**And** the response includes field `progress_percent` as a decimal percentage

#### Scenario: Progress Calculation
**Given** a project has `total=1000` and `done=750`
**When** calculating progress percentage
**Then** `progress_percent` is computed as `(done / total) * 100`
**And** the result is `75.0`
**And** the percentage is rounded to 2 decimal places

#### Scenario: Zero Total Handling
**Given** a project has `total=0` (not yet set by Collector)
**When** calculating progress percentage
**Then** `progress_percent` is `0.0`
**And** no division by zero error occurs

---

### Requirement: Progress API Performance
**ID**: PM-002

The progress API MUST respond within 100ms to support real-time polling from frontend applications.

#### Scenario: Fast Redis Read
**Given** a project state exists in Redis
**When** retrieving progress via the API
**Then** the total response time is less than 100ms
**And** the operation uses `HGETALL` for single-round-trip Redis read

#### Scenario: No Additional Caching
**Given** progress data is requested
**When** the API retrieves state
**Then** data is read directly from Redis (no additional cache layer)
**And** Redis acts as the cache for PostgreSQL

---

### Requirement: Progress API Error Handling
**ID**: PM-003

When Redis state is unavailable or missing, the progress API MUST gracefully degrade by returning project status from PostgreSQL with limited progress information.

#### Scenario: Redis State Not Found
**Given** a project exists in PostgreSQL
**And** no state exists in Redis (expired or not initialized)
**When** retrieving project progress
**Then** the response status is 200 OK
**And** `status` is derived from PostgreSQL project status
**And** `total_items`, `processed_items`, and `failed_items` are `0`
**And** `progress_percent` is `0.0`
**And** a field `state_source` indicates `postgresql` (vs `redis`)

#### Scenario: Redis Connection Error
**Given** Redis is unavailable
**When** retrieving project progress
**Then** the API fallbacks to PostgreSQL data
**And** a warning is logged
**And** the response includes field `state_source: "postgresql"`

#### Scenario: Project Not Found
**Given** a project ID does not exist
**When** retrieving project progress
**Then** the response status is 404 Not Found
**And** an error message indicates the project does not exist

---

### Requirement: Progress API Authorization
**ID**: PM-004

Users MUST only be able to retrieve progress for their own projects, enforced via JWT user ID validation.

#### Scenario: Authorized Access
**Given** a user is authenticated with JWT
**And** the user owns project with ID `proj_abc`
**When** the user requests `/projects/proj_abc/progress`
**Then** the request is authorized
**And** progress data is returned

#### Scenario: Unauthorized Access
**Given** a user is authenticated with JWT
**And** the user does NOT own project with ID `proj_xyz`
**When** the user requests `/projects/proj_xyz/progress`
**Then** the response status is 403 Forbidden
**And** no progress data is returned

#### Scenario: Unauthenticated Access
**Given** no JWT token is provided
**When** requesting project progress
**Then** the response status is 401 Unauthorized

---

### Requirement: Progress Polling Support
**ID**: PM-005

The progress API MUST support high-frequency polling (every 2-5 seconds) without performance degradation.

#### Scenario: Concurrent Progress Requests
**Given** 100 users are polling progress every 3 seconds
**When** handling concurrent requests
**Then** all requests complete within 100ms
**And** Redis connection pool handles load without exhaustion

#### Scenario: Cache-Control Headers
**Given** a progress API response
**When** HTTP headers are set
**Then** `Cache-Control: no-cache` is set
**And** responses are not cached by intermediaries
**And** clients always receive latest state

---

## Related Capabilities
- **Redis State Management** (`redis-state-management/spec.md`): Progress API reads from Redis state
