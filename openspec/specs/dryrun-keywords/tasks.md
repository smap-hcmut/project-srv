# Implementation Plan

- [x] 1. Extend Project service types and interfaces




- [ ] 1.1 Add dry-run types to project/type.go
  - Add DryRunKeywordsInput struct with Keywords field (validation: required, min=1, max=10)
  - Add DryRunKeywordsOutput struct with JobID and Status fields


  - _Requirements: 1.1, 1.5_

- [ ] 1.2 Implement DryRunKeywords in project/usecase/project.go
  - Validate keywords using existing keywordUC.Validate
  - Generate UUID for job_id using uuid.New()
  - Build DryRunCrawlRequest message
  - Call producer.PublishDryRunTask to send to RabbitMQ
  - Return DryRunKeywordsOutput with job_id and "processing" status
  - Handle errors with appropriate logging
  - _Requirements: 1.1, 1.4, 1.5_

- [x]* 1.3 Write property tests for dry-run usecase



  - **Property 1: Valid requests return job_id and processing status**
  - **Property 4: Invalid keywords are rejected without job creation**
  - **Property 5: Job IDs are valid UUIDs**
  - **Validates: Requirements 1.1, 1.4, 1.5**


- [ ] 2. Implement RabbitMQ delivery layer for dry-run tasks
- [ ] 2.1 Add constants to project/delivery/rabbitmq/constants.go
  - Define CollectorInboundExchangeName = "collector.inbound"
  - Define DryRunKeywordRoutingKey = "crawler.dryrun_keyword"
  - Define CollectorInboundExchange = ExchangeArgs{Name, Type: topic, Durable: true}

  - _Requirements: 1.2_

- [ ] 2.2 Add DryRunCrawlRequest presenter to project/delivery/rabbitmq/presenters.go
  - Define DryRunCrawlRequest struct matching collector's CrawlRequest format

  - Include JobID, TaskType, Payload, TimeRange, Attempt, MaxAttempts, EmittedAt fields
  - Define DryRunPayload struct with Keywords, LimitPerKeyword, IncludeComments, MaxComments
  - _Requirements: 1.2, 1.3_

- [ ] 2.3 Update Producer interface in project/delivery/rabbitmq/producer/new.go
  - Add PublishDryRunTask(ctx context.Context, msg rabbitmq.DryRunCrawlRequest) error to Producer interface
  - Add dryRunWriter *rabbitmq.Channel to implProducer struct

  - _Requirements: 1.2_

- [ ] 2.4 Implement PublishDryRunTask in project/delivery/rabbitmq/producer/producer.go
  - Marshal DryRunCrawlRequest to JSON
  - Publish to CollectorInboundExchange with routing key DryRunKeywordRoutingKey
  - Set ContentType to "application/json"
  - Log publish success/failure
  - Return error if marshal or publish fails
  - _Requirements: 1.2, 1.3_




- [ ] 2.5 Initialize dryRunWriter in project/delivery/rabbitmq/producer/common.go
  - Add dryRunWriter initialization in Run() method using getWriter(CollectorInboundExchange)
  - Add dryRunWriter.Close() in Close() method
  - Handle initialization errors appropriately

  - _Requirements: 1.2_

- [ ]* 2.6 Write property tests for RabbitMQ publishing
  - **Property 2: Dry-run requests publish to correct RabbitMQ destination**
  - **Property 3: CrawlRequest contains required dry-run parameters**
  - **Property 22: RabbitMQ messages follow CrawlRequest format**




  - **Validates: Requirements 1.2, 1.3, 5.1**

- [ ] 3. Wire RabbitMQ producer into project usecase
- [ ] 3.1 Update project/usecase/new.go
  - Add Producer parameter to New() function
  - Store producer in usecase struct
  - _Requirements: 1.2_

- [x] 3.2 Update httpserver/handler.go

  - Initialize RabbitMQ connection if not exists
  - Initialize RabbitMQ producer using producer.New()
  - Call producer.Run() to initialize channels
  - Pass producer to projectusecase.New()
  - Handle initialization errors
  - _Requirements: 1.2_

