# Spec: Event Publishing

## ADDED Requirements

### Requirement: Project Created Event Publishing
**ID**: EP-001

When a project is successfully created and saved to PostgreSQL, the system MUST publish a `project.created` event to the `smap.events` exchange to notify downstream services (Collector Service) to begin data collection.

#### Scenario: Publish Event on Successful Project Creation
**Given** a project is created and saved to PostgreSQL
**And** Redis state is initialized
**When** publishing the `project.created` event
**Then** an event message is sent to RabbitMQ exchange `smap.events`
**And** the routing key is `project.created`
**And** the message contains a JSON payload with project details

#### Scenario: Event Message Structure
**Given** a project created event is being published
**When** the event payload is serialized
**Then** the message includes field `event_id` with a UUID v4 value
**And** the message includes field `timestamp` in ISO 8601 format
**And** the message includes field `payload` with project data
**And** the `payload` contains `project_id`, `brand_name`, `brand_keywords`, `competitor_names`, `competitor_keywords_map`, and `date_range`

#### Scenario: Event Publishing Failure
**Given** project creation succeeds in PostgreSQL and Redis
**When** RabbitMQ publishing fails due to connection error
**Then** the error is logged at ERROR level
**And** an error is returned to the caller
**And** the project remains in PostgreSQL (no rollback)
**And** the user is informed that event publishing failed

---

### Requirement: Event Message Serialization
**ID**: EP-002

Event messages MUST be serialized to JSON format with a standardized schema to ensure interoperability between microservices.

#### Scenario: Project Created Event Schema
**Given** a project created event is being published
**When** the event is serialized to JSON
**Then** the JSON structure matches the following schema:
```json
{
  "event_id": "string (UUID v4)",
  "timestamp": "string (ISO 8601)",
  "payload": {
    "project_id": "string (UUID)",
    "brand_name": "string",
    "brand_keywords": ["string"],
    "competitor_names": ["string"],
    "competitor_keywords_map": {
      "competitor_name": ["keyword1", "keyword2"]
    },
    "date_range": {
      "from": "string (YYYY-MM-DD)",
      "to": "string (YYYY-MM-DD)"
    }
  }
}
```

#### Scenario: Event Serialization Error Handling
**Given** a project created event is being published
**When** JSON serialization fails
**Then** the error is logged
**And** publishing is aborted
**And** the error is returned to the caller

---

### Requirement: Event Publishing Idempotency
**ID**: EP-003

Event messages MUST include a unique `event_id` to enable downstream services to deduplicate events in case of message redelivery.

#### Scenario: Unique Event ID Generation
**Given** a project created event is being published
**When** generating the event ID
**Then** a UUID v4 is generated
**And** the event ID is unique across all events
**And** the event ID is included in the message payload

#### Scenario: Event ID for Deduplication
**Given** an event is published twice due to retry
**When** a downstream service receives the duplicate event
**Then** the service can use the `event_id` to detect the duplicate
**And** the duplicate event is discarded

---

### Requirement: Event Publishing Resilience
**ID**: EP-004

The system MUST log event publishing failures and continue operation without blocking project creation to maintain high availability.

#### Scenario: Publish Failure Does Not Rollback Project
**Given** a project is created in PostgreSQL
**And** Redis state is initialized
**When** event publishing fails
**Then** the project remains in PostgreSQL
**And** Redis state remains initialized
**And** the error is logged at ERROR level
**And** an error is returned to the user indicating partial failure

#### Scenario: Event Publishing Timeout
**Given** event publishing is in progress
**When** RabbitMQ does not acknowledge within 30 seconds
**Then** the publish operation times out
**And** the timeout error is logged
**And** the user is informed of the timeout

---

### Requirement: Event Publishing Monitoring
**ID**: EP-005

The system MUST emit metrics for event publishing to enable monitoring of event delivery success rates and latency.

#### Scenario: Success Metrics
**Given** an event is successfully published
**When** the publish operation completes
**Then** a counter metric `project_created_events_published_total` is incremented
**And** a histogram metric `event_publish_duration_seconds` records the duration

#### Scenario: Failure Metrics
**Given** event publishing fails
**When** the error is logged
**Then** a counter metric `project_created_events_failed_total` is incremented
**And** the failure reason is included in metric labels

---

## Related Capabilities
- **Redis State Management** (`redis-state-management/spec.md`): State must be initialized before event publishing
- **Event Architecture** (`event-architecture/spec.md`): Defines the `smap.events` exchange and routing keys
