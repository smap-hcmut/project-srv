# DryRunKeywords Feature Specification

## Overview

This specification defines the implementation of the DryRunKeywords feature, which enables users to preview social media content for a set of keywords before committing to a full project crawl. The feature implements a real-time, asynchronous workflow across three microservices: Project, Collector, and WebSocket.

**Key Simplifications:**
- No database persistence - results delivered directly via WebSocket
- No caching layer - real-time delivery only
- Integrated into existing `project` package - no separate domain
- Webhook handler in dedicated `webhook` package for clean separation

## Architecture Summary

```
User → Project API → RabbitMQ → Collector → Webhook → Project → Redis → WebSocket → Client
```

### Service Responsibilities

- **Project Service**: HTTP API, keyword validation, job management, webhook receiver, Redis publisher
- **Collector Service**: Task consumer, web crawler, webhook client with retry logic
- **WebSocket Service**: Redis subscriber, real-time message delivery to connected clients

## Key Design Decisions

1. **Asynchronous Processing**: Uses RabbitMQ for task distribution to avoid HTTP timeout issues
2. **Webhook Callbacks**: Collector sends results back via authenticated HTTP webhooks with retry logic
3. **Real-time Updates**: Redis Pub/Sub enables instant delivery to WebSocket clients
4. **Idempotency**: Callbacks are tracked by (job_id, platform) to prevent duplicate processing
5. **Limited Crawling**: Hard limits of 3 posts per keyword and 5 comments per post for preview purposes

## Code Conventions Applied

### Directory Structure
```
project/
├── internal/
│   ├── project/
│   │   ├── type.go              # Add DryRunKeywordsInput/Output
│   │   ├── interface.go         # DryRunKeywords already defined
│   │   ├── usecase/
│   │   │   ├── project.go       # Implement DryRunKeywords
│   │   │   └── new.go           # Add Producer dependency
│   │   └── delivery/
│   │       ├── http/
│   │       │   ├── handler.go   # Add dry-run endpoint
│   │       │   └── routes.go    # Register route
│   │       └── rabbitmq/
│   │           ├── constants.go # Add exchange and routing key
│   │           ├── presenters.go # Add DryRunCrawlRequest
│   │           └── producer/
│   │               ├── new.go      # Add PublishDryRunTask interface
│   │               ├── producer.go # Implement PublishDryRunTask
│   │               └── common.go   # Initialize dryRunWriter
│   ├── webhook/
│   │   ├── type.go              # Callback types
│   │   └── handler.go           # Webhook handler + Redis publish
│   └── httpserver/
│       ├── handler.go           # Init RabbitMQ producer, register webhook
│       └── server.go            # Add Redis client

collector/
├── internal/
│   ├── models/
│   │   └── task.go              # Add TaskTypeDryRunKeyword
│   ├── dispatcher/
│   │   └── usecase/
│   │       ├── dispatch_uc.go   # Add dryrun handling
│   │       └── util.go          # Add payload mapping
│   └── webhook/
│       └── client.go            # HTTP client with retry

websocket/
└── internal/
    ├── websocket/
    │   └── types.go             # Add DryRunResult type
    └── redis/
        └── subscriber.go        # Add dryrun message handling
```

### Naming Conventions

**Types**: PascalCase (e.g., `DryRunRequest`, `CallbackPayload`)
**Interfaces**: PascalCase with descriptive names (e.g., `DryRunUseCase`, `WebhookClient`)
**Methods**: PascalCase for exported, camelCase for private (e.g., `CreateDryRun`, `mapPayload`)
**Constants**: PascalCase with prefix (e.g., `TaskTypeDryRunKeyword`)
**Files**: snake_case (e.g., `http_client.go`, `dispatch_uc.go`)

### Error Handling Pattern

```go
// Define domain errors
var (
    ErrJobNotFound = errors.New("dry-run job not found")
    ErrInvalidKeywords = errors.New("invalid keywords")
)

// Log and wrap errors
if err != nil {
    uc.l.Errorf(ctx, "internal.dryrun.usecase.CreateDryRun: %v", err)
    return DryRunResponse{}, err
}
```

