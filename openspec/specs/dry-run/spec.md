# dry-run Specification

## Purpose
TBD - created by archiving change enforce-smart-keywords. Update Purpose after archive.
## Requirements
### Requirement: Fetch Sample Data

The system SHALL fetch a sample of real data (5-10 items) based on the provided keywords by calling the Collector Service. The system MUST integrate with the Collector Service API to retrieve actual posts matching the keywords. The system SHALL validate keywords before calling the Collector Service.

#### Scenario: Successful Dry Run Execution

Given a list of keywords ["VinFast", "VF3"]
When the user requests a dry run via `POST /projects/keywords/dry-run`
Then the system validates the keywords using keyword validation rules
And the system calls Collector Service with validated keywords and limit=10
And Collector Service returns 5-10 recent posts matching the keywords
And the system returns the posts to the user
And each post contains content, source, date, and other relevant fields

#### Scenario: Dry Run with Invalid Keywords

Given a list of keywords containing invalid entries (e.g., too short, invalid characters)
When the user requests a dry run
Then the system validates the keywords first
And invalid keywords are rejected with appropriate error messages
And the Collector Service is NOT called
And the API returns validation errors

#### Scenario: Collector Service Unavailable

Given a list of valid keywords ["VinFast", "VF3"]
When the user requests a dry run
And the Collector Service is unavailable (network error, service down)
Then the system logs an error with full context
And the API returns an error response indicating Collector Service is unavailable
And the error message is user-friendly (does not expose internal details)

#### Scenario: Collector Service Timeout

Given a list of valid keywords ["VinFast", "VF3"]
When the user requests a dry run
And the Collector Service does not respond within the configured timeout
Then the system cancels the request
And logs a timeout error
And the API returns an error response indicating timeout

#### Scenario: Collector Service Invalid Response

Given a list of valid keywords ["VinFast", "VF3"]
When the user requests a dry run
And the Collector Service returns an invalid or malformed response
Then the system logs an error with the response details
And the API returns an error response indicating invalid response from Collector Service

#### Scenario: Collector Service Returns Empty Results

Given a list of valid keywords ["VinFast", "VF3"]
When the user requests a dry run
And the Collector Service returns an empty list of posts
Then the system returns an empty list to the user
And no error is returned (empty results are valid)

#### Scenario: Collector Service Returns Partial Results

Given a list of valid keywords ["VinFast", "VF3"]
When the user requests a dry run with limit=10
And the Collector Service returns only 3 posts (less than requested)
Then the system returns the 3 posts to the user
And no error is returned (partial results are valid)

### Requirement: Visualize Results

The system SHALL display the fetched sample data to the user for verification. The system MUST return posts in a structured format that can be easily displayed in the UI.

#### Scenario: Display Sample

Given a successful dry run response
When the data is received from Collector Service
Then the system formats the posts into a consistent structure
And each post includes at minimum: content, source, date
And the API returns the formatted posts in the response
And the UI can display the posts to the user

#### Scenario: Post Data Structure

Given a dry run response
When the system returns posts
Then each post is a structured object with fields:
- `content` (string): The post content/text
- `source` (string): The source platform (e.g., "facebook", "twitter", "reddit")
- `date` (string): ISO 8601 formatted date/time
- Additional fields as provided by Collector Service

### Requirement: Collector Service Integration

The system SHALL integrate with Collector Service (external microservice) via HTTP API to fetch dry run data. The system MUST support configuration of Collector Service URL and timeout. The system SHALL use a default mock URL for development when URL is not configured.

#### Scenario: Collector Service Configuration

Given the system is configured with `COLLECTOR_SERVICE_URL=http://collector-service:8080` pointing to valid Collector Service
And `COLLECTOR_TIMEOUT=30` (seconds)
When the system initializes
Then the Collector client is configured with the base URL and timeout
And dry run requests use the configured settings
And requests are sent to `http://collector-service:8080/api/v1/collector/dry-run`

#### Scenario: Development Mock URL

Given the system is in development environment
When `COLLECTOR_SERVICE_URL` is not set
Then the system uses default mock URL `http://localhost:8081/api/v1/collector/dry-run`
And logs a warning that mock URL is being used
And the system can function for testing purposes

#### Scenario: Collector Service URL Missing - Uses Default

Given `COLLECTOR_SERVICE_URL` is not configured
When the system initializes
Then the system uses default mock URL `http://localhost:8081`
And logs a warning that default mock URL is being used
And dry run requests will use the mock URL (for development/testing)
And production deployments should set `COLLECTOR_SERVICE_URL` explicitly
And the error message indicates Collector Service is not configured

#### Scenario: Collector Service API Contract

Given the system calls Collector Service
When making a dry run request
Then the system sends HTTP POST request to `{COLLECTOR_SERVICE_URL}/api/v1/collector/dry-run`
And the request body contains JSON:
```json
{
  "keywords": ["VinFast", "VF3"],
  "limit": 10
}
```
Where:
- `keywords` (array of strings): The keywords to search for
- `limit` (integer): Maximum number of posts to return (default: 10)
And the system expects response (200 OK) with JSON:
```json
{
  "posts": [
    {
      "id": "post-uuid-123",
      "content": "Post content...",
      "source": "facebook",
      "source_id": "fb_post_123",
      "author": "Author Name",
      "author_id": "author_123",
      "date": "2025-01-27T10:00:00Z",
      "url": "https://facebook.com/posts/123",
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
Where:
- `posts` (array): Array of post objects with all fields
- `total_found` (integer, optional): Total number of posts found (for information)
- `limit` (integer, optional): Limit used in request (for information)

### Requirement: Error Handling for Dry Run

The system SHALL handle all error scenarios gracefully when calling Collector Service. The system MUST provide clear error messages to users without exposing internal implementation details.

#### Scenario: Network Error Handling

Given a dry run request
When Collector Service is unreachable (network error)
Then the system logs the error with full context
And the API returns 503 (Service Unavailable) or 500 (Internal Server Error)
And the error message indicates the service is temporarily unavailable

#### Scenario: HTTP Error Response Handling

Given a dry run request
When Collector Service returns 4xx or 5xx HTTP status
Then the system logs the error with status code and response body
And the API returns appropriate error response
And the error message is user-friendly

#### Scenario: Response Parsing Error

Given a dry run request
When Collector Service returns invalid JSON
Then the system logs the error with response details
And the API returns 502 (Bad Gateway) or 500 (Internal Server Error)
And the error message indicates invalid response from service

