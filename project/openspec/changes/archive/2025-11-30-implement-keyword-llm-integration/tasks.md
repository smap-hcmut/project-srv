# Implementation Tasks

## Phase 1: Infrastructure Setup

### 1.1 LLM Service Package
- [x] 1.1.1 Create `pkg/llm/` directory structure
- [x] 1.1.2 Create `pkg/llm/interface.go` with `Provider` interface
  - [x] Define `SuggestKeywords(ctx, brandName) ([]string, []string, error)`
  - [x] Define `CheckAmbiguity(ctx, keyword) (bool, string, error)`
- [x] 1.1.3 Create `pkg/llm/errors.go` with error types
  - [x] `ErrLLMUnavailable`
  - [x] `ErrLLMTimeout`
  - [x] `ErrLLMInvalidResponse`
  - [x] `ErrLLMInvalidAPIKey`
- [x] 1.1.4 Create `pkg/llm/gemini.go` with Gemini implementation
  - [x] Implement HTTP client for Gemini API
  - [x] Implement `SuggestKeywords` with proper prompt
  - [x] Implement `CheckAmbiguity` with proper prompt
  - [x] Add retry logic with exponential backoff
  - [x] Add timeout handling
  - [x] Parse JSON responses safely
- [x] 1.1.5 Create `pkg/llm/new.go` with constructor
  - [x] Accept logger and config
  - [x] Initialize HTTP client with timeout
  - [x] Return appropriate provider based on config
- [x] 1.1.6 Write unit tests for Gemini provider
  - [x] Test successful suggestions
  - [x] Test successful ambiguity check
  - [x] Test error handling (timeout, invalid response, etc.)
  - [x] Test retry logic

### 1.2 Collector Service Package
- [x] 1.2.1 Create `pkg/collector/` directory structure
- [x] 1.2.2 Create `pkg/collector/types.go` with data structures
  - [x] `Post` struct with fields:
    - [x] `ID` (string): Unique post identifier
    - [x] `Content` (string): Post text content
    - [x] `Source` (string): Platform name (facebook, twitter, reddit, etc.)
    - [x] `SourceID` (string): Original post ID on platform
    - [x] `Author` (string): Author name/username
    - [x] `AuthorID` (string): Author identifier on platform
    - [x] `Date` (time.Time): Post creation date
    - [x] `URL` (string): Direct link to original post
    - [x] `Engagement` (Engagement struct): Engagement metrics
    - [x] `Metadata` (Metadata struct): Additional metadata
  - [x] `Engagement` struct (likes, comments, shares)
  - [x] `Metadata` struct (language, sentiment)
  - [x] `DryRunRequest` struct (keywords, limit)
  - [x] `DryRunResponse` struct (posts array, total_found, limit)
- [x] 1.2.3 Create `pkg/collector/client.go` with `Client` interface
  - [x] Define `DryRun(ctx, keywords, limit) ([]Post, error)`
- [x] 1.2.4 Create `pkg/collector/errors.go` with error types
  - [x] `ErrCollectorUnavailable`
  - [x] `ErrCollectorTimeout`
  - [x] `ErrCollectorInvalidResponse`
- [x] 1.2.5 Create `pkg/collector/http_client.go` with HTTP implementation
  - [x] Implement HTTP client with timeout
  - [x] Implement `DryRun` method
  - [x] Construct request to `{baseURL}/api/v1/collector/dry-run`
  - [x] Parse JSON response safely with all Post fields
  - [x] Handle HTTP errors (4xx, 5xx)
  - [x] Support default mock URL `http://localhost:8081` when URL not configured
- [x] 1.2.6 Create `pkg/collector/new.go` with constructor
  - [x] Accept logger and config
  - [x] Initialize HTTP client with timeout
  - [x] Return HTTP client implementation
- [x] 1.2.7 Write unit tests for Collector client
  - [x] Test successful dry run
  - [x] Test error handling (timeout, 4xx, 5xx)
  - [x] Test response parsing

