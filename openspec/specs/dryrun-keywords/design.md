# Design Document

## Overview

The DryRunKeywords feature implements a distributed, event-driven architecture that spans three microservices: Project, Collector, and WebSocket. The design follows the existing architectural patterns in the codebase, using RabbitMQ for asynchronous task distribution, HTTP webhooks for result callbacks, and Redis Pub/Sub for real-time client notifications.

The workflow begins when a user submits keywords to the Project service's HTTP API. The service validates the keywords, generates a unique job identifier, and publishes a task to RabbitMQ. The Collector service consumes this task, performs limited crawling (3 posts per keyword, 5 comments per post), and sends results back via an authenticated webhook. The Project service then publishes these results to Redis, where the WebSocket service picks them up and pushes them to connected clients in real-time.

## Architecture

### Service Responsibilities

**Project Service:**
- Exposes HTTP API endpoint for dry-run requests
- Validates keywords using existing keyword validation logic
- Generates unique job identifiers
- Publishes CrawlRequest messages to RabbitMQ
- Receives webhook callbacks from Collector
- Publishes results to Redis Pub/Sub
- Maintains job state and status

**Collector Service:**
- Consumes dryrun_keyword tasks from RabbitMQ
- Maps generic payloads to platform-specific formats
- Dispatches tasks to YouTube and TikTok workers
- Aggregates crawl results
- Sends authenticated webhook callbacks with retry logic
- Handles platform-specific errors

**WebSocket Service:**
- Subscribes to Redis Pub/Sub pattern `user_noti:*`
- Routes messages to connected clients by user_id
- Maintains WebSocket connection lifecycle
- Handles reconnection to Redis on failures

### Message Flow

```
User → Project API → RabbitMQ → Collector → Webhook → Project → Redis → WebSocket → Client
```

1. User sends POST `/projects/keywords/dry-run` with keywords
2. Project validates, creates job, publishes to `collector.inbound` exchange
3. Collector consumes from queue bound to `crawler.#` pattern
4. Collector crawls platforms (YouTube, TikTok) in parallel
5. Collector POSTs results to `/internal/collector/dryrun/callback`
6. Project validates callback, publishes to Redis channel `user_noti:{user_id}`
7. WebSocket service receives from Redis, pushes to client connections

## Components and Interfaces

### Project Service Components

**Project UseCase Extension (in project/interface.go)**
```go
// Already exists:
// DryRunKeywords(ctx context.Context, sc model.Scope, keywords []string) error

// Implementation will:
// 1. Validate keywords
// 2. Generate job_id
// 3. Publish to RabbitMQ
// 4. Return job_id immediately
```

**HTTP Handler (in project/delivery/http/handler.go)**
```go
// POST /projects/keywords/dry-run
// Uses existing project handler structure
// Calls uc.DryRunKeywords()
```

**Webhook Handler (new package: internal/webhook)**
```go
// POST /internal/collector/dryrun/callback
// Validates authentication
// Publishes to Redis for WebSocket delivery
```

**RabbitMQ Publisher (in project/usecase/publisher.go)**
```go
// Extend existing publisher
func (uc *usecase) publishDryRunTask(ctx context.Context, jobID string, keywords []string) error
```

**Redis Publisher (in project/usecase/publisher.go)**
```go
// Add new method
func (uc *usecase) publishDryRunResult(ctx context.Context, userID string, result CallbackRequest) error
```

### Collector Service Components

**Task Type Extension**
```go
const (
    TaskTypeDryRunKeyword TaskType = "dryrun_keyword"
)
```

**Payload Mapping**
```go
type DryRunKeywordPayload struct {
    Keywords        []string `json:"keywords"`
    LimitPerKeyword int      `json:"limit_per_keyword"`
    IncludeComments bool     `json:"include_comments"`
    MaxComments     int      `json:"max_comments"`
    TimeRange       int      `json:"time_range,omitempty"`
}
```

**Webhook Client**
```go
type WebhookClient interface {
    SendCallback(ctx context.Context, req CallbackRequest) error
}
```

### WebSocket Service Components

**Redis Message Handler**
```go
type DryRunMessage struct {
    Type     string          `json:"type"` // "dryrun_result"
    JobID    string          `json:"job_id"`
    Platform string          `json:"platform"`
    Status   string          `json:"status"`
    Payload  json.RawMessage `json:"payload"`
}
```

## Data Models

### DryRunKeywordsInput (in project/type.go)
```go
type DryRunKeywordsInput struct {
    Keywords []string `json:"keywords" binding:"required,min=1,max=10"`
}

type DryRunKeywordsOutput struct {
    JobID  string `json:"job_id"`
    Status string `json:"status"` // "processing"
}
```

