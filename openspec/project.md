# Project Context

## Purpose

SMAP Project Service manages project-related operations for the SMAP platform. It provides CRUD operations for projects including brand tracking, competitor analysis, and keyword management.

## Tech Stack

### Backend Core
- **Go** 1.25+ - Primary programming language
- **Gin** v1.11.0 - HTTP web framework and routing
- **PostgreSQL** 15+ - Primary relational database
- **SQLBoiler** v4.19.5 - Type-safe ORM with code generation

### Security & Authentication
- **HttpOnly Cookies** - Primary authentication method using secure cookies
  - Cookie name: `smap_auth_token`
  - Domain: `.smap.com` (shared across SMAP services)
  - Attributes: HttpOnly, Secure, SameSite=Lax
  - Session duration: 2 hours (normal) / 30 days (remember me)
- **JWT** (golang-jwt/jwt) - Token validation from Identity service
- **Bearer Token Fallback** - Legacy support during migration (deprecated)
- **Custom Middleware** - Cookie-first authentication with Bearer fallback
- **Encrypter** - Data encryption utilities

### Development & Tooling
- **Swagger** (swaggo) - API documentation generation
- **Zap** (go.uber.org/zap) - Structured logging
- **Make** - Build automation and task management
- **Docker** - Containerization

### Infrastructure
- **Redis** - Pub/sub messaging and state management (DB 0: job mapping, DB 1: project progress tracking)
- **RabbitMQ** - Message queue for async operations
- **MinIO** - Object storage (infrastructure ready but not actively used)
- **Discord Webhooks** - Notification system for error reporting

### Supporting Libraries
- **caarlos0/env/v9** - Environment variable parsing
- **google/uuid** - UUID generation
- **nicksnyder/go-i18n/v2** - Internationalization
- **golang.org/x/crypto** - Cryptographic functions

## Project Conventions

**Naming Conventions:**
- **Exported identifiers**: PascalCase (e.g., `Project`, `CreateInput`, `Handler`)
- **Private identifiers**: camelCase (e.g., `handler`, `usecase`, `repository`)
- **Interfaces**: PascalCase with descriptive names (e.g., `UseCase`, `Repository`)
- **Error variables**: `Err` prefix (e.g., `ErrNotFound`, `ErrUnauthorized`)
- **Package names**: lowercase, single word (e.g., `http`, `usecase`, `repository`)

**File Organization:**
- One main type per file when possible
- Related types grouped in `type.go`
- Errors defined in `error.go`
- Constructors in `new.go`

**Code Formatting:**
- Use `gofmt` for standard formatting
- Follow Go standard library conventions
- Use meaningful variable names
- Keep functions focused and small

**Error Handling:**
- Always check and handle errors explicitly
- Use structured error types per module
- Log errors with context: `l.Errorf(ctx, "package.function: %v", err)`
- Return domain errors from UseCase, map to HTTP errors in Handler

**Logging:**
- Always pass `context.Context` to logger
- Use structured logging with Zap
- Log levels: `Error`, `Warn`, `Info`, `Debug`
- Include package and function name in log messages

### Architecture Patterns

**Clean Architecture (Hexagonal Architecture):**
- **3-Layer Structure**: Delivery → UseCase → Repository
- **Dependency Rule**: Dependencies flow inward only
- **Module-First Approach**: Each feature is a complete, independent module

**Module Structure:**
```
{module}/
├── delivery/http/     # HTTP handlers, routes, DTOs
├── usecase/           # Business logic
├── repository/         # Data access (interface + implementation)
├── interface.go       # UseCase interface
├── type.go            # Input/Output types
└── error.go           # Module errors
```

**Dependency Injection:**
- Constructor functions: `New()` pattern
- Dependencies passed as parameters
- Interfaces for all external dependencies
- No global state

**Data Flow:**
```
HTTP Request → Handler → UseCase → Repository → Database
                ↓          ↓          ↓
HTTP Response ← Presenter ← Output ← Domain Model
```

