# LLM Service

## ADDED Requirements

### Requirement: LLM Provider Abstraction

The system SHALL provide an abstraction layer for LLM providers through `pkg/llm/Provider` interface. The system MUST support at least one LLM provider (Google Gemini) and MAY support additional providers (e.g., OpenAI) in the future. The abstraction MUST allow switching providers without changing business logic.

#### Scenario: Provider Interface Definition

Given the system has LLM service package
When examining the `pkg/llm/Provider` interface
Then it defines `SuggestKeywords(ctx, brandName) ([]string, []string, error)`
And it defines `CheckAmbiguity(ctx, keyword) (bool, string, error)`
And all methods accept `context.Context` for cancellation and timeout

#### Scenario: Multiple Provider Support

Given the system supports multiple LLM providers
When the system initializes
Then the system selects the provider based on configuration (`LLM_PROVIDER` environment variable)
And the selected provider implements the `Provider` interface
And business logic uses the interface, not specific provider implementation

### Requirement: Google Gemini Provider Implementation

The system SHALL implement Google Gemini as an LLM provider. The implementation MUST use Gemini REST API to generate keyword suggestions and check keyword ambiguity.

#### Scenario: Gemini Provider Initialization

Given `LLM_PROVIDER=gemini` is configured
And `LLM_API_KEY` contains a valid Gemini API key
And `LLM_MODEL=gemini-1.5-flash` (or other valid model)
When the system initializes LLM service
Then the Gemini provider is created with the API key and model
And HTTP client is configured with timeout from `LLM_TIMEOUT`

#### Scenario: Gemini Suggest Keywords

Given Gemini provider is initialized
When `SuggestKeywords` is called with brand name "VinFast"
Then the provider constructs a prompt for keyword suggestions
And sends HTTP POST request to Gemini API endpoint
And includes API key in Authorization header
And includes model name in request
And parses JSON response to extract niche and negative keywords
And returns the keywords as two separate arrays

#### Scenario: Gemini Check Ambiguity

Given Gemini provider is initialized
When `CheckAmbiguity` is called with keyword "Apple"
Then the provider constructs a prompt for ambiguity detection
And sends HTTP POST request to Gemini API endpoint
And includes API key in Authorization header
And parses JSON response to extract ambiguous flag and context
And returns `(true, "may refer to fruit or technology company", nil)` if ambiguous
And returns `(false, "", nil)` if not ambiguous

#### Scenario: Gemini API Error Handling

Given Gemini provider is initialized
When API call fails with 401 (Unauthorized)
Then the provider returns `ErrLLMInvalidAPIKey` error
And logs the error with context

#### Scenario: Gemini API Timeout

Given Gemini provider is initialized with timeout=30s
When API call does not complete within 30 seconds
Then the provider cancels the request
And returns `ErrLLMTimeout` error
And logs the timeout

#### Scenario: Gemini Invalid Response

Given Gemini provider is initialized
When API call succeeds but returns invalid JSON
Then the provider returns `ErrLLMInvalidResponse` error
And logs the response body for debugging

### Requirement: Retry Logic for LLM Calls

The system SHALL implement retry logic with exponential backoff for transient LLM API failures. The system SHALL NOT retry on client errors (4xx) or invalid responses.

#### Scenario: Retry on Network Error

Given Gemini provider is initialized with `MaxRetries=3`
When `SuggestKeywords` is called
And the first attempt fails with network error
Then the provider waits 1 second
And retries the request
And if second attempt fails, waits 2 seconds and retries
And if third attempt fails, waits 4 seconds and retries
And if all retries fail, returns the error

#### Scenario: No Retry on 4xx Errors

Given Gemini provider is initialized
When `SuggestKeywords` is called
And API returns 401 (Unauthorized)
Then the provider does NOT retry
And immediately returns `ErrLLMInvalidAPIKey` error

#### Scenario: Retry Configuration

Given `LLM_MAX_RETRIES=3` is configured
When the system initializes Gemini provider
Then the provider is configured with max 3 retries
And retry logic uses exponential backoff (1s, 2s, 4s)

### Requirement: LLM Service Configuration

The system SHALL support configuration of LLM service through environment variables. All configuration MUST be optional with sensible defaults except for API key which is required for LLM functionality.

#### Scenario: Required Configuration

Given the system initializes LLM service
When `LLM_API_KEY` is not set
Then the system logs a warning
And LLM service is not available
And business logic falls back to non-LLM behavior

#### Scenario: Optional Configuration with Defaults

Given the system initializes LLM service
When `LLM_PROVIDER` is not set
Then the system uses default "gemini"
When `LLM_MODEL` is not set
Then the system uses default "gemini-1.5-flash"
When `LLM_TIMEOUT` is not set
Then the system uses default 30 seconds
When `LLM_MAX_RETRIES` is not set
Then the system uses default 3 retries

#### Scenario: Invalid Configuration

Given the system initializes LLM service
When `LLM_PROVIDER` is set to unsupported value (e.g., "invalid")
Then the system logs an error
And LLM service is not available
And business logic falls back to non-LLM behavior

### Requirement: LLM Service Error Types

The system SHALL define structured error types for LLM service failures. All errors MUST be exported and usable by calling code for error handling and logging.

#### Scenario: Error Type Definitions

Given the LLM service package
When examining error definitions
Then `ErrLLMUnavailable` is defined for service unavailable
And `ErrLLMTimeout` is defined for timeout errors
And `ErrLLMInvalidResponse` is defined for invalid responses
And `ErrLLMInvalidAPIKey` is defined for authentication errors
And all errors are exported and can be checked with `errors.Is()`

#### Scenario: Error Context

Given LLM service returns an error
When the error is logged or returned
Then the error includes context about the operation (suggest vs ambiguity check)
And the error includes relevant details (brand name, keyword, etc.)
And the error can be wrapped with additional context

### Requirement: HTTP Client Configuration

The system SHALL configure HTTP client for LLM API calls with appropriate timeout, retry, and connection pooling settings.

#### Scenario: HTTP Client Timeout

Given LLM service is initialized with `LLM_TIMEOUT=30`
When HTTP client is created
Then the client has request timeout of 30 seconds
And requests are cancelled if they exceed the timeout

#### Scenario: HTTP Client Connection Pooling

Given LLM service is initialized
When HTTP client is created
Then the client uses connection pooling
And idle connections are reused
And connection pool size is configured appropriately

#### Scenario: HTTP Client Headers

Given Gemini provider makes API calls
When HTTP request is sent
Then the request includes `Content-Type: application/json` header
And the request includes `Authorization: Bearer {API_KEY}` header
And the request includes appropriate `User-Agent` header
