# State Model Capability Specification

## ADDED Requirements

### Requirement: Phase-Based State Fields

The ProjectState SHALL include separate fields for tracking crawl and analyze phase progress.

#### Scenario: Crawl phase state fields

- **Given** a ProjectState struct
- **When** tracking crawl phase progress
- **Then** `CrawlTotal` must store total items to crawl
- **And** `CrawlDone` must store completed crawl items
- **And** `CrawlErrors` must store failed crawl items
- **And** all fields must be int64 type

#### Scenario: Analyze phase state fields

- **Given** a ProjectState struct
- **When** tracking analyze phase progress
- **Then** `AnalyzeTotal` must store total items to analyze
- **And** `AnalyzeDone` must store completed analyze items
- **And** `AnalyzeErrors` must store failed analyze items
- **And** all fields must be int64 type

#### Scenario: State JSON serialization

- **Given** a ProjectState with phase data
- **When** serializing to JSON
- **Then** field names must be snake_case
- **And** all phase fields must be included
- **And** zero values must be serialized

### Requirement: Progress Calculation Methods

The ProjectState SHALL provide methods for calculating progress percentages.

#### Scenario: CrawlProgressPercent calculation

- **Given** a ProjectState with CrawlTotal=100, CrawlDone=80, CrawlErrors=5
- **When** calling CrawlProgressPercent()
- **Then** the result must be 85.0 (done + errors / total \* 100)

#### Scenario: CrawlProgressPercent with zero total

- **Given** a ProjectState with CrawlTotal=0
- **When** calling CrawlProgressPercent()
- **Then** the result must be 0.0
- **And** no division by zero error must occur

#### Scenario: AnalyzeProgressPercent calculation

- **Given** a ProjectState with AnalyzeTotal=50, AnalyzeDone=25, AnalyzeErrors=5
- **When** calling AnalyzeProgressPercent()
- **Then** the result must be 60.0 (done + errors / total \* 100)

#### Scenario: AnalyzeProgressPercent with zero total

- **Given** a ProjectState with AnalyzeTotal=0
- **When** calling AnalyzeProgressPercent()
- **Then** the result must be 0.0
- **And** no division by zero error must occur

#### Scenario: OverallProgressPercent calculation

- **Given** a ProjectState with CrawlProgressPercent=100 and AnalyzeProgressPercent=50
- **When** calling OverallProgressPercent()
- **Then** the result must be 75.0 (crawl/2 + analyze/2)

#### Scenario: OverallProgressPercent with only crawl

- **Given** a ProjectState with CrawlProgressPercent=60 and AnalyzeProgressPercent=0
- **When** calling OverallProgressPercent()
- **Then** the result must be 30.0 (60/2 + 0/2)

### Requirement: Redis Hash Field Constants

The system SHALL define constants for Redis hash field names.

#### Scenario: Crawl phase field constants

- **Given** Redis state storage
- **When** storing crawl phase data
- **Then** `FieldCrawlTotal` must be "crawl_total"
- **And** `FieldCrawlDone` must be "crawl_done"
- **And** `FieldCrawlErrors` must be "crawl_errors"

#### Scenario: Analyze phase field constants

- **Given** Redis state storage
- **When** storing analyze phase data
- **Then** `FieldAnalyzeTotal` must be "analyze_total"
- **And** `FieldAnalyzeDone` must be "analyze_done"
- **And** `FieldAnalyzeErrors` must be "analyze_errors"

#### Scenario: Status field constant

- **Given** Redis state storage
- **When** storing project status
- **Then** `FieldStatus` must be "status"

## ADDED Requirements (Extended State)

### Requirement: ProjectStatus Enum (Simplified)

The ProjectStatus enum SHALL use simplified status values.

#### Scenario: Status enum values

- **Given** a project status
- **When** setting the status
- **Then** valid values must be: INITIALIZING, PROCESSING, DONE, FAILED
- **And** CRAWLING and ANALYZING statuses must NOT be used
- **And** phase information must come from phase data, not status

#### Scenario: Status transition

- **Given** a project in INITIALIZING status
- **When** processing begins
- **Then** status must change to PROCESSING
- **And** phase data must indicate which phase is active

### Requirement: Redis State Repository (Extended)

The Redis state repository SHALL support phase-based state storage and retrieval.

#### Scenario: Store phase state

- **Given** a ProjectState with phase data
- **When** storing to Redis
- **Then** all phase fields must be stored as hash fields
- **And** field names must match constants
- **And** values must be stored as strings (Redis requirement)

#### Scenario: Retrieve phase state

- **Given** a project ID with stored state
- **When** retrieving from Redis
- **Then** all phase fields must be retrieved
- **And** values must be parsed to int64
- **And** missing fields must default to 0

## ADDED Requirements (Deprecation Notes)

### Requirement: Flat Progress Fields (Deprecated)

The flat progress fields (Total, Done, Errors) in ProjectState are deprecated.

#### Scenario: Flat field removal

- **Given** a ProjectState struct
- **When** accessing progress data
- **Then** flat fields (Total, Done, Errors) must NOT be used
- **And** phase-specific fields must be used instead
- **And** backward compatibility may require temporary retention
