# Webhook Format Capability Specification

## ADDED Requirements

### Requirement: Phase-Based Progress Structure

The system SHALL accept webhook callbacks with phase-based progress data for crawl and analyze phases.

#### Scenario: New format webhook with both phases

- **Given** a webhook callback from Collector Service with new format
- **When** the callback contains `crawl` and `analyze` objects
- **Then** the system must parse both phase objects correctly
- **And** each phase must have `total`, `done`, `errors`, `progress_percent` fields
- **And** the `overall_progress_percent` field must be parsed
- **And** the callback must be processed using new format handler

#### Scenario: New format webhook with crawl phase only

- **Given** a webhook callback during crawl phase
- **When** the callback contains `crawl` object with data and empty `analyze` object
- **Then** the system must process crawl phase data
- **And** analyze phase must default to zero values
- **And** overall progress must reflect crawl-only state

#### Scenario: New format webhook with both phases active

- **Given** a webhook callback during analyze phase
- **When** the callback contains both `crawl` (completed) and `analyze` (in progress) data
- **Then** the system must preserve crawl phase final state
- **And** the system must update analyze phase progress
- **And** overall progress must be calculated from both phases

### Requirement: PhaseProgress Data Structure

The system SHALL define a PhaseProgress structure for tracking individual phase progress.

#### Scenario: PhaseProgress field validation

- **Given** a PhaseProgress object in webhook payload
- **When** validating the structure
- **Then** `total` must be a non-negative integer
- **And** `done` must be a non-negative integer not exceeding `total`
- **And** `errors` must be a non-negative integer
- **And** `progress_percent` must be a float64 between 0.0 and 100.0

#### Scenario: PhaseProgress JSON serialization

- **Given** a PhaseProgress struct
- **When** serializing to JSON
- **Then** field names must be snake_case (`progress_percent`)
- **And** all numeric fields must be serialized correctly
- **And** zero values must be included in output

### Requirement: Overall Progress Calculation

The system SHALL calculate overall progress from phase data.

#### Scenario: Overall progress with both phases

- **Given** crawl progress at 100% and analyze progress at 50%
- **When** calculating overall progress
- **Then** overall progress must be 75% (equal weight: 50% + 25%)

#### Scenario: Overall progress with crawl only

- **Given** crawl progress at 60% and analyze not started (0%)
- **When** calculating overall progress
- **Then** overall progress must be 30% (60% / 2)

#### Scenario: Overall progress at completion

- **Given** both crawl and analyze at 100%
- **When** calculating overall progress
- **Then** overall progress must be 100%

## ADDED Requirements (Extended Request)

### Requirement: ProgressCallbackRequest Structure (Extended)

The ProgressCallbackRequest SHALL be extended to support both old and new webhook formats.

#### Scenario: New format request parsing

- **Given** a webhook request with new format
- **When** parsing the request body
- **Then** `project_id`, `user_id`, `status` must be required
- **And** `crawl` object must be parsed to PhaseProgress
- **And** `analyze` object must be parsed to PhaseProgress
- **And** `overall_progress_percent` must be parsed

#### Scenario: Old format request parsing (backward compat)

- **Given** a webhook request with old format
- **When** parsing the request body
- **Then** `project_id`, `user_id`, `status` must be required
- **And** `total`, `done`, `errors` fields must be parsed
- **And** `crawl` and `analyze` objects may be empty/zero

### Requirement: Format Detection Logic (New)

The webhook handler SHALL detect and route requests based on format.

#### Scenario: Detect new format

- **Given** a webhook request
- **When** `crawl.total > 0` OR `analyze.total > 0`
- **Then** the request must be processed as new format
- **And** new format handler must be invoked

#### Scenario: Detect old format

- **Given** a webhook request
- **When** `crawl.total == 0` AND `analyze.total == 0`
- **And** `total > 0` (old format field)
- **Then** the request must be processed as old format
- **And** old format handler must be invoked with deprecation warning

## ADDED Requirements (Deprecation Notes)

### Requirement: Single-Phase Progress Model (Deprecated)

The flat progress model with single `total`, `done`, `errors` fields is deprecated.

#### Scenario: Old format deprecation

- **Given** a webhook request using old format
- **When** processing the request
- **Then** the system must log a deprecation warning
- **And** the request must still be processed successfully
- **And** old format support may be removed in future versions

#### Scenario: Status-based phase inference (Deprecated)

- **Given** an old format request with status "CRAWLING" or "ANALYZING"
- **When** processing the request
- **Then** the system must NOT infer phase from status
- **And** the system must use explicit phase data from new format
