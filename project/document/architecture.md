# SMAP API Project Architecture

## Overview

The SMAP API project is built using **Clean Architecture** (also known as Hexagonal Architecture) with a **Module-First** approach. Each module is an independent unit that contains all business logic for a specific feature.

---

## Architecture Principles

### 1. Module-First Approach

**Module-First** is a code organization method where each feature is organized into a complete and independent module. Each module contains:

- **Domain Logic**: Core business logic
- **Use Cases**: Business processing use cases
- **Repository Interface**: Defines how to access data
- **Delivery Layer**: How the module is exposed to the outside (HTTP, gRPC, etc.)
- **Error Handling**: Module-specific error handling

### 2. Clean Architecture Layers

The project follows Clean Architecture with the following layers:

```
Delivery Layer (HTTP/gRPC)       - Interface with external world
Use Case Layer                    - Business Logic
Repository Interface              - Contract for data access
Repository Implementation         - Data access implementation
Domain Model                      - Core entities
```

---

## Directory Structure

```
project/
├── cmd/                          # Application entry points
│   └── api/
│       ├── main.go              # Main entry point
│       └── Dockerfile
│
├── config/                       # Configuration management
│   ├── config.go                # Config loader
│   ├── postgre/                 # PostgreSQL config
│   └── minio/                   # MinIO config
│
├── internal/                     # Internal packages (not exported)
│   ├── httpserver/              # HTTP server setup
│   │   ├── handler.go           # Route mapping & dependency injection
│   │   ├── httpserver.go        # Server implementation
│   │   └── new.go               # Constructor
│   │
│   ├── middleware/              # HTTP middlewares
│   │   ├── auth.go              # Authentication
│   │   ├── cors.go              # CORS handling
│   │   ├── errors.go            # Error handling
│   │   └── recovery.go          # Panic recovery
│   │
│   ├── model/                   # Domain models (shared)
│   │   ├── project.go           # Project domain model
│   │   ├── role.go              # Role domain model
│   │   └── scope.go             # Scope (user context)
│   │
│   ├── project/                 # PROJECT MODULE (Module-First example)
│   │   ├── delivery/            # Delivery layer
│   │   │   └── http/            # HTTP delivery
│   │   │       ├── handler.go   # HTTP handlers
│   │   │       ├── routes.go    # Route definitions
│   │   │       ├── presenter.go # Response formatting
│   │   │       ├── process_request.go # Request DTOs
│   │   │       ├── error.go     # HTTP-specific errors
│   │   │       └── new.go       # Handler constructor
│   │   │
│   │   ├── repository/          # Repository layer
│   │   │   ├── interface.go    # Repository interface
│   │   │   ├── option.go        # Repository options
│   │   │   ├── errors.go        # Repository errors
│   │   │   └── postgre/         # PostgreSQL implementation
│   │   │       ├── new.go       # Repository constructor
│   │   │       ├── project.go   # CRUD operations
│   │   │       └── query.go     # SQL queries
│   │   │
│   │   ├── usecase/             # Use case layer
│   │   │   ├── new.go           # UseCase constructor
│   │   │   └── project.go       # Business logic
│   │   │
│   │   ├── interface.go         # UseCase interface
│   │   ├── type.go              # Input/Output types
│   │   └── error.go             # Module-specific errors
│   │
│   └── sqlboiler/               # Generated database models
│
├── pkg/                          # Shared packages (can be exported)
│   ├── discord/                 # Discord integration
│   ├── email/                   # Email service
│   ├── encrypter/               # Encryption utilities
│   ├── errors/                  # Error types
│   ├── i18n/                    # Internationalization
│   ├── log/                     # Logging
│   ├── minio/                   # MinIO client
│   ├── paginator/               # Pagination
│   ├── postgre/                 # PostgreSQL utilities
│   ├── rabbitmq/                # RabbitMQ client
│   ├── response/                # HTTP response helpers
│   ├── scope/                   # Scope management
│   └── util/                    # Utilities
│
├── migration/                    # Database migrations
│   └── 01_add_project.sql
│
├── document/                     # Documentation
│   ├── api.md
│   ├── overview.md
│   └── architecture.md          # This file
│
├── scripts/                      # Build scripts
├── go.mod
├── go.sum
└── README.md
```

---

## Module Structure (Module-First)

Each module in the project follows this structure:

```
internal/
└── {module-name}/              # Module name (e.g., project, user, order)
    ├── delivery/               # Delivery Layer - How module is exposed
    │   └── http/               # HTTP delivery (can have gRPC, CLI, etc.)
    │       ├── handler.go      # HTTP handlers (endpoints)
    │       ├── routes.go       # Route definitions
    │       ├── presenter.go    # Response DTOs & formatting
    │       ├── process_request.go # Request DTOs & validation
    │       ├── error.go        # HTTP-specific errors
    │       └── new.go          # Handler constructor
    │
    ├── repository/             # Repository Layer
    │   ├── interface.go        # Repository interface (contract)
    │   ├── option.go           # Repository options (filters, etc.)
    │   ├── errors.go           # Repository-specific errors
    │   └── postgre/            # PostgreSQL implementation
    │       ├── new.go          # Repository constructor
    │       ├── {entity}.go     # CRUD operations
    │       └── query.go        # SQL queries
    │
    ├── usecase/                # Use Case Layer
    │   ├── new.go              # UseCase constructor
    │   └── {entity}.go         # Business logic implementation
    │
    ├── interface.go            # UseCase interface (contract)
    ├── type.go                 # Input/Output types, DTOs
    └── error.go                # Module-specific errors
```

### Example: Project Module

The `project` module is a complete example of Module-First:

```
internal/project/
├── delivery/http/
│   ├── handler.go          # List, Get, Detail, Create, Update, Delete
│   ├── routes.go           # MapProjectRoutes()
│   ├── presenter.go        # ToProjectResp(), ToProjectListResp()
│   ├── process_request.go  # CreateReq, UpdateReq
│   └── new.go              # New() - create Handler
│
├── repository/
│   ├── interface.go        # Repository interface with methods
│   ├── option.go           # ListOptions, GetOptions, CreateOptions, etc.
│   ├── errors.go           # ErrNotFound, etc.
│   └── postgre/
│       ├── new.go          # New() - create Repository
│       ├── project.go      # Implement methods from interface
│       └── query.go        # SQL queries
│
├── usecase/
│   ├── new.go              # New() - create UseCase
│   └── project.go          # Implement methods from UseCase interface
│
├── interface.go            # UseCase interface
├── type.go                 # CreateInput, UpdateInput, ProjectOutput, etc.
└── error.go                # ErrProjectNotFound, ErrUnauthorized, etc.
```

---

## Request Processing Flow

### 1. Request Flow

```
HTTP Request
    |
[Middleware] - Authentication, CORS, Recovery
    |
[Handler] - Parse request, validate input
    |
[UseCase] - Business logic, validation
    |
[Repository] - Database operations
    |
[UseCase] - Transform data
    |
[Handler] - Format response
    |
HTTP Response
```

### 2. Specific Example: Create Project

```go
// 1. HTTP Request to POST /project/projects
// 2. Middleware authenticates JWT token
// 3. Handler receives request
func (h handler) Create(c *gin.Context) {
    // 4. Parse request body into CreateReq
    var req CreateReq
    c.ShouldBindJSON(&req)

    // 5. Convert request DTO to UseCase Input
    input := req.ToCreateInput()

    // 6. Call UseCase
    output, err := h.uc.Create(ctx, sc, input)

    // 7. Convert UseCase Output to HTTP Response
    resp := ToProjectResp(output.Project)
    response.OK(c, resp)
}

// 8. UseCase processes business logic
func (uc *usecase) Create(ctx context.Context, sc model.Scope, ip project.CreateInput) {
    // Validate business rules
    if !model.IsValidProjectStatus(ip.Status) {
        return project.ErrInvalidStatus
    }

    // Create domain model
    p := model.Project{...}

    // Call Repository
    created, err := uc.repo.Create(ctx, sc, repository.CreateOptions{Project: p})

    return project.ProjectOutput{Project: created}
}

// 9. Repository performs database operations
func (r *implRepository) Create(ctx context.Context, sc model.Scope, opts CreateOptions) {
    // Convert domain model to DB model
    dbProject := opts.Project.ToDBProject()

    // Execute SQL
    err := dbProject.Insert(ctx, r.db, boil.Infer())

    // Convert DB model to domain model
    return NewProjectFromDB(dbProject)
}
```

---

## Dependency Injection Flow