- [ ] 4. Add HTTP endpoint for dry-run in Project service
- [ ] 4.1 Add DryRunKeywords handler to project/delivery/http/handler.go
  - Implement DryRunKeywords(c *gin.Context) method
  - Bind DryRunKeywordsInput from request body
  - Extract scope from context using scope.Get(c)
  - Call h.uc.DryRunKeywords with scope and keywords





  - Return DryRunKeywordsOutput with 200 status using response.Success
  - Handle validation errors with 400 status using response.Error
  - Handle internal errors with 500 status
  - _Requirements: 1.1_


- [ ] 4.2 Register route in project/delivery/http/routes.go
  - Add POST /keywords/dry-run route to MapProjectRoutes


  - Apply authentication middleware (mw.Auth())
  - Map to handler.DryRunKeywords
  - _Requirements: 1.1_

- [ ]* 4.3 Write unit tests for HTTP handler
  - Test valid keywords return 200 with job_id
  - Test invalid keywords return 400


  - Test empty keywords return 400
  - Test authentication required (401 without token)
  - _Requirements: 1.1, 1.4_


- [ ] 5. Checkpoint - Ensure all Project service tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 6. Create webhook package for collector callbacks
- [ ] 6.1 Create internal/webhook/type.go
  - Define CallbackRequest struct with JobID, Status, Platform, Payload, UserID
  - Define CallbackPayload struct with Posts and Errors arrays
  - Define Post struct with ID, Title, URL, Author, PublishedAt, Comments
  - Define Comment struct with ID, Text, Author, CreatedAt
  - Define Error struct with Code, Message, Keyword
  - Add JSON tags and validation tags (binding:"required", etc.)
  - _Requirements: 3.1_





- [ ] 6.2 Create internal/webhook/handler.go
  - Create Handler struct with logger and Redis client
  - Implement New(l log.Logger, redisClient *redis.Client) *Handler constructor
  - Implement DryRunCallback(c *gin.Context) method
  - Validate X-Internal-Key header matches config.InternalKey
  - Bind CallbackRequest from request body

  - Extract user_id from CallbackRequest
  - Publish result to Redis channel `user_noti:{user_id}`
  - Return 200 on success, 401 on auth failure, 400 on invalid payload
  - Log all operations with context





  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_



- [ ] 6.3 Add publishDryRunResult method to webhook/handler.go
  - Format Redis channel as `user_noti:{user_id}`
  - Construct message with type="dryrun_result", job_id, platform, status, payload
  - Marshal message to JSON
  - Publish to Redis using client.Publish(ctx, channel, message)
  - Log publish success/failure
  - Return error if marshal or publish fails
  - _Requirements: 3.3, 3.4, 3.5_

- [ ]* 6.4 Write property tests for webhook handler
  - **Property 13: Webhook endpoint validates authentication**


  - **Property 27: Invalid webhook payloads return 400**
  - **Property 16: Redis channel naming convention**
  - **Property 24: Redis messages follow standard format**
  - **Validates: Requirements 3.1, 3.5, 5.3, 6.3**

- [ ] 7. Register webhook route and initialize Redis
- [-] 7.1 Initialize Redis client in httpserver/server.go



  - Add Redis client to HTTPServer struct if not exists
  - Initialize Redis client in New() using redis.NewClient(cfg.Redis)
  - Test connection with Ping()
  - Handle initialization errors


  - _Requirements: 3.5_

- [ ] 7.2 Register webhook route in httpserver/handler.go
  - Initialize webhook handler using webhook.New(srv.l, srv.redisClient)
  - Create internal routes group if not exists
  - Register POST /internal/collector/dryrun/callback
  - Apply internal authentication middleware (mw.InternalAuth())
  - Map to webhookHandler.DryRunCallback
  - _Requirements: 3.1_

- [ ] 8. Checkpoint - Ensure all Project service tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 9. Implement Collector service task type support
- [ ] 9.1 Add dryrun_keyword task type to collector/internal/models/task.go
  - Add TaskTypeDryRunKeyword TaskType = "dryrun_keyword" constant
  - _Requirements: 2.1_

