# collector-service Specification

## Purpose
TBD - created by archiving change implement-keyword-llm-integration. Update Purpose after archive.
## Requirements
### Requirement: Collector Service Client Abstraction

The system SHALL provide an abstraction layer for Collector Service through `pkg/collector/Client` interface. The abstraction MUST allow calling Collector Service for dry run operations without coupling business logic to HTTP implementation details.

#### Scenario: Client Interface Definition

Given the system has Collector service package
When examining the `pkg/collector/Client` interface
Then it defines `DryRun(ctx, keywords, limit) ([]Post, error)`
And the method accepts `context.Context` for cancellation and timeout
And the method accepts keywords as array of strings
And the method accepts limit as integer (max posts to return)
And the method returns array of `Post` structs

#### Scenario: Post Data Structure

Given the Collector service package
When examining the `Post` struct
Then it includes the following fields:
- `ID` (string): Unique post identifier
- `Content` (string): The post text content
- `Source` (string): The source platform name (e.g., "facebook", "twitter", "reddit")
- `SourceID` (string): Original post ID on the platform
- `Author` (string): Author name or username
- `AuthorID` (string): Author identifier on the platform
- `Date` (time.Time): Post creation date in ISO 8601 format
- `URL` (string): Direct link to the original post
- `Engagement` (struct): Engagement metrics (likes, comments, shares)
- `Metadata` (struct): Additional metadata (language, sentiment)

#### Scenario: Engagement Structure

Given the Collector service package
When examining the `Engagement` struct
Then it includes:
- `Likes` (int): Number of likes
- `Comments` (int): Number of comments
- `Shares` (int): Number of shares

#### Scenario: Metadata Structure

Given the Collector service package
When examining the `Metadata` struct
Then it includes:
- `Language` (string): Post language code (e.g., "vi", "en")
- `Sentiment` (string): Sentiment analysis result ("positive", "negative", "neutral")

### Requirement: HTTP Client Implementation

The system SHALL implement Collector Service client using HTTP client. The implementation MUST make HTTP POST requests to Collector Service API endpoint.

#### Scenario: HTTP Client Initialization

Given `COLLECTOR_SERVICE_URL` is configured with base URL
And `COLLECTOR_TIMEOUT=30` (seconds)
When the system initializes Collector client
Then HTTP client is created with configured timeout
And base URL is stored for constructing full endpoint URLs

#### Scenario: Dry Run API Call

Given Collector client is initialized with `COLLECTOR_SERVICE_URL=http://localhost:8081`
When `DryRun` is called with keywords ["VinFast", "VF3"] and limit=10
Then the client constructs HTTP POST request to `http://localhost:8081/api/v1/collector/dry-run`
And request body contains JSON:
```json
{
  "keywords": ["VinFast", "VF3"],
  "limit": 10
}
```
And request includes `Content-Type: application/json` header
And request may include authentication headers if required by Collector Service

#### Scenario: Mock URL for Development

Given the system is in development environment
When `COLLECTOR_SERVICE_URL` is not set or set to empty
Then the system uses default mock URL `http://localhost:8081/api/v1/collector/dry-run`
And logs a warning that mock URL is being used
And the system can still function for testing purposes

#### Scenario: Dry Run Response Parsing

Given Collector client receives successful HTTP response
When response status is 200 OK
And response body contains:
```json
{
  "posts": [
    {
      "id": "post-uuid-123",
      "content": "Just bought a new VinFast VF3!",
      "source": "facebook",
      "source_id": "fb_post_123456",
      "author": "John Doe",
      "author_id": "fb_user_789",
      "date": "2025-01-27T10:00:00Z",
      "url": "https://facebook.com/posts/123456",
      "engagement": {
        "likes": 42,
        "comments": 5,
        "shares": 3
      },
      "metadata": {
        "language": "vi",
        "sentiment": "positive"
      }
    }
  ],
  "total_found": 150,
  "limit": 10
}
```
Then the client parses JSON response body
And extracts `posts` array from response
And converts each post object to `Post` struct with all fields
And returns array of `Post` structs
And ignores `total_found` and `limit` fields (for future use)

#### Scenario: Dry Run Empty Results

Given Collector client receives successful HTTP response
When response contains empty `posts` array
Then the client returns empty array `[]Post{}`
And no error is returned (empty results are valid)

### Requirement: Collector Service Error Handling

The system SHALL handle all error scenarios when calling Collector Service. Errors MUST be mapped to structured error types for proper handling by calling code.

#### Scenario: Network Error

Given Collector client makes API call
When network error occurs (connection refused, DNS failure, etc.)
Then the client returns `ErrCollectorUnavailable` error
And logs the error with full context

