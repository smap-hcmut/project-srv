# Design: Keyword LLM Integration Implementation

## Context

The keyword module (`internal/keyword`) was created in a previous change but contains only placeholder/mock implementations. This design document outlines the technical decisions for implementing real LLM integration and Collector Service integration to deliver the intended functionality.

**Current State:**
- Module structure exists: `internal/keyword/usecase/` with interface, suggester, validator, tester
- Integration points exist: Keyword service injected into ProjectUseCase
- API endpoints exist: `/projects/keywords/suggest`, `/projects/keywords/dry-run`
- Implementation is mock: Hardcoded suggestions, no LLM calls, no Collector Service calls

**Goal:**
Replace mock implementations with real integrations while maintaining the existing architecture and API contracts.

## Goals / Non-Goals

### Goals
1. **LLM Integration**: Connect to real LLM provider (Gemini/OpenAI) for intelligent keyword suggestions
2. **Ambiguity Detection**: Use LLM to detect ambiguous keywords (e.g., "Apple" → fruit vs tech)
3. **Collector Integration**: Connect to Collector Service for real dry run data
4. **Resilience**: Graceful degradation when external services are unavailable
5. **Maintainability**: Clean abstraction for multiple LLM providers
6. **Testability**: Mockable interfaces for all external dependencies

### Non-Goals
1. **Caching Layer**: Out of scope for this phase (future enhancement)
2. **Rate Limiting**: Out of scope for this phase (future enhancement)
3. **Async Processing**: LLM calls are synchronous for now (future optimization)
4. **Multiple LLM Providers**: Support one provider initially (Gemini), add OpenAI later if needed
5. **Metrics/Observability**: Basic logging only, detailed metrics in future phase

## Decisions

### Decision 1: LLM Service Abstraction

**What**: Create `pkg/llm/` package with interface-based design

**Why**:
- Allows switching between LLM providers without changing business logic
- Enables testing with mock implementations
- Follows existing pattern in codebase (e.g., `pkg/discord/`)

**Alternatives Considered**:
1. **Direct integration in usecase**: Rejected - violates separation of concerns
2. **External service wrapper**: Rejected - adds unnecessary complexity
3. **Interface in internal/keyword**: Rejected - LLM is reusable across services

**Implementation**:
```go
// pkg/llm/interface.go
type Provider interface {
    SuggestKeywords(ctx context.Context, brandName string) ([]string, []string, error)
    CheckAmbiguity(ctx context.Context, keyword string) (bool, string, error)
}

// pkg/llm/gemini.go - Gemini implementation
// pkg/llm/openai.go - OpenAI implementation (future)
```

### Decision 2: LLM Provider Selection

**What**: Start with Google Gemini Flash (gemini-1.5-flash)

**Why**:
- Cost-effective: Free tier available, lower cost than GPT-4
- Fast: Optimized for speed, good for real-time suggestions
- Good quality: Sufficient for keyword suggestions and ambiguity detection
- Easy integration: Simple REST API

**Alternatives Considered**:
1. **OpenAI GPT-4o-mini**: More expensive, similar quality
2. **OpenAI GPT-4**: Too expensive for this use case
3. **Claude**: Good quality but less common, harder to find examples

**Migration Path**: Add OpenAI implementation later if needed (interface allows easy addition)

### Decision 3: Graceful Degradation Strategy

**What**: Fallback to basic suggestions when LLM unavailable

**Why**:
- Better UX: Users can still use the system even if LLM is down
- Resilience: System doesn't break on external service failures
- Progressive enhancement: Basic functionality always works

**Implementation**:
```go
func (uc *usecase) suggestProcessing(ctx context.Context, brandName string) ([]string, []string, error) {
    niche, negative, err := uc.llmProvider.SuggestKeywords(ctx, brandName)
    if err != nil {
        uc.l.Warnf(ctx, "LLM suggestion failed, using fallback: %v", err)
        return uc.fallbackSuggestions(brandName), nil
    }
    return niche, negative, nil
}
```

**Alternatives Considered**:
1. **Fail Fast**: Reject request if LLM unavailable - Rejected: Poor UX
2. **Queue for Retry**: Too complex for this phase
3. **Return Empty**: Rejected: Users expect some suggestions

### Decision 4: Collector Service Integration

**What**: HTTP client in `pkg/collector/` package to call external Collector Service

