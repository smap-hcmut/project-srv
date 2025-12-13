# Redis Messaging Capability Specification

## ADDED Requirements

### Requirement: Topic-Specific Message Publishing
The system SHALL publish messages to topic-specific Redis channels based on message type, replacing the generic single-topic pattern.

#### Scenario: Dry-run job message publishing
- **Given** a dry-run job callback with jobID "job_123" and userID "user_456"
- **When** the webhook handler processes the callback
- **Then** the message must be published to Redis channel "job:job_123:user_456"
- **And** the message must follow the JobMessage structure
- **And** no "type" field must be included in the message payload

#### Scenario: Project progress message publishing  
- **Given** a project progress callback with projectID "proj_789" and userID "user_456"
- **When** the webhook handler processes the callback
- **Then** the message must be published to Redis channel "project:proj_789:user_456"
- **And** the message must follow the ProjectMessage structure
- **And** no "type" field must be included in the message payload

#### Scenario: Topic pattern validation
- **Given** any message publishing operation
- **When** constructing the Redis topic
- **Then** the topic pattern must match "{type}:{id}:{userID}"
- **And** all components must be non-empty strings
- **And** userID must be a valid UUID format

### Requirement: Structured Message Types
The system SHALL use strongly-typed Go structures for all Redis messages, eliminating generic map[string]interface{} usage.

#### Scenario: JobMessage structure compliance
- **Given** a dry-run callback transformation
- **When** creating the JobMessage structure
- **Then** the message must include Platform enum (TIKTOK, YOUTUBE, INSTAGRAM)
- **And** the message must include Status enum (PROCESSING, COMPLETED, FAILED, PAUSED)
- **And** the Batch field must be optional and follow BatchData structure when present
- **And** the Progress field must be optional and follow Progress structure when present
- **And** all JSON tags must be correctly defined for serialization

#### Scenario: ProjectMessage structure compliance
- **Given** a project progress callback transformation
- **When** creating the ProjectMessage structure  
- **Then** the message must include Status enum (PROCESSING, COMPLETED, FAILED, PAUSED)
- **And** the Progress field must be optional and follow Progress structure when present
- **And** all JSON tags must be correctly defined for serialization

#### Scenario: Shared structure consistency
- **Given** any message using Progress structure
- **When** the Progress structure is populated
- **Then** Current must be a non-negative integer
- **And** Total must be a positive integer when provided
- **And** Percentage must be a float64 between 0.0 and 100.0
- **And** ETA must be a non-negative float64 representing minutes
- **And** Errors must be an array of human-readable error message strings

### Requirement: Enum Value Validation
The system SHALL validate and map all enum values with appropriate fallback defaults for unknown values.

#### Scenario: Platform enum mapping
- **Given** a callback with platform value
- **When** mapping to Platform enum
- **Then** "TIKTOK", "tiktok", "TikTok" must map to PlatformTikTok
- **And** "YOUTUBE", "youtube", "YouTube" must map to PlatformYouTube  
- **And** "INSTAGRAM", "instagram", "Instagram" must map to PlatformInstagram
- **And** any unknown value must default to PlatformTikTok
- **And** the mapping must be case-insensitive

#### Scenario: Status enum mapping for dry-run
- **Given** a dry-run callback with status value
- **When** mapping to Status enum
- **Then** "success" must map to StatusCompleted
- **And** "failed" must map to StatusFailed
- **And** any other value must default to StatusProcessing

#### Scenario: MediaType enum mapping
- **Given** content media information
- **When** mapping media type to MediaType enum
- **Then** "video" must map to MediaTypeVideo
- **And** "image" must map to MediaTypeImage
- **And** "audio" must map to MediaTypeAudio
- **And** any unknown value must default to MediaTypeVideo

## MODIFIED Requirements  

### Requirement: Redis Channel Naming (Modified from generic user_noti pattern)
Redis channels SHALL follow topic-specific patterns instead of the previous generic "user_noti:{userID}" pattern.

#### Scenario: Legacy channel pattern deprecation
- **Given** any webhook callback processing
- **When** determining the Redis channel name
- **Then** the system must NOT use "user_noti:{userID}" pattern
- **And** the system must use topic-specific patterns based on message type
- **And** the userID must still be included as the last component for routing

## REMOVED Requirements

### Requirement: Message Type Discrimination Field
The system no longer requires a "type" field in message payloads for message type identification.

#### Scenario: Type field elimination
- **Given** any Redis message being published
- **When** constructing the message payload
- **Then** the message must NOT include a "type" field
- **And** message type must be determined by the Redis topic pattern instead
- **And** the payload must contain only the structured message content

### Requirement: Generic Message Structure
The system no longer supports generic map[string]interface{} message structures for Redis publishing.

#### Scenario: Generic payload elimination
- **Given** any Redis message construction
- **When** building the message payload
- **Then** the system must NOT use map[string]interface{} for message content
- **And** all messages must use typed structs (JobMessage, ProjectMessage)
- **And** JSON marshaling must use struct field tags rather than dynamic map keys