### Logging Pattern

```go
// Use structured logging with context
uc.l.Infof(ctx, "Creating dry-run job for user %s with %d keywords", userID, len(keywords))
uc.l.Errorf(ctx, "Failed to publish to RabbitMQ: %v", err)
```

### Dependency Injection Pattern

```go
type usecase struct {
    l          log.Logger
    repo       DryRunRepository
    mqPublisher RabbitMQPublisher
    redisPublisher RedisPublisher
}

func New(l log.Logger, repo DryRunRepository, mqPub RabbitMQPublisher, redisPub RedisPublisher) DryRunUseCase {
    return &usecase{
        l:              l,
        repo:           repo,
        mqPublisher:    mqPub,
        redisPublisher: redisPub,
    }
}
```

### HTTP Handler Pattern

```go
func (h *handler) CreateDryRun(c *gin.Context) {
    var req DryRunRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err)
        return
    }

    sc := scope.Get(c)
    resp, err := h.uc.CreateDryRun(c.Request.Context(), sc, req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err)
        return
    }

    response.Success(c, http.StatusOK, resp)
}
```

### RabbitMQ Publishing Pattern

```go
func (p *publisher) PublishDryRunTask(ctx context.Context, req CrawlRequest) error {
    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }

    return p.channel.PublishWithContext(ctx, rabbitmq.PublishArgs{
        Exchange:   "collector.inbound",
        RoutingKey: "crawler.dryrun_keyword",
        Mandatory:  false,
        Immediate:  false,
        Msg: rabbitmq.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    })
}
```

### Redis Publishing Pattern

```go
func (p *publisher) PublishDryRunResult(ctx context.Context, userID string, result DryRunResult) error {
    channel := fmt.Sprintf("user_noti:%s", userID)
    
    msg := RedisMessage{
        Type:    "dryrun_result",
        Payload: result,
    }
    
    body, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    
    return p.client.Publish(ctx, channel, body).Err()
}
```

### Retry Logic Pattern

```go
func (c *client) SendCallback(ctx context.Context, req CallbackRequest) error {
    maxRetries := 5
    delay := 1 * time.Second
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := c.doSend(ctx, req)
        if err == nil {
            return nil
        }
        
        if attempt < maxRetries {
            c.l.Warnf(ctx, "Webhook attempt %d failed, retrying in %v: %v", 
                attempt+1, delay, err)
            time.Sleep(delay)
            delay *= 2 // Exponential backoff
            if delay > 32*time.Second {
                delay = 32 * time.Second
            }
        }
    }
    
    return fmt.Errorf("webhook failed after %d attempts", maxRetries)
}
```

## Data Storage

**No database persistence required** - Results are delivered directly to WebSocket clients without caching or storage. This keeps the implementation simple and focused on real-time delivery.

## API Endpoints

### POST /projects/keywords/dry-run
**Request:**
```json
{
  "keywords": ["keyword1", "keyword2"]
}
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing"
}
```

### POST /internal/collector/dryrun/callback
**Headers:**
```
X-Internal-Key: <secret>
```

**Request:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "platform": "youtube",
  "payload": {
    "posts": [
      {
        "id": "video123",
        "title": "Video Title",
        "url": "https://youtube.com/watch?v=...",
        "author": "Channel Name",
        "published_at": "2024-01-01T00:00:00Z",
        "comments": [...]
      }
    ]
  }
}
```

## Message Formats

### RabbitMQ CrawlRequest
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_type": "dryrun_keyword",
  "payload": {
    "keywords": ["keyword1", "keyword2"],
    "limit_per_keyword": 3,
    "include_comments": true,
    "max_comments": 5
  },
  "time_range": 0,
  "attempt": 1,
  "max_attempts": 3,
  "emitted_at": "2024-01-01T00:00:00Z"
}
```

### Redis Message
```json
{
  "type": "dryrun_result",
  "payload": {
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "platform": "youtube",
    "status": "success",
    "posts": [...]
  }
}
```