**Context**: Collector Service is a separate microservice (in different repository) that collects data from social media platforms based on keywords. The `pkg/collector/` package is only an HTTP client to communicate with Collector Service API.

**Why**:
- Simple and straightforward HTTP client (similar to `pkg/discord/`)
- Easy to test with mock HTTP client
- No need for gRPC complexity
- Follows existing patterns in codebase

**API Contract** (Defined for this implementation):
```
POST {COLLECTOR_SERVICE_URL}/api/v1/collector/dry-run

Request:
{
  "keywords": ["VinFast", "VF3"],
  "limit": 10
}

Response (200 OK):
{
  "posts": [
    {
      "id": "post-uuid-123",
      "content": "Just bought a new VinFast VF3! It's amazing...",
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

**Mock URL for Development**:
- Development: `http://localhost:8081/api/v1/collector/dry-run` (or configured via env)
- Production: Configured via `COLLECTOR_SERVICE_URL` environment variable

**Post Data Structure**:
```go
type Post struct {
    ID        string    `json:"id"`           // Unique post identifier
    Content   string    `json:"content"`       // Post text content
    Source    string    `json:"source"`        // Platform: "facebook", "twitter", "reddit", etc.
    SourceID  string    `json:"source_id"`    // Original post ID on platform
    Author    string    `json:"author"`       // Author name/username
    AuthorID  string    `json:"author_id"`    // Author identifier on platform
    Date      time.Time `json:"date"`         // Post creation date (ISO 8601)
    URL       string    `json:"url"`          // Direct link to original post
    Engagement Engagement `json:"engagement"` // Engagement metrics
    Metadata  Metadata   `json:"metadata"`     // Additional metadata
}

type Engagement struct {
    Likes    int `json:"likes"`
    Comments int `json:"comments"`
    Shares   int `json:"shares"`
}

type Metadata struct {
    Language string `json:"language"` // "vi", "en", etc.
    Sentiment string `json:"sentiment"` // "positive", "negative", "neutral"
}
```

**Alternatives Considered**:
1. **gRPC**: Rejected - Overkill, HTTP is sufficient
2. **Message Queue**: Rejected - Dry run should be synchronous for immediate feedback
3. **Direct HTTP in usecase**: Rejected - Violates separation of concerns

### Decision 5: Error Handling Strategy

**What**: Structured error types with context

**Why**:
- Better error messages for users
- Easier debugging with structured logs
- Consistent error handling across module

**Implementation**:
```go
// pkg/llm/errors.go
var (
    ErrLLMUnavailable = errors.New("LLM service unavailable")
    ErrLLMTimeout = errors.New("LLM request timeout")
    ErrLLMInvalidResponse = errors.New("LLM returned invalid response")
)

// pkg/collector/errors.go
var (
    ErrCollectorUnavailable = errors.New("collector service unavailable")
    ErrCollectorTimeout = errors.New("collector request timeout")
)
```

**Error Flow**:
1. External service error → Log with context
2. Map to domain error → Return to usecase
3. Usecase decides → Fallback or propagate
4. Handler maps → HTTP error response

### Decision 6: Configuration Management

**What**: Environment variables via `config/config.go`

**Why**:
- Consistent with existing pattern
- Easy to change per environment
- Secure (API keys in env, not code)

**Configuration Structure**:
```go
type LLMConfig struct {
    Provider   string `env:"LLM_PROVIDER" envDefault:"gemini"`
    APIKey     string `env:"LLM_API_KEY"`
    Model      string `env:"LLM_MODEL" envDefault:"gemini-1.5-flash"`
    Timeout    int    `env:"LLM_TIMEOUT" envDefault:"30"`
    MaxRetries int    `env:"LLM_MAX_RETRIES" envDefault:"3"`
}

type CollectorConfig struct {
    BaseURL string `env:"COLLECTOR_SERVICE_URL"`
    Timeout int    `env:"COLLECTOR_TIMEOUT" envDefault:"30"`
}
```

### Decision 7: Retry Logic

**What**: Simple retry with exponential backoff for LLM calls

**Why**:
- LLM APIs can be flaky
- Retry transient failures automatically
- Improve reliability without user intervention

**Implementation**:
- Max 3 retries
- Exponential backoff: 1s, 2s, 4s
- Only retry on timeout/network errors, not on 4xx errors