### 1.3 Configuration Updates
- [x] 1.3.1 Update `config/config.go`
  - [x] Add `LLMConfig` struct with fields:
    - [x] `Provider` (string, default: "gemini")
    - [x] `APIKey` (string, required)
    - [x] `Model` (string, default: "gemini-1.5-flash")
    - [x] `Timeout` (int, default: 30)
    - [x] `MaxRetries` (int, default: 3)
  - [x] Add `CollectorConfig` struct with fields:
    - [x] `BaseURL` (string, default: "http://localhost:8081" for development)
    - [x] `Timeout` (int, default: 30)
  - [x] Add `LLM` field to `Config` struct
  - [x] Add `Collector` field to `Config` struct
- [x] 1.3.2 Update `template.env` with new environment variables
  - [x] `LLM_PROVIDER=gemini`
  - [x] `LLM_API_KEY=`
  - [x] `LLM_MODEL=gemini-1.5-flash`
  - [x] `LLM_TIMEOUT=30`
  - [x] `LLM_MAX_RETRIES=3`
  - [x] `COLLECTOR_SERVICE_URL=http://localhost:8081` (default mock URL for development)
  - [x] `COLLECTOR_TIMEOUT=30`
  - [x] Add comment: "For production, set COLLECTOR_SERVICE_URL to actual Collector Service URL"
- [x] 1.3.3 Update `k8s/secret.yaml.template` with new secret keys (if applicable)

## Phase 2: Keyword Usecase Integration

### 2.1 Update Keyword Usecase Structure
- [x] 2.1.1 Update `internal/keyword/usecase/new.go`
  - [x] Add `llmProvider` field to `usecase` struct
  - [x] Update constructor to accept LLM provider and Collector client
  - [x] Store dependencies in struct
- [x] 2.1.2 Update `internal/keyword/usecase/keyword.go` (if needed for interface changes)

### 2.2 Implement Real AI Suggestion
- [x] 2.2.1 Update `internal/keyword/usecase/suggester.go`
  - [x] Replace hardcoded logic in `suggestProcessing` with LLM call
  - [x] Implement fallback to basic suggestions if LLM fails
  - [x] Post-validate suggestions using existing `validate` method
  - [x] Add proper error handling and logging
  - [x] Ensure suggestions are deduplicated
- [x] 2.2.2 Add `fallbackSuggestions` helper method
  - [x] Generate basic suggestions (current hardcoded logic)
  - [x] Return niche and negative keywords
- [x] 2.2.3 Write unit tests for suggester
  - [x] Test successful LLM suggestion
  - [x] Test fallback when LLM fails
  - [x] Test post-validation of suggestions
  - [x] Test with mock LLM provider

