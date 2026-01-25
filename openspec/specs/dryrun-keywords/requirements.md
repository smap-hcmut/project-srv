# Requirements Document

## Introduction

The DryRunKeywords feature enables users to preview social media content for a set of keywords before committing to a full project crawl. This feature implements a real-time, asynchronous workflow where the Project service receives keyword validation requests, dispatches crawl tasks to the Collector service via RabbitMQ, receives results via webhook callbacks, and pushes real-time updates to connected clients through the WebSocket service using Redis Pub/Sub.

## Glossary

- **Project Service**: The HTTP API service that handles user requests for keyword dry-run operations
- **Collector Service**: The background worker service that performs actual web crawling of social media platforms
- **WebSocket Service**: The real-time communication service that maintains persistent connections with clients
- **CrawlRequest**: A message structure sent to the Collector service containing task details
- **DryRunTask**: A specific task type that crawls a limited number of posts per keyword for preview purposes
- **Job**: A unique instance of a dry-run operation identified by job_id
- **Callback Webhook**: An HTTP endpoint that receives results from the Collector service
- **Redis Pub/Sub**: A publish-subscribe messaging pattern used to fan-out messages to WebSocket clients
- **Platform**: A social media platform (YouTube or TikTok) that can be crawled
- **Task Type**: A classification of crawler operations (e.g., dryrun_keyword, research_and_crawl)

## Requirements

### Requirement 1

**User Story:** As a user, I want to initiate a keyword dry-run operation, so that I can preview social media content before creating a full project.

#### Acceptance Criteria

1. WHEN a user sends a POST request to `/projects/keywords/dry-run` with valid keywords, THEN the Project Service SHALL validate the keywords and return a unique job_id with status "processing"
2. WHEN the Project Service receives a dry-run request, THEN the Project Service SHALL publish a CrawlRequest message to the RabbitMQ exchange `collector.inbound` with routing key `crawler.dryrun_keyword`
3. WHEN building the CrawlRequest, THEN the Project Service SHALL include keywords, limit_per_keyword of 3, include_comments as true, and max_comments of 5
4. WHEN a user provides invalid keywords, THEN the Project Service SHALL return an error response without creating a job
5. WHEN the Project Service creates a job, THEN the Project Service SHALL generate a unique UUID as the job_id

### Requirement 2

**User Story:** As the Collector service, I want to process dry-run tasks, so that I can crawl limited content and return results to the Project service.

#### Acceptance Criteria

1. WHEN the Collector service receives a message with task_type "dryrun_keyword", THEN the Collector service SHALL recognize it as a valid task type
2. WHEN processing a dryrun_keyword task, THEN the Collector service SHALL crawl a maximum of 3 posts per keyword per platform
3. WHEN crawling posts for a dry-run task, THEN the Collector service SHALL include a maximum of 5 comments per post
4. WHEN the Collector service completes crawling for a platform, THEN the Collector service SHALL send results to the Project service webhook endpoint
5. WHEN the Collector service encounters an error during crawling, THEN the Collector service SHALL send an error notification to the Project service webhook endpoint
6. WHEN sending webhook callbacks, THEN the Collector service SHALL implement retry logic with exponential backoff for failed requests
7. WHEN the Collector service sends a webhook callback, THEN the Collector service SHALL include authentication credentials in the request

### Requirement 3

**User Story:** As the Project service, I want to receive crawl results via webhook, so that I can distribute them to connected clients.

#### Acceptance Criteria

1. WHEN the Project service receives a POST request to `/internal/collector/dryrun/callback`, THEN the Project service SHALL validate the authentication credentials
2. WHEN the webhook receives a callback with a job_id, THEN the Project service SHALL accept it without database validation
3. WHEN the webhook receives results with status "success", THEN the Project service SHALL extract the posts and publish them to Redis
4. WHEN the webhook receives results with status "failed", THEN the Project service SHALL extract error information and publish it to Redis
5. WHEN publishing to Redis, THEN the Project service SHALL use the Pub/Sub pattern with channel format `user_noti:{user_id}`
6. WHEN an unauthenticated request arrives at the webhook endpoint, THEN the Project service SHALL reject it with a 401 Unauthorized response

### Requirement 4

**User Story:** As the WebSocket service, I want to receive dry-run events from Redis, so that I can push real-time updates to connected clients.

#### Acceptance Criteria

1. WHEN the WebSocket service subscribes to Redis, THEN the WebSocket service SHALL listen to the pattern `user_noti:*`
2. WHEN a dry-run result message arrives on a user channel, THEN the WebSocket service SHALL parse the message payload
3. WHEN the WebSocket service receives a valid dry-run message, THEN the WebSocket service SHALL identify all active connections for that user_id
4. WHEN the WebSocket service identifies active connections, THEN the WebSocket service SHALL send the message to all connections for that user
5. WHEN a dry-run message contains platform-specific results, THEN the WebSocket service SHALL include the platform identifier in the message sent to clients
6. WHEN the WebSocket service receives a malformed message, THEN the WebSocket service SHALL log the error without crashing

### Requirement 5

**User Story:** As a developer, I want clear separation between services, so that the system remains maintainable and each service has well-defined responsibilities.

#### Acceptance Criteria

1. WHEN the Project service publishes to RabbitMQ, THEN the Project service SHALL use the standard CrawlRequest message format
2. WHEN the Collector service sends callbacks, THEN the Collector service SHALL use the standard webhook payload format
3. WHEN the Project service publishes to Redis, THEN the Project service SHALL use the standard RedisMessage format
4. WHEN services communicate, THEN the services SHALL use JSON serialization for all message payloads
5. WHEN the Collector service processes tasks, THEN the Collector service SHALL remain independent of Project service implementation details

### Requirement 6

**User Story:** As a system administrator, I want proper error handling and monitoring, so that I can diagnose issues and ensure system reliability.

#### Acceptance Criteria

1. WHEN any service encounters an error, THEN the service SHALL log the error with appropriate context
2. WHEN the Collector service fails to deliver a webhook callback after all retries, THEN the Collector service SHALL log a critical error
3. WHEN the Project service receives an invalid webhook payload, THEN the Project service SHALL return a 400 Bad Request response
4. WHEN the WebSocket service loses connection to Redis, THEN the WebSocket service SHALL attempt to reconnect automatically
5. WHEN a dry-run job exceeds a reasonable timeout period, THEN the Project service SHALL mark the job as failed

### Requirement 7

**User Story:** As a user, I want to receive dry-run results in real-time via WebSocket, so that I can see preview content as soon as it's available.

#### Acceptance Criteria

1. WHEN a dry-run job is created, THEN the Project service SHALL return a job_id immediately with status "processing"
2. WHEN the Collector service completes crawling for a platform, THEN the results SHALL be delivered to the user's WebSocket connection
3. WHEN results arrive via WebSocket, THEN the message SHALL include the job_id, platform, and crawled posts
4. WHEN errors occur during crawling, THEN error information SHALL be delivered to the user's WebSocket connection
5. WHEN a user is not connected via WebSocket, THEN the results SHALL be lost (no persistence or caching)
