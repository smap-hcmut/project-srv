## MODIFIED Requirements

### Requirement: LLM Service Configuration

The system SHALL support configuration of LLM service through environment variables. The `LLM_API_KEY` is required for service startup. All other configuration MUST be optional with sensible defaults.

#### Scenario: Required Configuration - API Key

Given the system initializes LLM service
When `LLM_API_KEY` is not set or empty
Then the system fails to start
And logs an error "LLM API key is required"

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

#### Scenario: Invalid Provider Configuration

Given the system initializes LLM service
When `LLM_PROVIDER` is set to unsupported value (e.g., "invalid")
Then the system fails to start
And logs an error about unsupported provider

## ADDED Requirements

### Requirement: Runtime Error Handling

The system SHALL handle LLM API errors gracefully at runtime. When LLM calls fail, the system SHALL log warnings but SHALL NOT propagate errors to callers. Business logic MUST continue normally with original input data.

#### Scenario: Runtime API Error - 404

Given LLM service is initialized with valid config
When `CheckAmbiguity` is called
And Gemini API returns 404 (Not Found)
Then the provider logs a warning with error details
And returns `(false, "", error)` to caller
And caller logs warning and continues with original keyword

#### Scenario: Runtime API Error - Timeout

Given LLM service is initialized with valid config
When `CheckAmbiguity` is called
And Gemini API times out
Then the provider logs a warning with timeout details
And returns `(false, "", ErrLLMTimeout)` to caller
And caller logs warning and continues with original keyword

#### Scenario: Runtime API Error - Invalid Response

Given LLM service is initialized with valid config
When `CheckAmbiguity` is called
And Gemini API returns invalid JSON
Then the provider logs a warning with response details
And returns `(false, "", ErrLLMInvalidResponse)` to caller
And caller logs warning and continues with original keyword

#### Scenario: Runtime API Error - Network Failure

Given LLM service is initialized with valid config
When `CheckAmbiguity` is called
And network connection fails
Then the provider retries according to config
And if all retries fail, logs warning
And returns `(false, "", ErrLLMUnavailable)` to caller
And caller logs warning and continues with original keyword