#### Scenario: HTTP Timeout

Given Collector client is configured with timeout=30s
When API call does not complete within 30 seconds
Then the client cancels the request
And returns `ErrCollectorTimeout` error
And logs the timeout

#### Scenario: HTTP 4xx Errors

Given Collector client makes API call
When Collector Service returns 4xx status (e.g., 400 Bad Request, 401 Unauthorized)
Then the client returns appropriate error
And logs the error with status code and response body
And error message indicates client error (not retryable)

#### Scenario: HTTP 5xx Errors

Given Collector client makes API call
When Collector Service returns 5xx status (e.g., 500 Internal Server Error, 503 Service Unavailable)
Then the client returns `ErrCollectorUnavailable` error
And logs the error with status code and response body
And error message indicates server error (potentially retryable)

#### Scenario: Invalid JSON Response

Given Collector client receives HTTP 200 response
When response body is not valid JSON
Then the client returns `ErrCollectorInvalidResponse` error
And logs the response body for debugging

#### Scenario: Missing Required Fields in Response

Given Collector client receives HTTP 200 response
When response JSON does not contain `posts` field
Then the client returns `ErrCollectorInvalidResponse` error
And logs the response structure issue

### Requirement: Collector Service Configuration

The system SHALL support configuration of Collector Service through environment variables. Base URL MUST be required for Collector Service functionality.

#### Scenario: Required Configuration with Default

Given the system initializes Collector client
When `COLLECTOR_SERVICE_URL` is not set
Then the system uses default mock URL `http://localhost:8081`
And logs a warning that default mock URL is being used
And Collector client is available for development/testing
And production deployments MUST set `COLLECTOR_SERVICE_URL` explicitly

#### Scenario: Production Configuration

Given the system is in production environment
When `COLLECTOR_SERVICE_URL` is set to production Collector Service URL
Then the system uses the configured URL
And no warnings are logged
And Collector client connects to production service

#### Scenario: Optional Configuration with Defaults

Given the system initializes Collector client
When `COLLECTOR_TIMEOUT` is not set
Then the system uses default 30 seconds
And HTTP client timeout is set to 30 seconds

#### Scenario: Invalid URL Configuration

Given the system initializes Collector client
When `COLLECTOR_SERVICE_URL` is set to invalid URL format
Then the system logs an error
And Collector client initialization fails
And error is returned from constructor

### Requirement: Collector Service Error Types

The system SHALL define structured error types for Collector Service failures. All errors MUST be exported and usable by calling code.

#### Scenario: Error Type Definitions

Given the Collector service package
When examining error definitions
Then `ErrCollectorUnavailable` is defined for service unavailable
And `ErrCollectorTimeout` is defined for timeout errors
And `ErrCollectorInvalidResponse` is defined for invalid responses
And all errors are exported and can be checked with `errors.Is()`

#### Scenario: Error Context

Given Collector service returns an error
When the error is logged or returned
Then the error includes context about the operation (dry run)
And the error includes relevant details (keywords, limit, etc.)
And the error can be wrapped with additional context

### Requirement: HTTP Client Configuration

The system SHALL configure HTTP client for Collector Service API calls with appropriate timeout and connection pooling settings.

#### Scenario: HTTP Client Timeout

Given Collector client is initialized with `COLLECTOR_TIMEOUT=30`
When HTTP client is created
Then the client has request timeout of 30 seconds
And requests are cancelled if they exceed the timeout

#### Scenario: HTTP Client Connection Pooling

Given Collector client is initialized
When HTTP client is created
Then the client uses connection pooling
And idle connections are reused
And connection pool size is configured appropriately

#### Scenario: HTTP Client Headers

Given Collector client makes API calls
When HTTP request is sent
Then the request includes `Content-Type: application/json` header
And the request includes appropriate `User-Agent` header
And the request may include authentication headers if required by Collector Service

### Requirement: Context Propagation

The system SHALL properly propagate context through Collector Service calls for cancellation, timeout, and request tracing.

#### Scenario: Context Cancellation

Given Collector client receives context with cancellation
When `DryRun` is called with cancelled context
Then the HTTP request is not sent
And the method immediately returns context cancellation error

#### Scenario: Context Timeout

Given Collector client receives context with timeout
When `DryRun` is called
And the operation exceeds context timeout
Then the HTTP request is cancelled
And the method returns context deadline exceeded error

#### Scenario: Request ID Propagation

Given Collector client receives context with request ID
When `DryRun` is called
Then the request ID is included in logs
And the request ID can be used for tracing across services

