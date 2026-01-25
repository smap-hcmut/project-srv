# Webhook Processing Capability Specification

## ADDED Requirements

### Requirement: Callback Transformation Functions
The system SHALL provide transformation functions to convert external callback formats to internal Redis message structures.

#### Scenario: Dry-run callback transformation
- **Given** a CallbackRequest from the crawler service
- **When** transforming to JobMessage for Redis publishing
- **Then** the transformation must map req.Platform to Platform enum
- **And** the transformation must map req.Status to Status enum
- **And** the transformation must convert req.Payload.Content to BatchData.ContentList
- **And** the transformation must convert req.Payload.Errors to Progress.Errors string array
- **And** the transformation must handle empty or nil content gracefully
- **And** the transformation must preserve all essential content metadata

#### Scenario: Project progress callback transformation
- **Given** a ProgressCallbackRequest from the collector service
- **When** transforming to ProjectMessage for Redis publishing
- **Then** the transformation must map req.Status to Status enum
- **And** the transformation must calculate percentage from req.Done and req.Total
- **And** the transformation must convert req.Errors count to Progress.Errors string array
- **And** the transformation must set Progress.Current to req.Done
- **And** the transformation must set Progress.Total to req.Total
- **And** the transformation must handle zero total gracefully (percentage = 0.0)

#### Scenario: Content item transformation
- **Given** a Content object from a dry-run callback
- **When** transforming to ContentItem structure
- **Then** all core fields must be preserved (ID, Text, Permalink, PublishedAt)
- **And** Author information must be mapped to AuthorInfo structure
- **And** Interaction metrics must be mapped to MetricsInfo structure  
- **And** Media information must be mapped to MediaInfo structure if present
- **And** Time fields must be formatted as ISO 8601 strings
- **And** Optional fields must be handled gracefully (nil-safe)

### Requirement: Error Message Transformation
The system SHALL provide meaningful error message transformation for different callback error formats.

#### Scenario: Dry-run error transformation
- **Given** an array of Error objects with Code, Message, and optional Keyword
- **When** transforming to Progress.Errors string array
- **Then** each error must be formatted as "[{Code}] {Message}"
- **And** if Keyword is present, it must be appended as " (keyword: {Keyword})"
- **And** empty error arrays must result in empty string arrays
- **And** malformed error objects must be logged but not break transformation

#### Scenario: Project error count transformation  
- **Given** a numeric error count from progress callback
- **When** transforming to Progress.Errors string array
- **Then** zero errors must result in empty string array
- **And** non-zero errors must create a single descriptive message
- **And** the message must be "Processing encountered {count} errors"
- **And** the error count must be accurately reflected in the message

### Requirement: Job Mapping Integration
The system SHALL integrate with existing Redis job mapping for user and project lookup while supporting new topic patterns.

#### Scenario: Dry-run job mapping lookup
- **Given** a CallbackRequest with JobID but no UserID
- **When** processing the dry-run callback
- **Then** the system must lookup UserID from Redis job mapping using JobID
- **And** the system must lookup ProjectID from Redis job mapping using JobID  
- **And** if lookup fails, the callback processing must fail with appropriate error
- **And** successful lookup must be logged with job_id, user_id, and project_id
- **And** the UserID must be used in the new topic pattern "job:{JobID}:{UserID}"

#### Scenario: Project progress direct routing
- **Given** a ProgressCallbackRequest with ProjectID and UserID
- **When** processing the project progress callback
- **Then** no additional lookup must be required
- **And** the ProjectID and UserID must be used directly in topic pattern
- **And** the topic pattern must be "project:{ProjectID}:{UserID}"

## MODIFIED Requirements

### Requirement: Webhook Handler Message Publishing (Modified from generic to topic-specific)
Webhook handlers SHALL publish to topic-specific Redis channels instead of generic user notification channels.

#### Scenario: HandleDryRunCallback topic publishing
- **Given** a processed dry-run callback with successful transformation
- **When** publishing the JobMessage to Redis
- **Then** the Redis channel must be "job:{JobID}:{UserID}"
- **And** the message must be the transformed JobMessage structure
- **And** the publishing must NOT use "user_noti:{UserID}" pattern
- **And** successful publishing must be logged with channel, job_id, platform, and status

#### Scenario: HandleProgressCallback topic publishing
- **Given** a processed progress callback with successful transformation  
- **When** publishing the ProjectMessage to Redis
- **Then** the Redis channel must be "project:{ProjectID}:{UserID}"
- **And** the message must be the transformed ProjectMessage structure
- **And** the publishing must NOT use "user_noti:{UserID}" pattern
- **And** successful publishing must be logged with channel, project_id, and status

### Requirement: Error Logging Context (Enhanced for new message formats)
Error logging SHALL provide comprehensive context for debugging new message transformation and publishing processes.

#### Scenario: Transformation failure logging
- **Given** a callback transformation that fails
- **When** logging the transformation error
- **Then** the log must include the original callback data structure
- **And** the log must include the specific transformation step that failed
- **And** the log must include error context for debugging
- **And** sensitive information must be excluded from logs

#### Scenario: Publishing failure logging  
- **Given** a Redis publishing operation that fails
- **When** logging the publishing error
- **Then** the log must include the target Redis channel
- **And** the log must include the message type (JobMessage/ProjectMessage)
- **And** the log must include the message size for debugging
- **And** the log must include Redis client error details

## REMOVED Requirements

### Requirement: Generic Message Construction
The system no longer constructs generic messages with type discrimination fields for webhook processing.

#### Scenario: Legacy message structure elimination
- **Given** any webhook callback processing
- **When** constructing Redis messages
- **Then** the system must NOT create messages with "type" and "payload" structure
- **And** the system must NOT use map[string]interface{} for message construction
- **And** all message construction must use typed structs