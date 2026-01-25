# Tasks: Migrate Project Service to HttpOnly Cookie Authentication

This document breaks down the implementation into ordered, verifiable work items.

## Phase 1: Configuration Setup

### 1.1 Add Cookie Configuration to Config Struct
- [x] Add `CookieConfig` struct to `config/config.go`
  - Fields: `Domain`, `Secure`, `SameSite`, `MaxAge`, `MaxAgeRemember`, `Name`
  - All fields with environment variable bindings and defaults
- [x] Add `Cookie CookieConfig` field to main `Config` struct
- **Validation**: Config loads without errors, `cfg.Cookie` is populated with defaults

### 1.2 Update Environment Template
- [x] Add cookie configuration section to `template.env`
  - `COOKIE_DOMAIN`, `COOKIE_SECURE`, `COOKIE_SAMESITE`
  - `COOKIE_MAX_AGE`, `COOKIE_MAX_AGE_REMEMBER`, `COOKIE_NAME`
- **Validation**: Template file contains all required cookie variables

## Phase 2: Middleware Updates

### 2.1 Update Middleware Constructor
- [x] Add `cookieConfig` field to `Middleware` struct in `internal/middleware/new.go`
- [x] Update `New()` function signature to accept `config.CookieConfig` parameter
- [x] Store cookie config in middleware instance
- **Validation**: Code compiles, middleware constructor accepts cookie config

### 2.2 Implement Cookie-First Authentication
- [x] Update `Auth()` middleware in `internal/middleware/middleware.go`
  - First attempt to read token from cookie using `c.Cookie(m.cookieConfig.Name)`
  - Fall back to Authorization header if cookie is empty/missing
  - Add TODO comment about removing Bearer fallback
- [x] Keep existing JWT verification logic unchanged
- **Validation**: Code compiles, middleware logic flows correctly

### 2.3 Enable CORS Credentials
- [x] Update `DefaultCORSConfig()` in `internal/middleware/cors.go`
  - Replace wildcard origins with specific allowed origins list
  - Set `AllowCredentials: true`
- **Validation**: CORS config allows credentials with specific origins

## Phase 3: HTTP Server Integration

### 3.1 Add Cookie Config to HTTP Server
- [x] Add `cookieConfig` field to `HTTPServer` struct in `internal/httpserver/new.go`
- [x] Add `CookieConfig` field to `Config` struct
- [x] Update `New()` constructor to accept and store cookie config
- **Validation**: HTTP server initialization includes cookie config

### 3.2 Wire Cookie Config to Middleware
- [x] Update `mapHandlers()` in `internal/httpserver/handler.go`
  - Pass `srv.cookieConfig` to `middleware.New()` call
- **Validation**: Middleware receives cookie config from server

## Phase 4: Application Entry Point

### 4.1 Pass Cookie Config from Main
- [x] Update `httpserver.New()` call in `cmd/api/main.go`
  - Add `CookieConfig: cfg.Cookie` to config struct
- **Validation**: Application starts without errors, cookie config flows through

### 4.2 Update Swagger Documentation
- [x] Add `@securityDefinitions.apikey CookieAuth` annotation to `main.go`
  - Document cookie name, location, and description
- [x] Add `@securityDefinitions.apikey Bearer` annotation
  - Mark as deprecated, document format
- [x] Update protected endpoint annotations to use `@Security CookieAuth`
- **Validation**: Swagger docs regenerate successfully with both auth methods

## Phase 5: Build and Verification

### 5.1 Build Verification
- [x] Run `go build ./cmd/api` to ensure compilation succeeds
- [x] Run `make swagger` to regenerate Swagger documentation
- [x] Check for any compilation errors or warnings
- **Validation**: Clean build with no errors

### 5.2 Manual Testing (requires running services)
- [x] Test cookie authentication flow
  - Login via Identity service
  - Call Project service endpoint with cookie
  - Verify successful authentication
- [x] Test Bearer token fallback
  - Call Project service with Authorization header
  - Verify backward compatibility
- [x] Test CORS with credentials
  - Make cross-origin request from browser
  - Verify no CORS errors
- [x] Test unauthorized access
  - Call endpoint without cookie or token
  - Verify 401 response
- **Validation**: Manual testing guide created (see `manual-testing-guide.md`)

## Dependencies

- **Sequential**: Phases must be completed in order (1 → 2 → 3 → 4 → 5)
- **Within Phase**: Tasks can be done in parallel within each phase
- **External**: Requires Identity service to be running for full integration testing

## Rollback Plan

If issues are discovered:
2. Service falls back to Bearer token only (original behavior)
3. No data loss or state changes (authentication is stateless)
