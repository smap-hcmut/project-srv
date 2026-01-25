# Progress API Capability Specification

## ADDED Requirements

### Requirement: Phase-Based Progress Response

The GET /projects/:id/progress API SHALL return phase-based progress data.

#### Scenario: Progress response with both phases

- **Given** a project with crawl and analyze progress data
- **When** calling GET /projects/:id/progress
- **Then** response must include `project_id` string
- **And** response must include `status` string
- **And** response must include `crawl` object with phase progress
- **And** response must include `analyze` object with phase progress
- **And** response must include `overall_progress_percent` float

#### Scenario: Progress response structure

- **Given** a successful progress API call
- **When** receiving the response
- **Then** HTTP status must be 200 OK
- **And** Content-Type must be application/json
- **And** response body must match ProjectProgressResponse schema

#### Scenario: Phase progress object structure

- **Given** a `crawl` or `analyze` object in response
- **When** parsing the object
- **Then** `total` must be present as integer
- **And** `done` must be present as integer
- **And** `errors` must be present as integer
- **And** `progress_percent` must be present as float

### Requirement: PhaseProgressResponse Structure

The API SHALL use PhaseProgressResponse for phase data in responses.

#### Scenario: PhaseProgressResponse fields

- **Given** a PhaseProgressResponse struct
- **When** serializing to JSON
- **Then** `total` must serialize as integer
- **And** `done` must serialize as integer
- **And** `errors` must serialize as integer
- **And** `progress_percent` must serialize as float with snake_case

#### Scenario: PhaseProgressResponse from state

- **Given** a ProjectState with phase data
- **When** building PhaseProgressResponse
- **Then** Total must come from state phase total
- **And** Done must come from state phase done
- **And** Errors must come from state phase errors
- **And** ProgressPercent must be calculated from state method

### Requirement: ProjectProgressResponse Structure

The API SHALL use ProjectProgressResponse for the full progress response.

#### Scenario: ProjectProgressResponse fields

- **Given** a ProjectProgressResponse struct
- **When** serializing to JSON
- **Then** `project_id` must be string
- **And** `status` must be string
- **And** `crawl` must be PhaseProgressResponse object
- **And** `analyze` must be PhaseProgressResponse object
- **And** `overall_progress_percent` must be float

#### Scenario: ProjectProgressResponse from state

- **Given** a ProjectState retrieved from Redis
- **When** building ProjectProgressResponse
- **Then** ProjectID must come from request parameter
- **And** Status must come from state.Status
- **And** Crawl must be built from crawl phase data
- **And** Analyze must be built from analyze phase data
- **And** OverallProgressPercent must come from state.OverallProgressPercent()

## ADDED Requirements (Extended Handlers)

### Requirement: GetProgress Handler (Extended)

The GetProgress handler SHALL return phase-based progress data.

#### Scenario: GetProgress success flow

- **Given** a valid project ID in request
- **When** handling GET /projects/:id/progress
- **Then** handler must extract project ID from URL parameter
- **And** handler must call state usecase to get state
- **And** handler must build ProjectProgressResponse from state
- **And** handler must return 200 OK with response body

#### Scenario: GetProgress not found

- **Given** a non-existent project ID
- **When** handling GET /projects/:id/progress
- **Then** handler must return 404 Not Found
- **And** response must include error message

#### Scenario: GetProgress state retrieval error

- **Given** a Redis connection error
- **When** handling GET /projects/:id/progress
- **Then** handler must return 500 Internal Server Error
- **And** error must be logged with context

### Requirement: Swagger Documentation (Updated)

The API documentation SHALL reflect phase-based progress response.

#### Scenario: Swagger response schema

- **Given** Swagger documentation
- **When** documenting GET /projects/:id/progress
- **Then** response schema must show ProjectProgressResponse
- **And** nested PhaseProgressResponse must be documented
- **And** example response must show phase-based structure

#### Scenario: Swagger field descriptions

- **Given** Swagger documentation for progress endpoint
- **When** describing response fields
- **Then** each field must have description
- **And** data types must be accurate
- **And** required fields must be marked

## WebSocket Message Requirements

### Requirement: Phase-Based WebSocket Messages

WebSocket messages SHALL include phase-based progress data.

#### Scenario: project_progress message format

- **Given** a progress update to publish
- **When** building WebSocket message
- **Then** `type` must be "project_progress"
- **And** `payload.project_id` must be string
- **And** `payload.status` must be string
- **And** `payload.crawl` must be phase progress object
- **And** `payload.analyze` must be phase progress object
- **And** `payload.overall_progress_percent` must be float

#### Scenario: project_completed message format

- **Given** a completion event to publish
- **When** building WebSocket message
- **Then** `type` must be "project_completed"
- **And** `payload` must follow same structure as progress message
- **And** `payload.status` must be "DONE" or "FAILED"

#### Scenario: Message type determination

- **Given** a progress callback with status
- **When** determining message type
- **Then** status "DONE" must result in "project_completed" type
- **And** status "FAILED" must result in "project_completed" type
- **And** other statuses must result in "project_progress" type

### Requirement: Redis Pub/Sub Message Structure

Published Redis messages SHALL use phase-based structure.

#### Scenario: Redis message payload

- **Given** a progress update to publish to Redis
- **When** building the message
- **Then** message must be JSON serializable
- **And** message must include type and payload
- **And** payload must match WebSocket message format

#### Scenario: Redis channel pattern

- **Given** a progress update for project "proj_123" and user "user_456"
- **When** publishing to Redis
- **Then** channel must be "project:proj_123:user_456"
- **And** message must be published successfully
