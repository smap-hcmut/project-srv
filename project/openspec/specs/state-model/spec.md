# state-model Specification

## Purpose
TBD - created by archiving change 2025-12-15-update-webhook-phase-progress. Update Purpose after archive.
## Requirements
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

