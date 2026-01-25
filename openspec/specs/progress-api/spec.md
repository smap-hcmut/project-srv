# progress-api Specification

## Purpose
TBD - created by archiving change 2025-12-15-update-webhook-phase-progress. Update Purpose after archive.
## Requirements
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