## Configuration

### Project Service
```env
INTERNAL_API_KEY=<secret>
REDIS_HOST=redis
REDIS_PORT=6379
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
```

### Collector Service
```env
PROJECT_WEBHOOK_URL=http://project-service:8080/internal/collector/dryrun/callback
INTERNAL_API_KEY=<secret>
WEBHOOK_RETRY_ATTEMPTS=5
WEBHOOK_RETRY_DELAY=1
```

### WebSocket Service
```env
REDIS_PUBSUB_PATTERN=user_noti:*
REDIS_HOST=redis
REDIS_PORT=6379
```

## Testing Strategy

### Property-Based Testing
- Use `gopter` library with minimum 100 iterations per property
- Tag each test with: `// Feature: dryrun-keywords, Property X: <description>`
- Focus on message format compliance, constraints, idempotency, and state transitions

### Unit Testing
- Test individual functions and methods in isolation
- Mock external dependencies (database, RabbitMQ, Redis, HTTP clients)
- Focus on business logic, validation, and error handling

### Integration Testing
- Test end-to-end flow with real RabbitMQ and Redis instances
- Mock external platform crawlers
- Verify message flow through all services
- Test error scenarios and retry behavior

## Implementation Order

1. Extend Project types and usecase (Project)
2. Add RabbitMQ publisher (Project)
3. Add HTTP endpoint (Project)
4. Create webhook handler (Project)
5. Add Redis publisher (Project)
6. Add task type support (Collector)
7. Add webhook client (Collector)
8. Update message handling (WebSocket)
9. Configuration and logging
10. Testing and documentation

## Files to Create/Modify

### Project Service
**New Files:**
- `internal/webhook/type.go` (callback types)
- `internal/webhook/handler.go` (webhook handler + Redis publish)

**Modified Files:**
- `internal/project/type.go` (add DryRunKeywordsInput/Output)
- `internal/project/usecase/project.go` (implement DryRunKeywords)
- `internal/project/usecase/new.go` (add Producer dependency)
- `internal/project/delivery/http/handler.go` (add dry-run endpoint)
- `internal/project/delivery/http/routes.go` (register route)
- `internal/project/delivery/rabbitmq/constants.go` (add exchange and routing key)
- `internal/project/delivery/rabbitmq/presenters.go` (add DryRunCrawlRequest)
- `internal/project/delivery/rabbitmq/producer/new.go` (add PublishDryRunTask interface)
- `internal/project/delivery/rabbitmq/producer/producer.go` (implement PublishDryRunTask)
- `internal/project/delivery/rabbitmq/producer/common.go` (initialize dryRunWriter)
- `internal/httpserver/handler.go` (init RabbitMQ producer, register webhook route, init Redis)
- `internal/httpserver/server.go` (add Redis client)
- `config/config.go` (add INTERNAL_API_KEY if needed)
- `template.env` (add environment variables)

### Collector Service
**New Files:**
- `internal/webhook/client.go` (HTTP client with retry logic)

**Modified Files:**
- `internal/models/task.go` (add TaskTypeDryRunKeyword)
- `internal/dispatcher/usecase/dispatch_uc.go` (handle dryrun tasks)
- `internal/dispatcher/usecase/util.go` (add payload mapping)
- `internal/dispatcher/usecase/usecase.go` (integrate webhook client)
- `config/config.go` (add WebhookConfig)
- `template.env` (add environment variables)

### WebSocket Service
**Modified Files:**
- `internal/websocket/types.go` (add DryRunResult type)
- `internal/redis/subscriber.go` (handle dryrun messages)

## Next Steps

To begin implementation:

1. Open `project/openspec/specs/dryrun-keywords/tasks.md`
2. Click "Start task" next to task 1
3. Follow the implementation plan sequentially
4. Run tests after each checkpoint
5. Review and iterate as needed

## References

- Requirements: `requirements.md`
- Design: `design.md`
- Tasks: `tasks.md`
- Original Proposal: `../../../docs.md`