**Key Principles:**
- **Separation of Concerns**: Each layer has single responsibility
- **Interface Segregation**: Layers communicate via interfaces
- **Type Safety**: Separate types for each layer (Request → Input → Domain → Output → Response)
- **Stateless**: No state stored in structs, pass context through layers

### Testing Strategy

**Testing Approach:**
- **Unit Tests**: Test each layer independently
- **Mock Interfaces**: Use `mockery` to generate mocks (`//go:generate mockery`)
- **Test Structure**: Test UseCase with mock Repository, test Handler with mock UseCase
- **Test Files**: `*_test.go` alongside source files

**Testing Patterns:**
- Mock external dependencies (database, external services)
- Test business logic in UseCase layer
- Test HTTP handlers with mock UseCase
- Test repository with test database (if needed)

**Test Coverage:**
- Focus on business logic and critical paths
- Test error cases and edge cases
- Validate input/output transformations

**Test Tools:**
- Standard Go `testing` package
- `mockery` for interface mocking
- Test database for integration tests (if needed)

### Git Workflow

**Branching Strategy:**
- **Main Branch**: `master-project-api` (feature branch from main)
- **Feature Branches**: Create branches for new features/modules
- **Naming**: Descriptive branch names (e.g., `feature/add-user-module`)

**Commit Conventions:**
- Use descriptive commit messages
- Reference issues/PRs when applicable
- Keep commits focused and atomic

**Workflow:**
1. Create feature branch from `master-project-api`
2. Make changes following Clean Architecture patterns
3. Write/update tests
4. Create pull request for review
5. Merge after approval

**File Organization:**
- Keep changes modular and focused
- Update documentation when adding features
- Run `make models` after schema changes
- Run `make swagger` after API changes

## Domain Context

**Project Entity:**
- **Core Fields**: ID (UUID), name, description, status, date range (from_date, to_date)
- **Brand Tracking**: brand_name, brand_keywords (array)
- **Competitor Analysis**: competitor_names (array), competitor_keywords_map (JSONB)
- **Ownership**: created_by (UUID from JWT, no FK to users table)
- **Timestamps**: created_at, updated_at, deleted_at (soft delete)

**Project Status Workflow:**
- `draft` → Initial state, not yet active
- `active` → Project is running
- `completed` → Project finished successfully
- `archived` → Project archived for reference
- `cancelled` → Project cancelled/abandoned

**User Isolation:**
- Users can only access projects they created (`created_by` matches JWT `user_id`)
- All queries automatically filter by `created_by`
- Authorization checks in UseCase layer

**Data Validation:**
- `to_date` must be after `from_date`
- Status must be one of predefined values
- Brand and competitor names are required for tracking
- Keywords are arrays of strings

**Soft Delete Pattern:**
- Projects are never physically deleted
- `deleted_at` timestamp marks deletion
- Queries exclude deleted records by default
- Data retained for audit purposes

**Competitor Keywords Mapping:**
- JSONB structure: `{"Competitor A": ["kw1", "kw2"], "Competitor B": ["kw3"]}`
- Flexible structure allows different keywords per competitor
- Stored as JSONB for query flexibility

## Important Constraints

**Technical Constraints:**
- **Microservice Independence**: No foreign keys to Identity service (different database)
- **JWT-Based Auth**: Authentication via JWT tokens from Identity service (stateless)
- **PostgreSQL Required**: Database is mandatory, no fallback
- **User ID from JWT**: `created_by` comes from JWT token, not database lookup
- **No Direct Service Calls**: Identity service accessed only via JWT validation

**Architectural Constraints:**
- **Clean Architecture**: Must follow 3-layer pattern (Delivery → UseCase → Repository)
- **Module-First**: New features must be complete modules
- **Interface-Based**: All dependencies must use interfaces
- **No Circular Dependencies**: Dependencies flow inward only

