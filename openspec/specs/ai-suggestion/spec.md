# ai-suggestion Specification

## Purpose
TBD - created by archiving change enforce-smart-keywords. Update Purpose after archive.
## Requirements
### Requirement: Suggest Niche Keywords

The system SHALL suggest niche keywords based on a seed keyword using an LLM provider. The system MUST use a real LLM service (e.g., Google Gemini) to generate intelligent, context-aware suggestions. If the LLM service is unavailable, the system SHALL fallback to basic suggestions to ensure the feature remains functional.

#### Scenario: Successful LLM Suggestion

Given a seed keyword "VinFast"
When the user requests suggestions via `POST /projects/keywords/suggest`
Then the system calls the LLM provider with the brand name
And the LLM provider returns intelligent suggestions
And the system returns a list including "VinFast VF3", "VF Wild", "trạm sạc VinFast"
And all suggestions are validated using the keyword validation rules

#### Scenario: LLM Service Unavailable - Fallback

Given a seed keyword "VinFast"
When the user requests suggestions
And the LLM service is unavailable (timeout, network error, or service error)
Then the system logs a warning with the error details
And the system falls back to basic suggestions (e.g., "{brandName} review", "{brandName} price")
And the API returns the fallback suggestions successfully
And no error is returned to the user

#### Scenario: LLM Invalid Response - Fallback

Given a seed keyword "VinFast"
When the user requests suggestions
And the LLM service returns an invalid or malformed response
Then the system logs an error with the response details
And the system falls back to basic suggestions
And the API returns the fallback suggestions successfully

#### Scenario: Post-Validation of Suggestions

Given LLM returns suggestions including invalid keywords
When the system processes the suggestions
Then invalid keywords are filtered out using the keyword validation rules
And only valid keywords are returned to the user
And the system logs which keywords were filtered

### Requirement: Suggest Negative Keywords

The system SHALL suggest negative keywords to exclude irrelevant results using an LLM provider. The system MUST use a real LLM service to generate context-aware negative keyword suggestions. If the LLM service is unavailable, the system SHALL fallback to basic negative keyword suggestions.

#### Scenario: Successful Negative Keyword Suggestion

Given a seed keyword "VinFast"
When the user requests suggestions
Then the system calls the LLM provider
And the LLM provider returns negative keyword suggestions
And the system returns a list of negative keywords including "sim", "xổ số", "tuyển dụng"
And all negative keywords are validated using the keyword validation rules

#### Scenario: Negative Keywords Fallback

Given a seed keyword "VinFast"
When the user requests suggestions
And the LLM service is unavailable
Then the system falls back to basic negative keywords (e.g., "job", "hiring", "second hand", "used")
And the API returns the fallback negative keywords successfully

### Requirement: LLM Integration Configuration

The system SHALL support configuration of LLM provider through environment variables. The system MUST support at least one LLM provider (Google Gemini) and MAY support additional providers in the future.

#### Scenario: Gemini Provider Configuration

Given the system is configured with `LLM_PROVIDER=gemini`
And `LLM_API_KEY` is set to a valid Gemini API key
And `LLM_MODEL=gemini-1.5-flash`
When the system initializes
Then the LLM service uses Gemini provider
And suggestions are generated using Gemini API

#### Scenario: Invalid API Key

Given `LLM_API_KEY` is invalid or missing
When the system attempts to use LLM service
Then the system logs an error
And falls back to basic suggestions for all requests
And the system continues to function without LLM

#### Scenario: LLM Timeout Configuration

Given `LLM_TIMEOUT=30` (seconds)
When the system calls LLM service
And the LLM service does not respond within 30 seconds
Then the system cancels the request
And logs a timeout error
And falls back to basic suggestions

### Requirement: Retry Logic for LLM Calls

The system SHALL retry failed LLM API calls with exponential backoff for transient failures (network errors, timeouts). The system SHALL NOT retry on client errors (4xx) or invalid responses.

#### Scenario: Retry on Network Error

Given the system calls LLM service
And the first attempt fails with a network error
Then the system waits 1 second
And retries the request
And if the second attempt succeeds, returns the result
And if the second attempt fails, waits 2 seconds and retries once more

#### Scenario: No Retry on Client Error

Given the system calls LLM service
And the LLM service returns 401 (Unauthorized - invalid API key)
Then the system does NOT retry
And immediately falls back to basic suggestions
And logs the error

#### Scenario: Max Retries Reached

Given `LLM_MAX_RETRIES=3`
When the system calls LLM service
And all 3 retry attempts fail
Then the system stops retrying
And falls back to basic suggestions
And logs that max retries were reached

