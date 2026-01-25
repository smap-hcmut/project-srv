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

**Lưu ý:** Khi sử dụng Swagger UI, bạn có thể authenticate trực tiếp trong giao diện Swagger bằng cách click nút "Authorize" và nhập token hoặc cookie.

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

# Generate Swagger documentation
make swagger

# Run the service
make run-api
```

Service sẽ chạy tại `http://localhost:8080` (hoặc port được cấu hình trong `APP_PORT`).

### Testing API với Swagger

Sau khi service đã chạy, bạn có thể test API thông qua Swagger UI:

1. **Truy cập Swagger UI:**

   ```
   http://localhost:8080/swagger/index.html
   ```

2. **Authenticate (nếu cần):**
   - Swagger hỗ trợ cả Cookie và Bearer token authentication
   - Click vào nút "Authorize" ở góc trên bên phải
   - Nhập token hoặc cookie để authenticate

3. **Test các endpoints:**
   - Xem danh sách tất cả endpoints có sẵn
   - Click "Try it out" trên bất kỳ endpoint nào
   - Điền thông tin và click "Execute" để test
   - Xem response trực tiếp trong Swagger UI

**Lưu ý:** Để sử dụng Swagger với authentication:

- **Cookie Auth**: Đảm bảo bạn đã login qua Identity service và cookie được lưu trong browser
- **Bearer Token**: Nhập token vào phần "Authorize" trong Swagger UI

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