- [ ] 9.2 Add payload mapping for dry-run in collector/internal/dispatcher/usecase/util.go
  - Add mapDryRunPayload function
  - Extract keywords, limit_per_keyword, include_comments, max_comments from payload
  - For YouTube: return YouTubeResearchAndCrawlPayload with Keywords, LimitPerKeyword=3, IncludeComments=true, MaxComments=5
  - For TikTok: return TikTokResearchAndCrawlPayload with Keywords, LimitPerKeyword=3, IncludeComments=true, MaxComments=5
  - Handle missing or invalid payload fields
  - _Requirements: 1.3, 2.2, 2.3_

- [ ]* 9.3 Write property tests for payload mapping
  - **Property 7: Post count constraint per keyword**
  - **Property 8: Comment count constraint per post**
  - **Validates: Requirements 2.2, 2.3**

- [-] 9.4 Update dispatcher to handle dryrun_keyword tasks


  - Update `collector/internal/dispatcher/usecase/dispatch_uc.go`
  - Add case for TaskTypeDryRunKeyword in Dispatch method
  - Call mapDryRunPayload for payload transformation
  - Publish to platform-specific queues (YouTube and TikTok)
  - _Requirements: 2.1_



- [ ]* 9.5 Write property test for task recognition
  - **Property 6: Collector recognizes dryrun_keyword task type**
  - **Validates: Requirements 2.1**

- [ ] 10. Implement Collector webhook client
- [ ] 10.1 Create webhook client in collector/internal/webhook/client.go
  - Define WebhookClient interface with SendCallback method
  - Create httpClient struct with url, internalKey, httpClient, logger, retryConfig
  - Implement NewClient(url, internalKey string, logger log.Logger) WebhookClient constructor
  - _Requirements: 2.4, 2.5, 2.7_

- [x] 10.2 Implement SendCallback with retry logic





  - Marshal CallbackRequest to JSON
  - Create HTTP POST request to webhook URL
  - Add X-Internal-Key header for authentication


  - Set Content-Type to application/json
  - Set timeout to 10 seconds
  - Implement retry loop with max 5 attempts
  - Start with 1 second delay, double each retry (exponential backoff)
  - Cap max delay at 32 seconds
  - Log each retry attempt with attempt number and delay
  - Return error after all retries exhausted
  - _Requirements: 2.4, 2.5, 2.6, 2.7_

- [ ]* 10.3 Write property tests for webhook client
  - **Property 11: Webhook retries use exponential backoff**
  - **Property 12: Webhook requests include authentication**
  - **Property 23: Webhook payloads follow standard format**
  - **Validates: Requirements 2.6, 2.7, 5.2**

- [ ] 10.4 Integrate webhook client into dispatcher
  - Update `collector/internal/dispatcher/usecase/usecase.go`
  - Add WebhookClient to implUseCase struct
  - Update NewUseCase to accept and store WebhookClient
  - After crawl completion, construct CallbackRequest with job_id, status="success", platform, posts
  - After crawl error, construct CallbackRequest with job_id, status="failed", platform, errors
  - Call webhookClient.SendCallback
  - Log callback success/failure
  - _Requirements: 2.4, 2.5_

- [ ]* 10.5 Write property tests for callback integration
  - **Property 9: Successful crawls trigger webhook callbacks**
  - **Property 10: Failed crawls trigger error callbacks**
  - **Validates: Requirements 2.4, 2.5**

- [ ] 11. Add webhook configuration to Collector
- [ ] 11.1 Update collector/config/config.go
  - Add WebhookConfig struct with URL, InternalKey, RetryAttempts, RetryDelay fields
  - Add environment variable bindings (env tags)
  - Add WebhookConfig to main Config struct
  - _Requirements: 2.6, 2.7_

- [ ] 11.2 Update collector/template.env
  - Add PROJECT_WEBHOOK_URL=http://project-service:8080/internal/collector/dryrun/callback
  - Add WEBHOOK_INTERNAL_KEY=<secret>
  - Add WEBHOOK_RETRY_ATTEMPTS=5
  - Add WEBHOOK_RETRY_DELAY=1
  - _Requirements: 2.6, 2.7_

