# Migrate Project Service to HttpOnly Cookie Authentication

## Overview

This change migrates the Project service from Bearer token authentication to HttpOnly cookie authentication, following the same secure pattern successfully implemented in the Identity service. The migration will enable cookie-based authentication while maintaining backward compatibility with Bearer tokens during the transition period.

## Why

The Identity service has successfully migrated to HttpOnly cookie authentication, establishing a secure, centralized authentication pattern for the SMAP platform. The Project service must adopt this same pattern to:

1. **Maintain Security Consistency**: HttpOnly cookies prevent XSS attacks by making authentication tokens inaccessible to JavaScript, providing better security than tokens stored in localStorage
2. **Enable Seamless User Experience**: Browsers automatically manage cookies, eliminating the need for manual token handling in frontend code
3. **Support Cross-Service Authentication**: Shared cookie configuration (same domain, same JWT secret) allows users to authenticate once via Identity service and access all SMAP microservices
4. **Align with Platform Standards**: All SMAP microservices should use the same authentication mechanism for consistency and maintainability

Without this migration, the Project service remains isolated with Bearer token authentication, creating:
- **Inconsistent Security Posture**: Different services using different authentication methods
- **Poor User Experience**: Users may need to authenticate separately for different services
- **Maintenance Burden**: Multiple authentication patterns to maintain and secure

This change is a critical step in the platform-wide migration to HttpOnly cookie authentication.

## Background

The Identity service has completed its migration to HttpOnly cookie authentication, which provides:
- **Enhanced Security**: HttpOnly cookies prevent XSS attacks by making tokens inaccessible to JavaScript
- **Automatic Cookie Management**: Browsers handle cookie storage and transmission automatically
- **Simplified Frontend**: No manual token management required
- **Shared Authentication**: All microservices share the same JWT secret and cookie configuration

The Project service currently only supports Bearer token authentication via the `Authorization` header. This change will add cookie-based authentication as the primary method while keeping Bearer tokens as a fallback for backward compatibility.

## User Review Required

> [!IMPORTANT]
> **Backward Compatibility Period**
> 
> This implementation maintains Bearer token authentication as a fallback during the migration period. The middleware will:
> 1. First attempt to read the token from the cookie (preferred)
> 2. Fall back to the Authorization header if no cookie is present
> 
> **Future Work**: After all clients have migrated to cookie authentication, a follow-up change should remove the Bearer token fallback to complete the migration.

> [!WARNING]
> **Configuration Requirements**
> 
> All SMAP microservices must use **identical** cookie configuration:
> - **Cookie Name**: `smap_auth_token` (must match across all services)
> - **Cookie Domain**: `.smap.com` (enables cookie sharing across subdomains)
> - **JWT Secret**: Must match the Identity service secret (for token validation)
> - **CORS Origins**: Must include all frontend domains with `AllowCredentials: true`
> 
> Mismatched configuration will break cross-service authentication.

## Proposed Changes

### Authentication & Security

#### [MODIFY] [config.go](file:///Users/tantai/Workspaces/smap/smap-api/project/config/config.go)

Add `CookieConfig` struct to the main `Config` struct to store HttpOnly cookie settings. This includes domain, security flags (Secure, SameSite), cookie name, and max-age values for normal and "remember me" sessions.

**Changes**:
- Add `Cookie CookieConfig` field to `Config` struct
- Add new `CookieConfig` struct with fields: `Domain`, `Secure`, `SameSite`, `MaxAge`, `MaxAgeRemember`, `Name`
- All fields use environment variable bindings with sensible defaults

---

### Middleware Layer

#### [MODIFY] [new.go](file:///Users/tantai/Workspaces/smap/smap-api/project/internal/middleware/new.go)

Update the middleware constructor to accept and store the cookie configuration, which will be used by the authentication middleware to extract tokens from cookies.

**Changes**:
- Add `cookieConfig config.CookieConfig` field to `Middleware` struct
- Update `New()` constructor to accept `cookieConfig` parameter
- Store cookie config in the middleware instance

#### [MODIFY] [middleware.go](file:///Users/tantai/Workspaces/smap/smap-api/project/internal/middleware/middleware.go)

Implement cookie-first authentication strategy in the `Auth()` middleware. The middleware will prioritize reading tokens from cookies, falling back to the Authorization header for backward compatibility.

**Changes**:
- Update `Auth()` to first attempt reading token from cookie using `c.Cookie(m.cookieConfig.Name)`
- If cookie is not present or empty, fall back to Authorization header (existing behavior)
- Add TODO comment noting that Bearer token fallback should be removed after client migration
- Keep all existing JWT verification and context setting logic unchanged

#### [MODIFY] [cors.go](file:///Users/tantai/Workspaces/smap/smap-api/project/internal/middleware/cors.go)

Update CORS configuration to enable credential support, which is required for browsers to send cookies with cross-origin requests.

**Changes**:
- Replace wildcard `AllowedOrigins: []string{"*"}` with specific origins:
  - `http://localhost`, `http://127.0.0.1`
  - `http://localhost:3000`, `http://127.0.0.1:3000`
  - `https://smap.tantai.dev`
  - `https://smap-api.tantai.dev`
  - `https://smap-api.tantai.dev/*`
