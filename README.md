# SMAP Project Service

> Project management service for the SMAP platform

---

## Overview

**SMAP Project Service** manages project-related operations for the SMAP platform. It provides CRUD operations for projects including brand tracking, competitor analysis, and keyword management.

### Key Features

- **Project Management**: Create, read, update, and delete projects
- **Brand Tracking**: Track brand names and keywords
- **Competitor Analysis**: Monitor competitor names and their associated keywords
- **Date Range Management**: Project timeline management with validation
- **Status Tracking**: Draft, Active, Completed, Archived, Cancelled
- **User Isolation**: Users can only access their own projects
- **Soft Delete**: Data retention for audit purposes

---

## Authentication

The Project service uses **HttpOnly cookie authentication** for secure, stateless authentication.

### Authentication Methods

**Primary: HttpOnly Cookies** (Recommended)

- Cookie name: `smap_auth_token`
- Set automatically by Identity service `/login` endpoint
- Sent automatically by browser with each request
- Secure attributes: HttpOnly, Secure, SameSite=Lax

**Legacy: Bearer Token** (Deprecated)

- Supported for backward compatibility during migration
- Format: `Authorization: Bearer {token}`
- Will be removed in future versions

### Getting Authenticated

```bash
# Login via Identity service to get cookie
curl -i -X POST https://smap-api.tantai.dev/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your_password",
    "remember": false
  }' \
  -c cookies.txt

# Cookie is now stored and will be sent automatically
```

---

## API Endpoints

### Base URL

```
https://smap-api.tantai.dev/project
```

### Project Endpoints

| Method | Endpoint         | Description                  | Auth Required       |
| ------ | ---------------- | ---------------------------- | ------------------- |
| GET    | `/projects`      | List all user's projects     | Yes (Cookie/Bearer) |
| GET    | `/projects/page` | Get projects with pagination | Yes (Cookie/Bearer) |
| GET    | `/projects/:id`  | Get project details          | Yes (Cookie/Bearer) |
| POST   | `/projects`      | Create new project           | Yes (Cookie/Bearer) |
| PUT    | `/projects/:id`  | Update project               | Yes (Cookie/Bearer) |
| DELETE | `/projects/:id`  | Delete project (soft delete) | Yes (Cookie/Bearer) |

---

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 15+
- Make

### Quick Start

```bash
# Install dependencies
go mod download

# Run migrations
make migrate-up

# Generate SQLBoiler models
make sqlboiler

# Run the service
make run-api
```

### API Examples

**Using Cookie Authentication (Recommended):**

```bash
# After logging in via Identity service, cookie is stored
# Use -b flag to send cookies with request
curl -X GET https://smap-api.tantai.dev/project/projects \
  -b cookies.txt
```

**Create Project with Cookie:**

```bash
curl -X POST https://smap-api.tantai.dev/project/projects \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Q1 2025 Campaign",
    "status": "draft",
    "from_date": "2025-01-01T00:00:00Z",
    "to_date": "2025-03-31T23:59:59Z",
    "brand_name": "MyBrand",
    "brand_keywords": ["mybrand", "my brand"],
    "competitor_names": ["Competitor A"],
    "competitor_keywords_map": {
      "Competitor A": ["competitor-a", "comp-a"]
    }
  }'
```

**Using Bearer Token (Legacy):**

```bash
# For backward compatibility during migration
curl -X POST https://smap-api.tantai.dev/project/projects \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Q1 2025 Campaign",
    "status": "draft",
    "from_date": "2025-01-01T00:00:00Z",
    "to_date": "2025-03-31T23:59:59Z",
    "brand_name": "MyBrand",
    "brand_keywords": ["mybrand", "my brand"],
    "competitor_names": ["Competitor A"],
    "competitor_keywords_map": {
      "Competitor A": ["competitor-a", "comp-a"]
    }
  }'
```

---

## Configuration

The service is configured using environment variables. The following variables are available:

| Variable                | Description                                             | Default                 |
| ----------------------- | ------------------------------------------------------- | ----------------------- |
| `LLM_PROVIDER`          | The LLM provider to use for keyword suggestions.        | `gemini`                |
| `LLM_API_KEY`           | The API key for the LLM provider.                       |                         |
| `LLM_MODEL`             | The LLM model to use.                                   | `gemini-2.0-flash`      |
| `LLM_TIMEOUT`           | The timeout in seconds for LLM API calls.               | `30`                    |
| `LLM_MAX_RETRIES`       | The maximum number of retries for failed LLM API calls. | `3`                     |
| `COLLECTOR_SERVICE_URL` | The base URL of the Collector Service for dry runs.     | `http://localhost:8081` |
| `COLLECTOR_TIMEOUT`     | The timeout in seconds for Collector Service API calls. | `30`                    |

---

**Built for SMAP Graduation Project**
