# Spec: Redis State Management

## ADDED Requirements

### Requirement: Redis State Database Configuration
**ID**: RSM-001

The system MUST maintain a dedicated Redis database (DB 1) for SMAP project state management, separate from cache operations (DB 0), to prevent cache eviction from deleting critical state tracking data.

#### Scenario: Dedicated State Database Connection
**Given** the service initializes Redis connections
**When** connecting to Redis for state management
**Then** a dedicated client connects to Redis DB 1
**And** the state database uses `noeviction` policy
**And** cache operations continue using DB 0 with `volatile-lru` policy

#### Scenario: State Database Configuration
**Given** environment variables are loaded
**When** Redis configuration is parsed
**Then** `REDIS_STATE_DB` environment variable defines the state database number
**And** the default state database is 1
**And** the state database is separate from the cache database

---

### Requirement: Project State Initialization
**ID**: RSM-002

When a project is created, the system MUST initialize a Redis Hash structure containing the project's lifecycle state with a 7-day TTL to enable distributed state tracking across microservices.

#### Scenario: Initialize Project State on Creation
**Given** a new project is created in PostgreSQL
**When** initializing project state in Redis
**Then** a Redis Hash is created at key `smap:proj:{project_id}`
**And** the Hash contains field `status` with value `INITIALIZING`
**And** the Hash contains field `total` with value `0`
**And** the Hash contains field `done` with value `0`
**And** the Hash contains field `errors` with value `0`
**And** the key has a TTL of 604800 seconds (7 days)

#### Scenario: State Initialization Uses Atomic Pipeline
**Given** project state is being initialized
**When** setting multiple Hash fields and TTL
**Then** all operations are executed in a single Redis pipeline
**And** operations are atomic (all succeed or all fail)

#### Scenario: State Initialization Failure Handling
**Given** a project is created in PostgreSQL
**When** Redis state initialization fails due to connection error
**Then** the error is logged at ERROR level
**And** project creation continues (state can be initialized lazily)
**And** the failure does not prevent event publishing

---

### Requirement: Atomic State Counter Updates
**ID**: RSM-003

The system MUST provide atomic increment operations for project state counters (`done`, `errors`) to prevent race conditions when multiple workers update the same project concurrently.

#### Scenario: Atomic Done Counter Increment
**Given** an item processing completes successfully
**When** incrementing the `done` counter
**Then** Redis `HINCRBY` command is used for atomic increment
**And** the operation returns the new value after increment
**And** no race condition occurs with concurrent increments

#### Scenario: Atomic Error Counter Increment
**Given** an item processing fails
**When** marking the item as error
**Then** both `done` and `errors` counters are incremented atomically
**And** the total processed count includes errors

#### Scenario: Completion Detection After Increment
**Given** the `done` counter is incremented atomically
**When** the new `done` value equals or exceeds `total`
**And** the current status is not `DONE`
**Then** the status is updated to `DONE`
**And** completion is signaled to trigger `job.completed` event
**And** only the first worker to reach completion triggers the event

---

### Requirement: Project State Status Updates
**ID**: RSM-004

The system MUST support updating project status as it transitions through lifecycle stages (`INITIALIZING` → `CRAWLING` → `PROCESSING` → `DONE`).

#### Scenario: Update Project Status
**Given** a project exists in Redis state
**When** updating the project status to `CRAWLING`
**Then** the `status` field in the Redis Hash is updated
**And** the TTL is refreshed to 7 days
**And** the operation is idempotent (repeated updates with same status succeed)

#### Scenario: Status Transition Validation
**Given** a project state exists
**When** updating status from `INITIALIZING` to `DONE` directly
**Then** the update succeeds (no strict state machine enforcement at this layer)
**And** validation is handled by business logic layer if needed

---

### Requirement: Project State Retrieval
**ID**: RSM-005

The system MUST provide read access to project state for real-time progress monitoring with sub-100ms latency.

#### Scenario: Retrieve Complete Project State
**Given** a project state exists in Redis
**When** retrieving the project state by project ID
**Then** all Hash fields are returned in a single operation using `HGETALL`
**And** the operation completes within 100ms
**And** the returned state includes status, total, done, and errors

#### Scenario: Project State Not Found
**Given** no state exists for a project ID
**When** retrieving the project state
**Then** Redis returns empty result
**And** the system returns a "state not found" indicator
**And** the caller can fallback to PostgreSQL or initialize state lazily

---

### Requirement: State Key TTL Management
**ID**: RSM-006

Project state keys MUST have a 7-day TTL to prevent indefinite memory growth while supporting long-running projects.

#### Scenario: TTL Set on State Initialization
**Given** a project state is initialized
**When** creating the Redis Hash
**Then** a 7-day (604800 seconds) TTL is set
**And** the key auto-expires after 7 days if not accessed

#### Scenario: TTL Refresh on State Updates
**Given** a project state exists with remaining TTL
**When** the state is updated (status change, counter increment)
**Then** the TTL is refreshed to 7 days
**And** active projects never expire

#### Scenario: Expired State Handling
**Given** a project state key has expired
**When** attempting to retrieve the state
**Then** Redis returns no data
**And** the system can reinitialize state from PostgreSQL if needed

---

## Related Capabilities
- **Event Publishing** (`event-publishing/spec.md`): State initialization triggers before event publishing
- **Progress Monitoring** (`progress-monitoring/spec.md`): Progress API reads from Redis state
