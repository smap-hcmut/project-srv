# Spec: Event Architecture Standardization

## MODIFIED Requirements

### Requirement: Standardized Event Exchange
**ID**: EA-001

The Project Service MUST use the standardized `smap.events` topic exchange for all SMAP domain events to enable consistent event routing across microservices.

**Previous Behavior**: Events published to `collector.inbound` exchange

**New Behavior**: Events published to `smap.events` exchange with standardized routing keys

#### Scenario: SMAP Events Exchange Declaration
**Given** the Project Service initializes
**When** setting up RabbitMQ infrastructure
**Then** a topic exchange named `smap.events` is declared
**And** the exchange is durable (survives broker restarts)
**And** the exchange is not auto-deleted
**And** the exchange type is `topic`

#### Scenario: Project Created Event Routing
**Given** a project is created
**When** publishing the `project.created` event
**Then** the event is sent to exchange `smap.events`
**And** the routing key is `project.created`
**And** downstream services can bind queues with pattern `project.#`

---

### Requirement: Routing Key Convention
**ID**: EA-002

All events published to `smap.events` exchange MUST follow the routing key pattern `{entity}.{action}` to enable flexible topic-based routing.

#### Scenario: Routing Key Format
**Given** an event is being published
**When** determining the routing key
**Then** the key follows pattern `{entity}.{action}`
**And** entity is lowercase (e.g., `project`, `data`, `analysis`, `job`)
**And** action is past tense (e.g., `created`, `collected`, `finished`, `completed`)

#### Scenario: Supported Routing Keys
**Given** the SMAP events exchange is configured
**When** Project Service publishes events
**Then** the following routing keys are supported:
- `project.created` - New project created
- `project.updated` - Project metadata updated (future)
- `project.deleted` - Project soft-deleted (future)

---

## ADDED Requirements

### Requirement: Event Exchange Migration Strategy
**ID**: EA-003

During transition from legacy `collector.inbound` exchange to standardized `smap.events`, the system MUST support dual publishing controlled by feature flag to ensure zero-downtime migration.

#### Scenario: Feature Flag for Dual Publishing
**Given** the service is configured with environment variable `USE_LEGACY_EXCHANGE=true`
**When** publishing a project created event
**Then** the event is published to `smap.events` exchange
**And** the event is ALSO published to `collector.inbound` exchange (legacy)
**And** both publishes must succeed for operation to succeed

#### Scenario: Migration Complete
**Given** the service is configured with environment variable `USE_LEGACY_EXCHANGE=false`
**When** publishing events
**Then** events are published ONLY to `smap.events`
**And** no events are sent to `collector.inbound`

#### Scenario: Feature Flag Default
**Given** no `USE_LEGACY_EXCHANGE` environment variable is set
**When** loading configuration
**Then** the default value is `false` (new standard exchange only)

---

### Requirement: Event Message Durability
**ID**: EA-004

Event messages MUST be marked as persistent to survive RabbitMQ broker restarts and prevent data loss.

#### Scenario: Persistent Message Publishing
**Given** an event is being published
**When** setting message properties
**Then** the message delivery mode is set to persistent (2)
**And** the message is written to disk by RabbitMQ
**And** the message survives broker restarts

#### Scenario: Durable Exchange and Queues
**Given** exchanges and queues are declared
**When** creating infrastructure
**Then** all exchanges are durable
**And** all queues are durable
**And** infrastructure survives broker restarts

---

### Requirement: Event Publishing Confirmation
**ID**: EA-005

The system MUST use publisher confirms to ensure events are successfully delivered to RabbitMQ before considering the publish operation complete.

#### Scenario: Publisher Confirms Enabled
**Given** a RabbitMQ channel is created for publishing
**When** initializing the publisher
**Then** publisher confirms mode is enabled on the channel
**And** the system waits for broker acknowledgment before returning success

#### Scenario: Publish Confirmation Timeout
**Given** an event is published
**When** RabbitMQ does not confirm within 30 seconds
**Then** the publish operation times out
**And** an error is returned to the caller

#### Scenario: Negative Acknowledgment Handling
**Given** an event is published
**When** RabbitMQ sends a negative acknowledgment (nack)
**Then** the publish operation fails
**And** the error is logged
**And** an error is returned to the caller

---

## Related Capabilities
- **Event Publishing** (`event-publishing/spec.md`): Implements event publishing using this standardized architecture