### 2.3 Enhance Semantic Validation
- [x] 2.3.1 Update `internal/keyword/usecase/validator.go`
  - [x] Add `isSingleWord` helper function
  - [x] Update `validateOne` to call LLM for ambiguity check on single words
  - [x] Implement warning system (log warning, don't reject)
  - [x] Handle LLM errors gracefully (continue without LLM check)
  - [x] Add context to ambiguity warnings
- [x] 2.3.2 Write unit tests for validator
  - [x] Test ambiguity detection for single words
  - [x] Test that multi-word keywords skip LLM check
  - [x] Test graceful degradation when LLM fails
  - [x] Test with mock LLM provider

### 2.4 Implement Real Dry Run
- [x] 2.4.1 Update `internal/keyword/usecase/tester.go`
  - [x] Replace mock data with Collector Service call
  - [x] Validate keywords before calling Collector
  - [x] Call Collector Service with validated keywords and limit (10)
  - [x] Convert Collector `Post` structs to `interface{}` for response
  - [x] Add proper error handling and logging
- [x] 2.4.2 Write unit tests for tester
  - [x] Test successful dry run with Collector
  - [x] Test error handling when Collector fails
  - [x] Test keyword validation before dry run
  - [x] Test with mock Collector client

## Phase 3: Dependency Injection & Initialization

### 3.1 Update HTTP Server Handler
- [x] 3.1.1 Update `internal/httpserver/handler.go`
  - [x] Initialize LLM provider using config
  - [x] Initialize Collector client using config
  - [x] Pass LLM provider to keyword usecase constructor
  - [x] Pass Collector client to keyword usecase constructor
  - [x] Handle initialization errors gracefully
- [x] 3.1.2 Update `internal/httpserver/new.go` (if needed)
  - [x] Add LLM config to `Config` struct
  - [x] Add Collector config to `Config` struct
  - [x] Validate new configs in `validate()` method

### 3.2 Update Main Application
- [x] 3.2.1 Update `cmd/api/main.go`
  - [x] Load LLM config from environment
  - [x] Load Collector config from environment
  - [x] Pass configs to HTTP server initialization
  - [x] Handle missing required configs (log error, fail fast)

## Phase 4: Error Handling & Resilience

### 4.1 Error Type Definitions
- [x] 4.1.1 Ensure all error types are properly defined
  - [x] LLM errors in `pkg/llm/errors.go`
  - [x] Collector errors in `pkg/collector/errors.go`
  - [ ] Keyword errors in `internal/keyword/error.go` (if needed)

### 4.2 Error Mapping
- [ ] 4.2.1 Update `internal/project/delivery/http/error.go` (if exists)
  - [ ] Map LLM errors to HTTP errors
  - [ ] Map Collector errors to HTTP errors
  - [ ] Ensure user-friendly error messages

### 4.3 Logging
- [x] 4.3.1 Add structured logging for all external service calls
  - [x] Log LLM API calls (request, response, latency)
  - [x] Log Collector API calls (request, response, latency)
  - [x] Log errors with full context
  - [x] Log fallback activations

## Phase 5: Testing

### 5.1 Unit Tests
- [x] 5.1.1 Test LLM provider with mock HTTP client
- [x] 5.1.2 Test Collector client with mock HTTP client
- [x] 5.1.3 Test keyword usecase with mock LLM and Collector
- [x] 5.1.4 Test error handling paths
- [x] 5.1.5 Test fallback mechanisms

### 5.2 Integration Tests (Optional)
- [ ] 5.2.1 Test with real Gemini API (staging environment)
- [ ] 5.2.2 Test with real Collector Service (staging environment)
- [ ] 5.2.3 Test full flow from API endpoint to response

### 5.3 Manual Testing
- [ ] 5.3.1 Test suggestion endpoint with various brand names
- [ ] 5.3.2 Test validation with ambiguous keywords
- [ ] 5.3.3 Test dry run with various keyword combinations
- [ ] 5.3.4 Test error scenarios (invalid API key, service down, etc.)

## Phase 6: Documentation & Cleanup

### 6.1 Code Documentation
- [x] 6.1.1 Add Go doc comments to all exported functions
- [x] 6.1.2 Add inline comments for complex logic
- [ ] 6.1.3 Document LLM prompt design decisions

### 6.2 Configuration Documentation
- [x] 6.2.1 Update `README.md` with new environment variables
- [ ] 6.2.2 Document LLM provider setup instructions
- [ ] 6.2.3 Document Collector Service URL configuration

### 6.3 API Documentation
- [ ] 6.3.1 Update Swagger docs if API contracts changed (shouldn't)
- [ ] 6.3.2 Verify API examples in documentation

### 6.4 Code Review Checklist
- [x] 6.4.1 All tests passing
- [x] 6.4.2 No linter errors
- [x] 6.4.3 Error handling comprehensive
- [x] 6.4.4 Logging adequate
- [x] 6.4.5 Code follows project conventions
- [x] 6.4.6 No hardcoded values (use config)
- [x] 6.4.7 All TODOs addressed or documented

## Validation Checklist

Before considering this change complete:
- [x] All tasks in Phase 1-6 are checked off
- [x] Unit tests have >80% coverage for new code
- [ ] Integration tests pass (if implemented)
- [ ] Manual testing completed successfully
- [x] Code review approved
- [x] Documentation updated
- [x] Configuration templates updated
- [x] No breaking changes to existing APIs
- [x] Error handling tested for all failure scenarios
- [x] Fallback mechanisms verified