# Implement Keyword LLM Integration

## Why

Mặc dù proposal "enforce-smart-keywords" đã được archive và module structure đã được setup, nhưng implementation hiện tại chỉ là **placeholder/mock logic**. Các usecase trong `internal/keyword/usecase/` chưa có implementation thực sự:

1. **AI Suggestion** (`suggestProcessing`): Hiện tại chỉ hardcode suggestions (append "review", "price", etc.) thay vì gọi LLM service
2. **Semantic Validation** (`validateOne`): Chỉ có stopwords check, chưa có LLM-based ambiguity detection
3. **Dry Run** (`test`): Trả về mock data thay vì gọi Collector Service để fetch real data

Điều này khiến system không thể deliver giá trị thực sự cho users - suggestions không intelligent, validation không detect ambiguity, và dry run không show real results.

## What Changes

### Core Infrastructure
- **ADDED**: `pkg/llm/` package với interface và implementations (Gemini, OpenAI)
- **ADDED**: `pkg/collector/` package với HTTP client để gọi Collector Service
- **ADDED**: LLM configuration trong `config/config.go`
- **ADDED**: Collector Service configuration trong `config/config.go`

### Implementation Updates
- **MODIFIED**: `internal/keyword/usecase/suggester.go` - Implement real LLM integration với fallback
- **MODIFIED**: `internal/keyword/usecase/validator.go` - Add LLM-based ambiguity detection
- **MODIFIED**: `internal/keyword/usecase/tester.go` - Integrate với Collector Service
- **MODIFIED**: `internal/keyword/usecase/new.go` - Inject LLM provider và Collector client
- **MODIFIED**: `internal/httpserver/handler.go` - Initialize LLM và Collector services

### Error Handling & Resilience
- **ADDED**: Graceful degradation khi LLM unavailable (fallback to basic suggestions)
- **ADDED**: Proper error types cho LLM và Collector failures
- **ADDED**: Retry logic cho LLM API calls
- **ADDED**: Timeout handling cho external service calls

### Configuration
- **ADDED**: Environment variables cho LLM provider (provider, API key, model, timeout)
- **ADDED**: Environment variables cho Collector Service (base URL, timeout)

## Impact

### Affected Specs
- `specs/ai-suggestion/spec.md` - **MODIFIED**: Add detailed LLM integration requirements, fallback behavior, error handling
- `specs/keyword-validation/spec.md` - **MODIFIED**: Add LLM ambiguity detection requirements, warning system
- `specs/dry-run/spec.md` - **MODIFIED**: Add Collector Service integration requirements, error handling
- `specs/llm-service/spec.md` - **ADDED**: New capability for LLM service abstraction
- `specs/collector-service/spec.md` - **ADDED**: New capability for Collector Service client

### Affected Code
- `pkg/llm/` - New package (interface.go, gemini.go, openai.go, errors.go, new.go)
- `pkg/collector/` - New package (client.go, types.go, errors.go, new.go)
- `config/config.go` - Add LLMConfig và CollectorConfig
- `internal/keyword/usecase/` - All files modified để use real services
- `internal/httpserver/handler.go` - Initialize new services
- `cmd/api/main.go` - Load new configs

### External Dependencies
- **LLM Provider**: Gemini API hoặc OpenAI API (new external dependency)
- **Collector Service**: HTTP API endpoint cho dry run (existing service, new integration)

### Breaking Changes
- **NONE**: All changes are internal implementation details. API contracts remain unchanged.

### Performance Impact
- **LLM API Calls**: 2-5 seconds per suggestion request (async consideration for future)
- **Collector Service Calls**: 1-3 seconds per dry run request
- **Mitigation**: Implement caching layer in future phase (out of scope for this proposal)

### Security Impact
- **API Keys**: LLM API keys must be stored securely in environment variables
- **No new security risks**: All external calls use existing HTTP client patterns