### 1. Initialize Dependencies (main.go)

```go
// 1. Load config
cfg := config.Load()

// 2. Initialize logger
logger := log.Init(...)

// 3. Initialize database
postgresDB := postgre.Connect(...)

// 4. Initialize HTTP server with dependencies
httpServer := httpserver.New(logger, httpserver.Config{
    PostgresDB: postgresDB,
    JwtSecretKey: cfg.JWT.SecretKey,
    // ...
})
```

### 2. Map Handlers (handler.go)

```go
func (srv HTTPServer) mapHandlers() error {
    // 1. Initialize Repository (data layer)
    projectRepo := projectrepository.New(srv.postgresDB, srv.l)

    // 2. Initialize UseCase (business layer)
    projectUC := projectusecase.New(srv.l, projectRepo)

    // 3. Initialize Handler (delivery layer)
    projectHandler := projecthttp.New(srv.l, projectUC)

    // 4. Map routes
    api := srv.gin.Group(apiPrefix)
    projecthttp.MapProjectRoutes(api.Group("/projects"), projectHandler, mw)

    return nil
}
```

### Dependency Graph

```
Handler
  └── UseCase
        └── Repository
              └── Database
```

---

## Creating a New Module

### Step 1: Create Directory Structure

```bash
mkdir -p internal/{module-name}/{delivery/http,repository/postgre,usecase}
```

### Step 2: Define Domain Model

Create file `internal/model/{entity}.go`:

```go
package model

type Entity struct {
    ID        string
    Name      string
    CreatedAt time.Time
    // ...
}
```

### Step 3: Create Module Structure

#### 3.1. Error Definitions (`error.go`)

```go
package {module}

import "errors"

var (
    ErrEntityNotFound = errors.New("entity not found")
    ErrInvalidInput   = errors.New("invalid input")
)
```

#### 3.2. Types (`type.go`)

```go
package {module}

type CreateInput struct {
    Name string
    // ...
}

type UpdateInput struct {
    ID   string
    Name *string
    // ...
}

type EntityOutput struct {
    Entity model.Entity
}
```

#### 3.3. UseCase Interface (`interface.go`)

```go
package {module}

//go:generate mockery --name UseCase
type UseCase interface {
    Create(ctx context.Context, sc model.Scope, ip CreateInput) (EntityOutput, error)
    Update(ctx context.Context, sc model.Scope, ip UpdateInput) (EntityOutput, error)
    // ...
}
```

#### 3.4. Repository Interface (`repository/interface.go`)

```go
package repository

//go:generate mockery --name Repository
type Repository interface {
    Create(ctx context.Context, sc model.Scope, opts CreateOptions) (model.Entity, error)
    Update(ctx context.Context, sc model.Scope, opts UpdateOptions) (model.Entity, error)
    // ...
}
```

#### 3.5. Repository Implementation (`repository/postgre/{entity}.go`)

```go
package postgres

func (r *implRepository) Create(ctx context.Context, sc model.Scope, opts CreateOptions) (model.Entity, error) {
    // Implement database operations
}
```

#### 3.6. UseCase Implementation (`usecase/{entity}.go`)

```go
package usecase

func (uc *usecase) Create(ctx context.Context, sc model.Scope, ip {module}.CreateInput) ({module}.EntityOutput, error) {
    // Business logic
    entity := model.Entity{...}

    created, err := uc.repo.Create(ctx, sc, repository.CreateOptions{Entity: entity})
    if err != nil {
        return {module}.EntityOutput{}, err
    }

    return {module}.EntityOutput{Entity: created}, nil
}
```

#### 3.7. HTTP Handler (`delivery/http/handler.go`)

```go
package http

func (h handler) Create(c *gin.Context) {
    ctx := c.Request.Context()
    sc, _ := scope.GetScopeFromContext(ctx)

    var req CreateEntityRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, err, nil)
        return
    }

    input := req.ToCreateInput()
    output, err := h.uc.Create(ctx, sc, input)
    if err != nil {
        response.Error(c, err, nil)
        return
    }

    resp := ToEntityResponse(output.Entity)
    response.OK(c, resp)
}
```

#### 3.8. Routes (`delivery/http/routes.go`)