- [ ] 11.3 Initialize webhook client in collector/internal/consumer/server.go
  - Create webhook client using webhook.NewClient(cfg.Webhook.URL, cfg.Webhook.InternalKey, srv.l)
  - Pass webhook client to dispatcher usecase
  - _Requirements: 2.4_

- [ ] 12. Checkpoint - Ensure all Collector service tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 13. Update WebSocket service for dry-run messages
- [ ] 13.1 Add dry-run message type to websocket/internal/websocket/types.go
  - Add MessageTypeDryRunResult constant = "dryrun_result"
  - Add DryRunResultPayload struct with JobID, Platform, Status, Posts, Errors if needed
  - _Requirements: 4.2_

- [ ] 13.2 Update Redis subscriber to handle dry-run messages
  - Update `websocket/internal/redis/subscriber.go`
  - In handleMessage, add parsing for type="dryrun_result"
  - Extract job_id, platform, status, payload from message
  - Create WebSocket message with parsed data
  - Forward to Hub.SendToUser with user_id extracted from channel name
  - Log message reception and delivery
  - _Requirements: 4.2, 4.3, 4.4, 4.5_

- [ ]* 13.3 Write property tests for WebSocket message handling
  - **Property 17: WebSocket parses valid messages**
  - **Property 20: Platform information is preserved**
  - **Property 21: Malformed messages don't crash WebSocket service**
  - **Validates: Requirements 4.2, 4.5, 4.6**

- [ ]* 13.4 Write property tests for message routing
  - **Property 18: WebSocket identifies user connections**
  - **Property 19: Messages fan out to all user connections**
  - **Validates: Requirements 4.3, 4.4**

- [ ] 14. Checkpoint - Ensure all WebSocket service tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 15. Add configuration and logging
- [ ] 15.1 Update Project service configuration
  - Add INTERNAL_API_KEY to project/config/config.go if not exists
  - Add Redis configuration fields if not exists
  - Update project/template.env with new variables
  - _Requirements: 3.1, 3.5_

- [ ] 15.2 Add structured logging to Project service
  - Log dry-run request with job_id, user_id, and keywords count
  - Log RabbitMQ publish success/failure with job_id
  - Log webhook callback reception with job_id, platform, status
  - Log Redis publish success/failure with channel and job_id
  - Use ctx for all log calls
  - _Requirements: 6.1_

- [ ] 15.3 Add structured logging to Collector service
  - Log dryrun_keyword task consumption with job_id
  - Log crawl start with job_id and platform
  - Log crawl completion with job_id, platform, and result count
  - Log webhook callback attempts with job_id, platform, attempt number
  - Log webhook success/failure with job_id and platform
  - _Requirements: 6.1, 6.2_

- [ ] 15.4 Add structured logging to WebSocket service
  - Log dry-run message reception from Redis with job_id and platform
  - Log message parsing success/failure
  - Log message delivery to connections with user_id and connection count
  - _Requirements: 6.1_

- [ ]* 15.5 Write property test for error logging
  - **Property 26: Errors are logged with context**
  - **Validates: Requirements 6.1**

- [ ] 16. Integration testing and documentation
- [ ]* 16.1 Write end-to-end integration test
  - Test complete flow: API → RabbitMQ → Collector → Webhook → Redis → WebSocket
  - Use test RabbitMQ and Redis instances
  - Mock platform crawlers to return test data
  - Verify message delivery to WebSocket client
  - Test error scenarios (invalid keywords, crawl failures)
  - _Requirements: All_

- [ ]* 16.2 Update API documentation
  - Add Swagger annotations for POST /projects/keywords/dry-run
  - Add Swagger annotations for webhook endpoint (internal)
  - Document request/response formats with examples
  - Document error codes and messages
  - Run `make swagger` to regenerate docs
  - _Requirements: 1.1, 3.1_

- [ ]* 16.3 Update README documentation
  - Document the dry-run feature in project README
  - Add example curl requests for dry-run endpoint
  - Document WebSocket message format for frontend clients
  - Add architecture diagram showing message flow
  - _Requirements: All_

- [ ] 17. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.
