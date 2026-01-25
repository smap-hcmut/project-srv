# Authentication Capability

## MODIFIED Requirements

### Requirement: HttpOnly cookie-based authentication

The Project service SHALL support HttpOnly cookie-based authentication as the primary authentication method, while maintaining backward compatibility with Bearer token authentication during the migration period.

#### Scenario: User authenticates with HttpOnly cookie from Identity service

**Given** a user has logged in via the Identity service and received a `smap_auth_token` cookie  
**When** the user makes a request to any protected Project service endpoint  
**Then** the middleware SHALL extract the JWT token from the `smap_auth_token` cookie  
**And** the middleware SHALL validate the JWT token using the shared JWT secret  
**And** the request SHALL be authenticated successfully  
**And** the user context SHALL be populated with the JWT payload

#### Scenario: User authenticates with Bearer token (backward compatibility)

**Given** a user has a valid JWT token  
**And** no `smap_auth_token` cookie is present in the request  
**When** the user makes a request with the `Authorization: Bearer <token>` header  
**Then** the middleware SHALL extract the JWT token from the Authorization header  
**And** the middleware SHALL validate the JWT token  
**And** the request SHALL be authenticated successfully  
**And** the user context SHALL be populated with the JWT payload

#### Scenario: Cookie authentication takes precedence over Bearer token

**Given** a user has both a valid `smap_auth_token` cookie and an `Authorization` header  
**When** the user makes a request to a protected endpoint  
**Then** the middleware SHALL use the token from the cookie  
**And** the middleware SHALL ignore the Authorization header

#### Scenario: Request without authentication credentials is rejected

**Given** a user makes a request without a cookie or Authorization header  
**When** the request reaches a protected endpoint  
**Then** the middleware SHALL return a 401 Unauthorized response  
**And** the request SHALL be aborted before reaching the handler

### Requirement: CORS with credentials

The Project service SHALL configure CORS to allow cross-origin requests with credentials (cookies) from authorized frontend origins.

#### Scenario: Browser sends cross-origin request with credentials

**Given** a frontend application is running on an allowed origin (e.g., `https://smap.tantai.dev`)  
**And** the user has a valid `smap_auth_token` cookie  
**When** the frontend makes a cross-origin request with `credentials: 'include'`  
**Then** the server SHALL respond with `Access-Control-Allow-Origin` header matching the request origin  
**And** the server SHALL respond with `Access-Control-Allow-Credentials: true` header  
**And** the cookie SHALL be included in the request  
**And** the request SHALL be authenticated successfully

#### Scenario: Preflight request for protected endpoint

**Given** a browser is about to make a cross-origin request with credentials  
**When** the browser sends an OPTIONS preflight request  
**Then** the server SHALL respond with appropriate CORS headers  
**And** the response SHALL include `Access-Control-Allow-Credentials: true`  
**And** the response SHALL include `Access-Control-Allow-Methods` with supported HTTP methods  
**And** the response SHALL include `Access-Control-Allow-Headers` with required headers

### Requirement: Shared cookie configuration

The Project service SHALL use identical cookie configuration as the Identity service to enable cookie sharing across SMAP microservices.

#### Scenario: Cookie configuration matches Identity service

**Given** the Identity service sets a cookie with specific attributes  
**When** the Project service validates the cookie  
**Then** the cookie name SHALL be `smap_auth_token`  
**And** the cookie domain SHALL be `.smap.com`  
**And** the JWT secret SHALL match the Identity service secret  
**And** the token SHALL be validated successfully

#### Scenario: Service reads cookie configuration from environment

**Given** the service is starting up  
**When** the configuration is loaded  
**Then** the cookie configuration SHALL be read from environment variables  
**And** the cookie name SHALL default to `smap_auth_token` if not specified  
**And** the cookie domain SHALL default to `.smap.com` if not specified  
**And** the cookie security flags SHALL default to `Secure=true` and `SameSite=Lax`