```go
package http

func MapEntityRoutes(r *gin.RouterGroup, h Handler, mw middleware.Middleware) {
    r.GET("", mw.Auth(), h.List)
    r.GET("/:id", mw.Auth(), h.Detail)
    r.POST("", mw.Auth(), h.Create)
    r.PUT("/:id", mw.Auth(), h.Update)
    r.DELETE("/:id", mw.Auth(), h.Delete)
}
```

### Step 4: Register Module in HTTP Server

In `internal/httpserver/handler.go`:

```go
func (srv HTTPServer) mapHandlers() error {
    // ... existing code ...

    // Initialize new module
    entityRepo := entityrepository.New(srv.postgresDB, srv.l)
    entityUC := entityusecase.New(srv.l, entityRepo)
    entityHandler := entityhttp.New(srv.l, entityUC)

    // Map routes
    entityhttp.MapEntityRoutes(api.Group("/entities"), entityHandler, mw)

    return nil
}
```

---

## Principles & Best Practices

### 1. Dependency Rule

**Dependencies flow in one direction only:**

```
Delivery → UseCase → Repository → Database
```

**NEVER:**

- Repository imports UseCase
- UseCase imports Delivery
- Database imports Repository

### 2. Interface Segregation

- Each layer only knows about the interface of the layer below it
- UseCase doesn't know about Repository implementation
- Handler doesn't know about UseCase implementation

### 3. Error Handling

- **Domain Errors**: Defined in `{module}/error.go`
- **Repository Errors**: Defined in `{module}/repository/errors.go`
- **HTTP Errors**: Defined in `{module}/delivery/http/error.go`

### 4. Type Safety

- Use separate types for Input/Output at each layer
- Never pass domain models directly through HTTP
- Convert between layers: `Request → Input → Domain → Output → Response`

### 5. Scope & Context

- Always pass `context.Context` through all layers
- Use `model.Scope` to pass user/tenant information
- Don't store state in structs (stateless)

### 6. Testing

- Mock interfaces to test each layer independently
- Use `//go:generate mockery` to generate mocks
- Test UseCase with mock Repository
- Test Handler with mock UseCase

---

## Module Communication

### 1. Within Same Service

Modules can communicate with each other through UseCase:

```go
// Module A needs data from Module B
type UseCaseA struct {
    repoA RepositoryA
    ucB   moduleB.UseCase  // Inject UseCase of module B
}

func (uc *UseCaseA) SomeMethod() {
    // Call UseCase of module B
    data, err := uc.ucB.GetSomething(ctx, sc, input)
}
```

### 2. Cross-Service Communication

Use message queue (RabbitMQ) or HTTP client for communication between services.

---

## Database & Migrations

### 1. Migrations

- Migrations are stored in `migration/`
- Name format: `{number}_{description}.sql`
- Example: `01_add_project.sql`, `02_add_user.sql`

### 2. SQLBoiler

- Models are generated from database schema
- Run `make sqlboiler` after changing schema
- Generated models are in `internal/sqlboiler/`

### 3. Domain Model vs DB Model

- **DB Model** (`sqlboiler.*`): Direct mapping to database
- **Domain Model** (`model.*`): Business entities, independent of database
- Always convert between the two model types in Repository layer

---

## Configuration Management

### 1. Config Structure

Config is defined in `config/config.go` and loaded from environment variables.

### 2. Environment Variables

Use `.env` file (template: `template.env`) to manage config.

---

## Logging

### 1. Logger Interface

Use `pkg/log.Logger` interface for logging.

### 2. Log Levels

- `Error`: Errors that need handling
- `Warn`: Warnings
- `Info`: Important information
- `Debug`: Debug information

### 3. Context Logging

Always pass `context.Context` to logger to trace requests.

---

## Summary

### Module-First Approach

1. **Each module is an independent unit** containing all logic for a feature
2. **Module has clear structure**: Delivery → UseCase → Repository
3. **Module can be tested independently** thanks to dependency injection
4. **Module is easy to extend** when adding new features

### Clean Architecture Benefits

1. **Separation of Concerns**: Each layer has its own responsibility
2. **Testability**: Easy to test each layer independently
3. **Maintainability**: Code is easy to read and maintain
4. **Flexibility**: Easy to change implementation without affecting other layers

### When Creating a New Module

1. Create directory structure according to template
2. Define interfaces first
3. Implement from bottom up: Repository → UseCase → Handler
4. Register module in HTTP server
5. Write tests for each layer

