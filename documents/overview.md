# SMAP Project API - Documentation Overview

> Complete documentation structure for SMAP Project Service - Project Management, Brand Tracking & Competitor Analysis

---

## Table of Contents

- [Service Overview](#service-overview)
- [Architecture](#architecture)
- [Documentation Structure](#documentation-structure)
- [Entity-Relationship Diagram](#entity-relationship-diagram)
- [Technology Stack](#technology-stack)
- [Document Maintenance](#document-maintenance)

---

## Service Overview

**SMAP Project API** is a project management system built with Clean Architecture pattern, providing project lifecycle management, brand tracking, and competitor analysis capabilities.

### API Server (`cmd/api/`)

RESTful API service providing:

**Project Management:**
- Project CRUD operations (Create, Read, Update, Delete)
- Project lifecycle management with status tracking
- Date range validation and timeline management
- Soft delete support for data retention
- User isolation (users only access their own projects)

**Brand & Competitor Tracking:**
- Brand name and keyword management
- Competitor names tracking
- Competitor keyword mapping (JSON structure)
- Multi-brand/multi-competitor support per project

**Key Features:**
- JWT-based authentication (tokens from Identity service)
- Pagination support for large datasets
- Search and filtering capabilities
- Status-based project organization
- Comprehensive error handling

---

## Architecture

### System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Applications                         │
│                    (Web, Mobile, Third-party)                       │
└────────────────────────┬────────────────────────────────────────────┘
                         │ HTTPS/REST API + JWT Token
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Project API Server (:8080)                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                  HTTP Delivery Layer                         │   │
│  │  ├─ Project Handlers     (/project/projects/*)               │   │
│  │  ├─ JWT Middleware       (validates tokens)                  │   │
│  │  ├─ Error Middleware     (error handling)                    │   │
│  │  └─ CORS Middleware      (cross-origin)                      │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                            │                                        │
│  ┌─────────────────────────▼────────────────────────────────────┐   │
│  │                   UseCase Layer                              │   │
│  │             (Business Logic & Validation)                    │   │
│  │  ├─ Create Project (with validation)                         │   │
│  │  ├─ Update Project (ownership check)                         │   │
│  │  ├─ Delete Project (soft delete)                             │   │
│  │  ├─ List Projects (user isolation)                           │   │
│  │  ├─ Get Project (pagination)                                 │   │
│  │  └─ Get Project Detail (authorization)                       │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                            │                                        │
│  ┌─────────────────────────▼────────────────────────────────────┐   │
│  │                  Repository Layer                            │   │
│  │                (Data Access & Queries)                       │   │
│  │  ├─ Project Repository (PostgreSQL)                          │   │
│  │  ├─ Query Builders (filtering, pagination)                   │   │
│  │  └─ CRUD Operations                                          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
└────────┬────────────────────────────────────────────────────────────┘
         │ SQL Queries
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database (:5432)                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Table: projects                                             │   │
│  │  ├─ id (UUID, auto-generated)                                │   │
│  │  ├─ name, description, status                                │   │
│  │  ├─ from_date, to_date                                       │   │
│  │  ├─ brand_name, brand_keywords[]                             │   │
│  │  ├─ competitor_names[], competitor_keywords_map (JSONB)      │   │
│  │  ├─ created_by (UUID - from JWT, no FK)                      │   │
│  │  └─ timestamps (created_at, updated_at, deleted_at)          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Indexes:                                                           │
│  ├─ idx_projects_created_by (query by user)                         │
│  ├─ idx_projects_status (filter by status)                          │
│  └─ idx_projects_deleted_at (exclude deleted)                       │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│              Identity Service (Separate Microservice)               │
│  - Issues JWT tokens                                                │
│  - Handles user authentication                                      │
│  - Separate database                                                │
└─────────────────────────────────────────────────────────────────────┘
```

### Clean Architecture Layers

**3-Layer Architecture Pattern:**

```
┌──────────────────────────────────────────────────────────────────┐
│                      1. Delivery Layer                           │
│                                                                  │
│  - HTTP Handlers (Gin framework)                                 │
│  - Request/Response DTOs                                         │
│  - Input validation                                              │
│  - Middleware (JWT Auth, CORS, Error handling)                   │
│  - Presenter (response formatting)                               │
│                                                                  │
└─────────────────────────────┬────────────────────────────────────┘
                              │ Calls
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                       2. UseCase Layer                           │
│                                                                  │
│  - Business Logic                                                │
│  - Authorization checks (user ownership)                         │
│  - Validation rules (dates, status, etc.)                        │
│  - Error handling                                                │
│  - User isolation enforcement                                    │
│                                                                  │
└─────────────────────────────┬────────────────────────────────────┘
                              │ Calls
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                     3. Repository Layer                          │
│                                                                  │
│  - Data access abstraction                                       │
│  - Query builders (filtering, pagination)                        │
│  - Database operations (CRUD)                                    │
│  - SQLBoiler integration                                         │
│                                                                  │
└─────────────────────────────┬────────────────────────────────────┘
                              │ Queries
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                  4. External Services Layer                      │
│                                                                  │
│  - PostgreSQL Database (projects data)                           │
│  - Identity Service (JWT validation - via middleware)            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Key Architectural Decisions

**1. Microservice Independence:**
- Project service manages its own database
- No foreign keys to Identity service (different database)
- JWT tokens carry user information (no need to query Identity)
- Fully decoupled services

**2. Database Design:**
- Projects table: All project data including brand & competitor info
- Auto-generated UUIDs via `gen_random_uuid()`
- JSONB for competitor keyword mapping (flexible structure)
- Array types for multiple values (brand_keywords, competitor_names)
- Soft delete pattern: deleted_at timestamp

**3. User Isolation:**
```
User Authentication Flow:
Identity Service → JWT Token (contains user_id) → Project Service → Filter by created_by
```

**4. Data Flow:**
```
Request → JWT Validation → Extract user_id → UseCase (ownership check) → Repository (query) → Response
```

---

## Entity-Relationship Diagram

```mermaid
erDiagram
    PROJECTS {
        uuid id PK "Auto-generated UUID"
        varchar name "Project name"
        text description "Optional description"
        varchar status "draft|active|completed|archived|cancelled"
        timestamptz from_date "Project start date"
        timestamptz to_date "Project end date"
        varchar brand_name "Brand being tracked"
        text_array competitor_names "Array of competitor names"
        text_array brand_keywords "Array of brand keywords"
        jsonb competitor_keywords_map "Map: competitor -> keywords"
        uuid created_by "User ID from JWT (no FK)"
        timestamptz created_at "Record creation"
        timestamptz updated_at "Last update"
        timestamptz deleted_at "Soft delete"
    }
```

**Notes:**
- `created_by` does NOT reference users table (different service/database)
- `competitor_keywords_map` example: `{"Competitor A": ["kw1", "kw2"], "Competitor B": ["kw3"]}`
- `status` must be one of the predefined values
- `to_date` must be after `from_date`

---

## Technology Stack

### Backend Core
| Technology | Version | Purpose |
|------------|---------|---------|
| **Go** | 1.23+ | Primary programming language |
| **Gin** | Latest | HTTP web framework, routing |
| **PostgreSQL** | 15 | Primary database, relational data |
| **SQLBoiler** | v4.19.5 | Type-safe ORM, code generation |

### Security & Auth
| Technology | Version | Purpose |
|------------|---------|---------|
| **JWT** | - | Token validation from Identity service |
| **Middleware** | Custom | JWT extraction and validation |

### Development Tools
| Technology | Version | Purpose |
|------------|---------|---------|
| **Swagger** | v1.8.12 | API documentation |
| **Zap** | Latest | Structured logging |
| **Make** | Built-in | Build automation |

### Infrastructure & DevOps
| Technology | Version | Purpose |
|------------|---------|---------|
| **Docker** | >= 20.10 | Containerization |
| **BuildKit** | Latest | Docker build optimization |
| **Distroless** | debian12 | Minimal runtime image |

---

## External Services

### Required Services

#### 1. PostgreSQL Database
**Purpose**: Primary data storage for projects
- **Port**: 5432
- **Tables**: projects
- **Features Used**:
  - UUID auto-generation (`gen_random_uuid()`)
  - Array types (TEXT[])
  - JSONB for flexible data
  - Indexes (created_by, status, deleted_at)
  - Soft delete (deleted_at timestamp)

**Connection:**
```
postgresql://user:password@host:5432/smap_project?sslmode=disable
```

---

#### 2. Identity Service (Indirect)
**Purpose**: User authentication (via JWT tokens)
- **Integration**: Stateless (no direct API calls)
- **JWT Token Contains**:
  - `user_id`: UUID of authenticated user
  - `username`: User's email
  - `role`: User's role
- **Validation**: Done by middleware, no service-to-service calls

**JWT Flow:**
```
User → Identity Service (login) → JWT Token → Project Service (validates token)
```

---

### Service Dependencies Map

```
Project API Server Dependencies:
└─ PostgreSQL (Required)
   └─ Projects data persistence

Identity Service (Separate):
└─ JWT Token Provider
   └─ User authentication (external)
```

---

## Documentation Structure

### Core Documents

1. **`overview.md`** (This File)
   - Service architecture overview
   - Technology stack
   - System diagrams
   - Dependencies

2. **`api.md`**
   - API endpoint documentation
   - Sequence diagrams for each operation
   - Request/response examples
   - Error handling flows

---

## Document Maintenance

### Version History
- **v1.0.0** (November 21, 2025): Initial Project service documentation
  - 2 core documentation files (overview, api)
  - 6 sequence diagrams (CRUD operations)
  - Complete architecture diagrams
  - Clean Architecture implementation

**Last Updated**: November 21, 2025
**Document Version**: 1.0.0
**Maintained By**: SMAP Development Team

---

*Built with love using Clean Architecture principles*