**Data Constraints:**
- **UUID Primary Keys**: All IDs are UUIDs, auto-generated by database
- **Soft Delete**: All deletions are soft deletes
- **User Isolation**: All queries must filter by user
- **Date Validation**: End date must be after start date

**Business Constraints:**
- **User Privacy**: Users cannot access other users' projects
- **Data Retention**: Deleted projects retained for audit
- **Status Transitions**: Status changes should follow logical workflow

**Performance Constraints:**
- **Pagination Required**: Large datasets must support pagination
- **Indexing**: Database indexes on `created_by`, `status`, `deleted_at`
- **Query Optimization**: Use efficient queries, avoid N+1 problems

## External Dependencies

### Required Services

**1. PostgreSQL Database**
- **Purpose**: Primary data storage for projects
- **Connection**: `postgresql://user:password@host:5432/smap_project?sslmode=disable`
- **Port**: 5432 (default)
- **Features Used**:
  - UUID auto-generation (`gen_random_uuid()`)
  - Array types (TEXT[]) for keywords and competitor names
  - JSONB for flexible competitor keyword mapping
  - Indexes on `created_by`, `status`, `deleted_at`
  - Soft delete pattern with `deleted_at` timestamp
- **Migration**: SQL files in `migration/` directory
- **Model Generation**: SQLBoiler generates models from schema

**2. Identity Service (Indirect)**
- **Purpose**: User authentication via JWT tokens
- **Integration**: Stateless (no direct API calls)
- **JWT Token Contains**:
  - `user_id`: UUID of authenticated user
  - `username`: User's email/identifier
  - `role`: User's role (if applicable)
- **Validation**: Done by middleware, extracts `user_id` from token
- **Flow**: `User → Identity Service (login) → JWT Token → Project Service (validates token)`
- **No Foreign Keys**: `created_by` stores UUID but has no FK constraint

### Optional Services (Currently Commented Out)

**3. MinIO (Object Storage)**
- **Purpose**: File storage (currently not active)
- **Configuration**: Available in config but commented out in main.go
- **Use Case**: Would be used for project file attachments if enabled

**3. Redis (Cache & State Management)**
- **Purpose**: Pub/sub messaging and project progress state tracking
- **Configuration**: Two Redis databases
  - DB 0: Main Redis for job mapping and pub/sub
  - DB 1: State Redis for project progress tracking
- **Connection**: `redis://host:port` with password support
- **Features Used**:
  - Pub/sub for webhook event processing
  - Job mapping storage (job_id → user_id, project_id)
  - Project progress state tracking
- **Port**: 6379 (default)

**4. RabbitMQ (Message Queue)**
- **Purpose**: Async message processing for project operations
- **Configuration**: Connection URL from environment variables
- **Use Case**: Used for async project-related message processing

**5. MinIO (Object Storage)**
- **Purpose**: File storage (infrastructure ready but not actively used)
- **Configuration**: Available in config but not initialized in main.go
- **Use Case**: Would be used for project file attachments if enabled

**6. Discord Webhooks**
- **Purpose**: Error notifications and monitoring alerts
- **Integration**: Active, used for error reporting
- **Configuration**: Webhook ID and token from environment variables

### Service Dependencies Map

```
Project API Server
├── PostgreSQL (Required)
│   └── Projects data persistence
│
├── Redis (Required)
│   ├── DB 0: Job mapping and pub/sub
│   └── DB 1: Project progress state tracking
│
├── RabbitMQ (Required)
│   └── Async message processing
│
├── Identity Service (Indirect - via JWT)
│   └── User authentication (stateless)
│
└── Discord (Active)
    └── Error notifications
```

**Dependency Notes:**
- PostgreSQL, Redis, and RabbitMQ are hard dependencies (required for service to start)
- Identity service is indirect (JWT tokens validated, no service calls)
- MinIO infrastructure is ready but not actively used
- Discord is active for monitoring but not critical for core functionality