---

**This document will be updated when there are changes to architecture or new best practices.**

---

## Webhook Module & Redis Pub/Sub

### Overview

The webhook module handles callbacks from external services (crawler, collector) and publishes real-time updates to WebSocket clients via Redis Pub/Sub.

### Architecture

```
Crawler/Collector → Project Service (Webhook) → Redis Pub/Sub → WebSocket Service → Client
```

### Topic Patterns

The webhook module uses topic-specific routing patterns for Redis Pub/Sub:

| Message Type     | Topic Pattern                  | Description                            |
| ---------------- | ------------------------------ | -------------------------------------- |
| Dry-run Jobs     | `job:{jobID}:{userID}`         | Results from dry-run crawl jobs        |
| Project Progress | `project:{projectID}:{userID}` | Progress updates for project execution |

### Message Types

#### JobMessage (Dry-run Results)

Published to topic: `job:{jobID}:{userID}`

```go
type JobMessage struct {
    Platform Platform   `json:"platform"`           // TIKTOK, YOUTUBE, INSTAGRAM
    Status   Status     `json:"status"`             // PROCESSING, COMPLETED, FAILED
    Batch    *BatchData `json:"batch,omitempty"`    // Crawled content
    Progress *Progress  `json:"progress,omitempty"` // Progress metrics
}
```

#### ProjectMessage (Progress Updates)

Published to topic: `project:{projectID}:{userID}`

```go
type ProjectMessage struct {
    Status   Status    `json:"status"`             // PROCESSING, COMPLETED, FAILED
    Progress *Progress `json:"progress,omitempty"` // Progress metrics
}
```

### Data Flow

#### Dry-Run Job Flow

```
1. Crawler sends callback → Project Service
   CallbackRequest { JobID, Platform, Status, Payload }

2. Project Service looks up userID from Redis job mapping
   Key: job:mapping:{jobID}

3. Transform to JobMessage:
   - Map content to BatchData
   - Map errors to Progress.Errors
   - Set platform and status enums

4. Publish to Redis: job:{jobID}:{userID}
```

#### Project Progress Flow

```
1. Collector sends callback → Project Service
   ProgressCallbackRequest { ProjectID, UserID, Status, Total, Done, Errors }

2. Transform to ProjectMessage:
   - Calculate progress percentage
   - Map status to enum
   - Convert error count to messages

3. Publish to Redis: project:{projectID}:{userID}
```

### Webhook Module Structure

```
internal/webhook/
├── delivery/http/
│   ├── handler.go          # DryRunCallback, ProgressCallback
│   ├── routes.go           # MapWebhookRoutes()
│   ├── process_request.go  # Request parsing
│   ├── error.go            # HTTP-specific errors
│   └── new.go              # Handler constructor
│
├── usecase/
│   ├── new.go              # UseCase constructor
│   ├── webhook.go          # HandleDryRunCallback, HandleProgressCallback
│   ├── transformers.go     # TransformDryRunCallback, TransformProjectCallback
│   └── job_mapping.go      # StoreJobMapping, getJobMapping
│
├── interface.go            # UseCase interface
├── type.go                 # CallbackRequest, ProgressCallbackRequest
└── redis_types.go          # JobMessage, ProjectMessage, enums
```

### Job Mapping

Job mappings are stored in Redis to associate JobID with UserID and ProjectID:

```
Key: job:mapping:{jobID}
Value: {
    "user_id": "...",
    "project_id": "...",
    "platform": "...",
    "created_at": "..."
}
TTL: 7 days
```

### WebSocket Service Integration

The WebSocket service subscribes to Redis patterns:

```go
// Subscribe to new topic patterns
redis.PSubscribe("job:*")
redis.PSubscribe("project:*")

// Handle messages
func handleMessage(channel string, data []byte) {
    if strings.HasPrefix(channel, "job:") {
        // Parse JobMessage and deliver to client
    } else if strings.HasPrefix(channel, "project:") {
        // Parse ProjectMessage and deliver to client
    }
}
```

### Related Documentation

- [Redis Pub/Sub Deployment Checklist](redis_pubsub_deployment_checklist.md)
- [Redis Pub/Sub Rollback Procedures](redis_pubsub_rollback_procedures.md)
- [Redis Pub/Sub Troubleshooting Guide](redis_pubsub_troubleshooting.md)