- Change `AllowCredentials: false` to `AllowCredentials: true`

---

### HTTP Server Integration

#### [MODIFY] [new.go](file:///Users/tantai/Workspaces/smap/smap-api/project/internal/httpserver/new.go)

Add cookie configuration to the HTTP server struct and config, enabling the server to pass cookie settings to the middleware layer.

**Changes**:
- Add `cookieConfig config.CookieConfig` field to `HTTPServer` struct
- Add `CookieConfig config.CookieConfig` field to `Config` struct
- Update `New()` constructor to accept and store `cfg.CookieConfig`

#### [MODIFY] [handler.go](file:///Users/tantai/Workspaces/smap/smap-api/project/internal/httpserver/handler.go)

Wire the cookie configuration through to the middleware when initializing the middleware instance.

**Changes**:
- Update `middleware.New()` call in `mapHandlers()` to pass `srv.cookieConfig` as third parameter

---

### Application Entry Point

#### [MODIFY] [main.go](file:///Users/tantai/Workspaces/smap/smap-api/project/cmd/api/main.go)

Pass the cookie configuration from the loaded config to the HTTP server initialization.

**Changes**:
- Add `CookieConfig: cfg.Cookie` to the `httpserver.Config` struct initialization
- Update Swagger annotations to document both cookie and Bearer authentication methods

**Swagger Security Definitions**:
```go
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by Identity service /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
```

---

### Configuration

#### [MODIFY] [template.env](file:///Users/tantai/Workspaces/smap/smap-api/project/template.env)

Add cookie configuration environment variables that match the Identity service configuration.

**Changes**:
- Add new section: `# Cookie Configuration (for HttpOnly authentication)`
- Add variables:
  - `COOKIE_DOMAIN=.smap.com`
  - `COOKIE_SECURE=true`
  - `COOKIE_SAMESITE=Lax`
  - `COOKIE_MAX_AGE=7200` (2 hours)
  - `COOKIE_MAX_AGE_REMEMBER=2592000` (30 days)
  - `COOKIE_NAME=smap_auth_token`

## Verification Plan

### Automated Tests

**Unit Tests** (if they exist):
```bash
# Check for existing tests
find . -name "*_test.go" -path "*/middleware/*" -o -path "*/httpserver/*"

# Run middleware tests
go test ./internal/middleware/... -v

# Run httpserver tests
go test ./internal/httpserver/... -v
```

**Note**: The current codebase has minimal test files (only in `pkg/` directory). New tests are not proposed as part of this change to keep scope focused.

### Manual Verification

#### 1. Test Cookie Authentication Flow

**Prerequisites**:
- Identity service running and accessible
- Project service running with new cookie configuration
- Valid test user credentials

**Steps**:
```bash
# Step 1: Login via Identity service to get cookie
curl -i -X POST https://smap-api.tantai.dev/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","remember":false}' \
  -c cookies.txt

# Expected: 200 OK with Set-Cookie header containing smap_auth_token

# Step 2: Call Project service endpoint with cookie
curl -X GET https://smap-api.tantai.dev/project/projects \
  -b cookies.txt

# Expected: 200 OK with list of projects (authenticated response)
```

#### 2. Test Backward Compatibility (Bearer Token)

**Steps**:
```bash
# Get a valid JWT token (from previous login or existing token)
TOKEN="<your-jwt-token>"

# Call Project service with Authorization header
curl -X GET https://smap-api.tantai.dev/project/projects \
  -H "Authorization: Bearer $TOKEN"

# Expected: 200 OK with list of projects (Bearer token still works)
```

#### 3. Test CORS with Credentials

**Steps**:
```javascript
// In browser console on https://smap.tantai.dev
// First login to get cookie, then:

fetch('https://smap-api.tantai.dev/project/projects', {
  credentials: 'include'  // Required for cookies
})
.then(r => r.json())
.then(console.log);

// Expected: No CORS errors, successful response with projects data
```

#### 4. Verify Cookie Attributes

**Steps**:
```bash
# Check Set-Cookie header from Identity service login
curl -i -X POST https://smap-api.tantai.dev/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | grep -i "set-cookie"

# Expected output should contain:
# - Cookie name: smap_auth_token
# - HttpOnly flag
# - Secure flag
# - SameSite=Lax
# - Max-Age=7200 (2 hours for normal login)
```

#### 5. Test Unauthorized Access

**Steps**:
```bash
# Call Project service without cookie or token
curl -X GET https://smap-api.tantai.dev/project/projects

# Expected: 401 Unauthorized
```

### Verification Checklist

- [ ] Cookie authentication works (login via Identity, access Project service)
- [ ] Bearer token fallback still works (Authorization header)
- [ ] CORS with credentials enabled (no browser errors)
- [ ] Cookie has correct attributes (HttpOnly, Secure, SameSite=Lax)
- [ ] Unauthorized requests return 401
- [ ] Swagger documentation updated with both auth methods
- [ ] Environment variables documented in template.env