### CrawlRequest (RabbitMQ Message - existing in collector)
```go
type CrawlRequest struct {
    JobID       string         `json:"job_id"`
    TaskType    TaskType       `json:"task_type"` // "dryrun_keyword"
    Payload     map[string]any `json:"payload"`
    TimeRange   int            `json:"time_range,omitempty"`
    Attempt     int            `json:"attempt,omitempty"`
    MaxAttempts int            `json:"max_attempts,omitempty"`
    EmittedAt   time.Time      `json:"emitted_at"`
}
```

### CallbackPayload (in webhook package)
```go
type CallbackRequest struct {
    JobID    string          `json:"job_id" binding:"required"`
    Status   string          `json:"status" binding:"required,oneof=success failed"`
    Platform string          `json:"platform" binding:"required,oneof=youtube tiktok"`
    Payload  CallbackPayload `json:"payload"`
}

type CallbackPayload struct {
    Posts  []Post  `json:"posts,omitempty"`
    Errors []Error `json:"errors,omitempty"`
}

type Post struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    URL         string    `json:"url"`
    Author      string    `json:"author"`
    PublishedAt time.Time `json:"published_at"`
    Comments    []Comment `json:"comments,omitempty"`
}

type Comment struct {
    ID        string    `json:"id"`
    Text      string    `json:"text"`
    Author    string    `json:"author"`
    CreatedAt time.Time `json:"created_at"`
}

type Error struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Keyword string `json:"keyword,omitempty"`
}
```