**Alternatives Considered**:
1. **No Retry**: Rejected - Too many false failures
2. **Complex Retry Library**: Rejected - Overkill for this use case

### Decision 8: Ambiguity Detection Scope

**What**: Only check single-word keywords for ambiguity

**Why**:
- Performance: Multi-word keywords are usually specific enough
- Cost: Reduce LLM API calls
- User experience: Most ambiguous cases are single words

**Implementation**:
```go
if isSingleWord(keyword) {
    isAmbiguous, context, err := uc.llmProvider.CheckAmbiguity(ctx, keyword)
    // ...
}
```

**Alternatives Considered**:
1. **Check All Keywords**: Rejected - Too expensive
2. **User-Configurable**: Rejected - Too complex for this phase

## Risks / Trade-offs

### Risk 1: LLM API Costs
**Risk**: High usage could lead to unexpected costs
**Mitigation**: 
- Start with Gemini Flash (free tier available)
- Monitor usage in logs
- Add rate limiting in future phase

### Risk 2: LLM API Latency
**Risk**: Slow responses (2-5s) could degrade UX
**Mitigation**:
- Use fast model (Gemini Flash)
- Implement timeout (30s max)
- Consider async processing in future if needed

### Risk 3: Collector Service Unavailable
**Risk**: Dry run fails if Collector Service is down
**Mitigation**:
- Clear error messages to users
- Log errors for monitoring
- Consider fallback mock data in future (not in this phase)

### Risk 4: LLM Response Quality
**Risk**: LLM might suggest irrelevant keywords
**Mitigation**:
- Post-validate suggestions with existing validator
- Users can review and select/deselect suggestions
- Monitor and adjust prompts if needed

### Risk 5: API Key Security
**Risk**: API keys exposed in environment
**Mitigation**:
- Use environment variables (not hardcoded)
- Document secure key management
- Use secrets management in production (K8s secrets)

## Migration Plan

### Phase 1: Infrastructure (Week 1)
1. Create `pkg/llm/` package with interface
2. Implement Gemini provider
3. Create `pkg/collector/` package
4. Add configurations

### Phase 2: Integration (Week 2)
1. Update keyword usecase to use LLM provider
2. Update keyword usecase to use Collector client
3. Add error handling and fallbacks
4. Update dependency injection

### Phase 3: Testing & Polish (Week 3)
1. Write unit tests with mocks
2. Integration testing (optional, with real services)
3. Error handling refinement
4. Documentation updates

### Rollback Plan
- Feature flag: Can disable LLM integration via config (fallback always works)
- Gradual rollout: Deploy to staging first, monitor, then production
- Quick revert: Remove LLM calls, restore fallback logic

## Open Questions

1. ~~**Collector Service API Spec**: What is the exact API contract?~~ ✅ **RESOLVED**: Defined in Decision 4
2. **LLM Prompt Engineering**: What prompts give best results? (to be refined during implementation)
3. **Caching Strategy**: When to add caching? (future phase, but worth planning)
4. **Rate Limiting**: Per-user or global? (future phase)
5. **Collector Service Authentication**: Does Collector Service require authentication? (to be confirmed during integration)

## Implementation Notes

### LLM Prompt Design
Initial prompts (to be refined):
- **Suggestion**: "Suggest 5-10 niche keywords and 5-10 negative keywords for brand: {brandName}. Return as JSON: {\"niche\": [...], \"negative\": [...]}"
- **Ambiguity**: "Is the keyword '{keyword}' ambiguous? Return JSON: {\"ambiguous\": true/false, \"context\": \"explanation\"}"

### Testing Strategy
1. **Unit Tests**: Mock LLM provider and Collector client
2. **Integration Tests**: Optional, test with real services in staging
3. **E2E Tests**: Test full flow from API to response

### Code Organization
```
pkg/llm/
├── interface.go      # Provider interface
├── gemini.go         # Gemini implementation
├── openai.go         # OpenAI implementation (future)
├── errors.go         # Error types
└── new.go            # Constructor

pkg/collector/
├── client.go         # HTTP client interface
├── types.go          # Post, Request, Response types
├── errors.go         # Error types
└── new.go            # Constructor
```

---

**Last Updated**: 2025-01-27
**Status**: Design for Implementation
**Author**: AI Assistant