### RedisMessage (for WebSocket delivery)
```go
type DryRunResultMessage struct {
    Type     string          `json:"type"` // "dryrun_result"
    JobID    string          `json:"job_id"`
    Platform string          `json:"platform"`
    Status   string          `json:"status"`
    Payload  json.RawMessage `json:"payload"`
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Property 1: Valid requests return job_id and processing status
*For any* valid keyword list submitted to the dry-run endpoint, the response should contain a unique job_id and status "processing"
**Validates: Requirements 1.1**

Property 2: Dry-run requests publish to correct RabbitMQ destination
*For any* dry-run request, a CrawlRequest message should be published to exchange `collector.inbound` with routing key `crawler.dryrun_keyword`
**Validates: Requirements 1.2**

Property 3: CrawlRequest contains required dry-run parameters
*For any* CrawlRequest built for dry-run, the payload should include limit_per_keyword=3, include_comments=true, and max_comments=5
**Validates: Requirements 1.3**

Property 4: Invalid keywords are rejected without job creation
*For any* invalid keyword input, the API should return an error response and no job should be created in the database
**Validates: Requirements 1.4**

Property 5: Job IDs are valid UUIDs
*For any* created dry-run job, the job_id should be a valid UUID format
**Validates: Requirements 1.5**

Property 6: Collector recognizes dryrun_keyword task type
*For any* message with task_type "dryrun_keyword", the Collector should accept and process it without rejecting as unknown
**Validates: Requirements 2.1**

Property 7: Post count constraint per keyword
*For any* dry-run crawl result, the number of posts per keyword per platform should not exceed 3
**Validates: Requirements 2.2**

Property 8: Comment count constraint per post
*For any* post in a dry-run result, the number of comments should not exceed 5
**Validates: Requirements 2.3**

Property 9: Successful crawls trigger webhook callbacks
*For any* completed platform crawl, a webhook callback should be sent to the Project service
**Validates: Requirements 2.4**

Property 10: Failed crawls trigger error callbacks
*For any* crawl that encounters errors, an error notification should be sent via webhook
**Validates: Requirements 2.5**

Property 11: Webhook retries use exponential backoff
*For any* failed webhook delivery, subsequent retry attempts should have exponentially increasing delays
**Validates: Requirements 2.6**

Property 12: Webhook requests include authentication
*For any* webhook callback sent by Collector, the request should contain authentication credentials
**Validates: Requirements 2.7**

Property 13: Webhook endpoint validates authentication
*For any* request to the webhook endpoint, only those with valid authentication should be accepted
**Validates: Requirements 3.1**

Property 14: Successful callbacks publish to Redis
*For any* webhook callback with status "success", the posts should be published to Redis
**Validates: Requirements 3.3**

Property 15: Failed callbacks publish errors to Redis
*For any* webhook callback with status "failed", the error information should be published to Redis
**Validates: Requirements 3.4**

Property 16: Redis channel naming convention
*For any* Redis publish operation, the channel name should follow the format `user_noti:{user_id}`
**Validates: Requirements 3.5**

Property 17: WebSocket parses valid messages
*For any* valid dry-run message from Redis, the WebSocket service should successfully parse it without errors
**Validates: Requirements 4.2**

Property 18: WebSocket identifies user connections
*For any* message with a user_id, the WebSocket service should find all active connections for that user
**Validates: Requirements 4.3**

Property 19: Messages fan out to all user connections
*For any* set of active connections for a user, all connections should receive the message
**Validates: Requirements 4.4**

Property 20: Platform information is preserved
*For any* message containing platform-specific results, the platform identifier should be included in the forwarded message
**Validates: Requirements 4.5**

Property 21: Malformed messages don't crash WebSocket service
*For any* malformed message received from Redis, the WebSocket service should log the error and continue operating
**Validates: Requirements 4.6**

Property 22: RabbitMQ messages follow CrawlRequest format
*For any* message published to RabbitMQ by Project service, it should conform to the CrawlRequest schema
**Validates: Requirements 5.1**

Property 23: Webhook payloads follow standard format
*For any* callback sent by Collector, the payload should conform to the CallbackRequest schema
**Validates: Requirements 5.2**

Property 24: Redis messages follow standard format
*For any* message published to Redis by Project service, it should conform to the RedisMessage schema
**Validates: Requirements 5.3**

Property 25: All messages use JSON serialization
*For any* inter-service message, the payload should be valid JSON
**Validates: Requirements 5.4**

Property 26: Errors are logged with context
*For any* error encountered by any service, a log entry should be created with appropriate context information
**Validates: Requirements 6.1**

Property 27: Invalid webhook payloads return 400
*For any* invalid payload sent to the webhook endpoint, the response should be 400 Bad Request
**Validates: Requirements 6.3**

Property 28: WebSocket reconnects to Redis on failure
*For any* Redis connection loss, the WebSocket service should attempt automatic reconnection
**Validates: Requirements 6.4**



## Error Handling

### Project Service Error Handling

**Validation Errors:**
- Invalid keywords: Return 400 Bad Request with validation details
- Missing required fields: Return 400 Bad Request
- Malformed JSON: Return 400 Bad Request

**RabbitMQ Errors:**
- Connection failure: Log error, return 503 Service Unavailable
- Publish failure: Retry with exponential backoff, log critical error if all retries fail

**Webhook Errors:**
- Invalid authentication: Return 401 Unauthorized
- Invalid job_id: Return 404 Not Found
- Malformed payload: Return 400 Bad Request
- Redis publish failure: Log error, return 500 Internal Server Error

**Database Errors:**
- Connection failure: Return 503 Service Unavailable
- Query failure: Log error, return 500 Internal Server Error

### Collector Service Error Handling

**Task Processing Errors:**
- Unknown task type: Log warning, acknowledge message (dead letter)
- Invalid payload: Log error, send error callback
- Platform unavailable: Retry with backoff, send error callback if exhausted

**Crawling Errors:**
- Rate limiting: Implement backoff, retry
- Network timeout: Retry with exponential backoff
- Invalid content: Log warning, continue with partial results

**Webhook Errors:**
- Connection failure: Retry with exponential backoff (max 5 attempts)
- 4xx response: Log error, do not retry
- 5xx response: Retry with exponential backoff
- All retries exhausted: Log critical error, send to dead letter queue

### WebSocket Service Error Handling

**Redis Errors:**
- Connection loss: Attempt reconnection with exponential backoff
- Subscription failure: Log error, retry subscription
- Message parse error: Log error, continue processing other messages

**WebSocket Errors:**
- Client disconnection: Clean up resources, update connection registry
- Send failure: Log error, close connection
- Invalid message format: Log error, do not forward to client

## Testing Strategy

### Unit Testing

**Project Service:**
- Test keyword validation logic with valid and invalid inputs
- Test job creation and UUID generation
- Test RabbitMQ message construction
- Test webhook authentication validation
- Test Redis message formatting
- Test idempotency logic for callbacks

**Collector Service:**
- Test task type recognition
- Test payload mapping for each platform
- Test webhook client retry logic
- Test result aggregation
- Test error callback construction

**WebSocket Service:**
- Test Redis message parsing
- Test connection lookup by user_id
- Test message fan-out logic
- Test reconnection logic

### Property-Based Testing

We will use the `gopter` library for property-based testing in Go. Each property-based test will run a minimum of 100 iterations to ensure comprehensive coverage.

**Property Test Configuration:**
```go
parameters := gopter.DefaultTestParameters()
parameters.MinSuccessfulTests = 100
```

**Property Test Tagging:**
Each property-based test must include a comment tag in this format:
```go
// Feature: dryrun-keywords, Property 1: Valid requests return job_id and processing status
```

**Test Organization:**
- Property tests should be co-located with the code they test
- Use `_property_test.go` suffix for property test files
- Group related properties in the same test file

**Key Properties to Test:**
- Message format compliance (Properties 24, 25, 26, 27)
- Constraint validation (Properties 7, 8)
- Idempotency (Property 15)
- Error handling (Properties 4, 23, 29)
- State transitions (Properties 32, 33, 34, 35)

### Integration Testing

**End-to-End Flow:**
- Test complete flow from API request to WebSocket delivery
- Use test RabbitMQ and Redis instances
- Mock external platform crawlers
- Verify message flow through all services

**Webhook Testing:**
- Test callback delivery with various payloads
- Test authentication rejection
- Test retry behavior with simulated failures
- Test idempotency with duplicate callbacks

**WebSocket Testing:**
- Test message delivery to multiple connected clients
- Test reconnection behavior
- Test message routing by user_id

## Security Considerations

### Authentication

**Webhook Authentication:**
- Use internal API key passed in `X-Internal-Key` header
- Key should be configured via environment variable
- Validate on every webhook request
- Use constant-time comparison to prevent timing attacks

**WebSocket Authentication:**
- Existing JWT-based authentication for WebSocket connections
- User ID extracted from validated JWT token
- No changes required to existing auth flow

### Authorization

**Project Service:**
- Users can only create dry-run jobs for themselves
- Job results are scoped to the creating user
- Webhook endpoint is internal-only (not exposed publicly)

**WebSocket Service:**
- Messages only delivered to authenticated connections
- User ID from Redis channel must match connection user ID

### Data Validation

**Input Validation:**
- Sanitize all user-provided keywords
- Limit keyword count (max 10)
- Limit keyword length (max 100 characters)
- Validate all webhook payload fields

**Output Validation:**
- Validate crawled content before storage
- Sanitize URLs and text content
- Limit result size to prevent memory issues

## Performance Considerations

### Scalability

**Project Service:**
- Stateless design allows horizontal scaling
- RabbitMQ handles load distribution
- Redis Pub/Sub scales with WebSocket instances

**Collector Service:**
- Worker pool for parallel platform crawling
- Configurable concurrency limits
- Queue-based backpressure handling

**WebSocket Service:**
- Connection pooling for Redis
- Efficient message routing with in-memory connection registry
- Graceful handling of high connection counts

### Resource Limits

**Crawl Limits:**
- 3 posts per keyword per platform (hard limit)
- 5 comments per post (hard limit)
- 10 keywords per request (hard limit)
- Total result size capped at 1MB

**Timeout Configuration:**
- API request timeout: 30 seconds
- Webhook callback timeout: 10 seconds
- Crawl operation timeout: 5 minutes per platform
- Job overall timeout: 10 minutes

**Retry Configuration:**
- Webhook retry attempts: 5
- Initial retry delay: 1 second
- Max retry delay: 32 seconds
- Exponential backoff factor: 2

## Monitoring and Observability

### Metrics

**Project Service:**
- Dry-run request rate
- Job creation success/failure rate
- Webhook callback latency
- Redis publish success rate
- Active job count by status

**Collector Service:**
- Task consumption rate
- Crawl success/failure rate by platform
- Webhook delivery success rate
- Retry attempt distribution
- Average crawl duration

**WebSocket Service:**
- Active connection count
- Message delivery rate
- Redis reconnection events
- Message parse error rate

### Logging

**Structured Logging:**
- Use existing zap logger
- Include correlation IDs (job_id, trace_id)
- Log levels: DEBUG, INFO, WARN, ERROR, CRITICAL

**Key Log Events:**
- Job creation
- RabbitMQ publish
- Task consumption
- Crawl start/complete
- Webhook callback attempts
- Redis publish
- WebSocket message delivery
- All errors with full context

### Alerting

**Critical Alerts:**
- Webhook delivery failure rate > 10%
- Job timeout rate > 5%
- Redis connection failures
- RabbitMQ connection failures

**Warning Alerts:**
- Crawl failure rate > 20%
- Average job duration > 8 minutes
- WebSocket reconnection rate > 1/minute

## Deployment Considerations

### Configuration

**Environment Variables:**
```
# Project Service
INTERNAL_API_KEY=<secret>
REDIS_HOST=redis
REDIS_PORT=6379

# Collector Service
PROJECT_WEBHOOK_URL=http://project-service:8080/internal/collector/dryrun/callback
INTERNAL_API_KEY=<secret>
WEBHOOK_RETRY_ATTEMPTS=5
WEBHOOK_RETRY_DELAY=1 # seconds

# WebSocket Service
REDIS_PUBSUB_PATTERN=user_noti:*
```

### Database Migrations

**No database changes required** - Results are delivered directly via WebSocket without persistence

### Rollout Strategy

1. Deploy database migrations
2. Deploy Collector service with new task type support
3. Deploy Project service with webhook endpoint (but don't expose API yet)
4. Deploy WebSocket service with new message type support
5. Enable Project service API endpoint
6. Monitor metrics and logs
7. Gradually increase traffic

### Rollback Plan

1. Disable Project service API endpoint
2. Drain RabbitMQ queue of dryrun_keyword tasks
3. Rollback Project service
4. Rollback Collector service
5. Rollback WebSocket service
6. Rollback database migrations (if necessary)
