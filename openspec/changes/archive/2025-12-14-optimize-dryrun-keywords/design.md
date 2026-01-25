# Design Document: Dry-Run Keywords Optimization

## Architecture Overview

### High-Level Flow

```
User Request → Project API → Keyword Sampler → Optimized Keywords → Collector → Results → WebSocket
```

### Component Responsibilities

**1. Project Service - Business Layer**
- Receives user dry-run requests with full keyword arrays
- Applies business-driven sampling rules (percentage-based)
- Implements keyword selection strategies (priority/diversity)
- Sends optimized keyword subset to Collector Service
- Manages sampling configuration and monitoring

**2. Collector Service - Technical Layer**
- Receives pre-optimized keyword lists
- Applies technical constraints (3 items/keyword, 5 comments/post)
- Executes crawling operations
- Returns results via webhook callbacks
- Maintains existing technical limits unchanged

**3. WebSocket Service - Delivery Layer**
- Continues receiving results via Redis Pub/Sub
- Delivers real-time results to connected clients
- No changes required to existing implementation

## Detailed Component Design

### 1. Keyword Sampling Component

#### DryRunKeywordSampler Interface

```go
type DryRunKeywordSampler interface {
    SampleKeywords(ctx context.Context, input SamplingInput) (SamplingOutput, error)
}

type SamplingInput struct {
    Keywords    []string
    UserID      string
    ProjectID   string
    Strategy    SamplingStrategy
    Config      SamplingConfig
}

type SamplingOutput struct {
    SelectedKeywords []string
    TotalKeywords    int
    SamplingRatio    float64
    SelectionMethod  string
    EstimatedTime    time.Duration
}

type SamplingStrategy string
const (
    PercentageStrategy SamplingStrategy = "percentage"
    FixedStrategy      SamplingStrategy = "fixed"
    TieredStrategy     SamplingStrategy = "tiered"
)
```

#### Percentage-Based Implementation

```go
type percentageSampler struct {
    config SamplingConfig
    logger log.Logger
}

func (s *percentageSampler) SampleKeywords(ctx context.Context, input SamplingInput) (SamplingOutput, error) {
    // Calculate target count based on percentage
    targetCount := s.calculateTargetCount(len(input.Keywords), input.Config)
    
    // Apply selection strategy
    selected, method := s.selectKeywords(input.Keywords, targetCount)
    
    // Calculate estimates
    estimatedTime := time.Duration(len(selected)) * input.Config.KeywordTimeEstimate
    ratio := float64(len(selected)) / float64(len(input.Keywords))
    
    return SamplingOutput{
        SelectedKeywords: selected,
        TotalKeywords:    len(input.Keywords),
        SamplingRatio:    ratio,
        SelectionMethod:  method,
        EstimatedTime:    estimatedTime,
    }, nil
}

func (s *percentageSampler) calculateTargetCount(total int, config SamplingConfig) int {
    // Apply percentage-based calculation
    percentageCount := int(float64(total) * config.Percentage / 100.0)
    
    // Apply min/max constraints
    targetCount := percentageCount
    if targetCount < config.MinKeywords {
        targetCount = config.MinKeywords
    }
    if targetCount > config.MaxKeywords {
        targetCount = config.MaxKeywords
    }
    
    // Cannot exceed available keywords
    if targetCount > total {
        targetCount = total
    }
    
    return targetCount
}
```

#### Keyword Selection Strategies

```go
type KeywordSelector interface {
    Select(keywords []string, count int) ([]string, string)
}

// Priority-based selector (future enhancement)
type prioritySelector struct {
    metadataStore KeywordMetadataStore
}

func (s *prioritySelector) Select(keywords []string, count int) ([]string, string) {
    // Sort keywords by priority metadata (search volume, competition, etc.)
    // Select top N keywords
    // Return selected keywords and "priority" method
}

// Diversity selector (future enhancement)
type diversitySelector struct {
    categorizer KeywordCategorizer
}

func (s *diversitySelector) Select(keywords []string, count int) ([]string, string) {
    // Group keywords by category/theme
    // Select representative keywords from each category
    // Ensure diverse representation
    // Return selected keywords and "diversity" method
}

// Random selector (current implementation)
type randomSelector struct {
    rand *rand.Rand
}

func (s *randomSelector) Select(keywords []string, count int) ([]string, string) {
    if count >= len(keywords) {
        return keywords, "all"
    }
    
    // Shuffle and select first N keywords
    shuffled := make([]string, len(keywords))
    copy(shuffled, keywords)
    s.rand.Shuffle(len(shuffled), func(i, j int) {
        shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
    })
    
    return shuffled[:count], "random"
}
```

### 2. Configuration Management

#### SamplingConfig Structure

```go
type SamplingConfig struct {
    // Percentage-based sampling
    Percentage           float64       `env:"DRY_RUN_PERCENTAGE" default:"10"`
    MinKeywords         int           `env:"DRY_RUN_MIN_KEYWORDS" default:"3"`
    MaxKeywords         int           `env:"DRY_RUN_MAX_KEYWORDS" default:"5"`
    
    // Timing configuration
    KeywordTimeEstimate time.Duration `env:"DRY_RUN_KEYWORD_TIME_ESTIMATE" default:"16s"`
    TimeoutThreshold    time.Duration `env:"DRY_RUN_TIMEOUT" default:"85s"`
    WebSocketTimeout    time.Duration `env:"DRY_RUN_WS_TIMEOUT" default:"90s"`
    BufferTime         time.Duration `env:"DRY_RUN_BUFFER_TIME" default:"10s"`
    
    // Strategy selection
    DefaultStrategy     SamplingStrategy `env:"DRY_RUN_SAMPLING_STRATEGY" default:"percentage"`
    
    // Emergency fallback
    EmergencyThreshold  time.Duration `env:"DRY_RUN_EMERGENCY_THRESHOLD" default:"70s"`
    EmergencyKeywords   int           `env:"DRY_RUN_EMERGENCY_KEYWORDS" default:"3"`
}
```

#### Configuration Validation

```go
func (c *SamplingConfig) Validate() error {
    if c.Percentage <= 0 || c.Percentage > 100 {
        return fmt.Errorf("percentage must be between 0 and 100, got %f", c.Percentage)
    }
    
    if c.MinKeywords < 1 {
        return fmt.Errorf("minimum keywords must be at least 1, got %d", c.MinKeywords)
    }
    
    if c.MaxKeywords < c.MinKeywords {
        return fmt.Errorf("maximum keywords (%d) cannot be less than minimum (%d)", 
            c.MaxKeywords, c.MinKeywords)
    }
    
    estimatedMaxTime := time.Duration(c.MaxKeywords) * c.KeywordTimeEstimate
    if estimatedMaxTime >= c.WebSocketTimeout {
        return fmt.Errorf("maximum estimated time (%v) exceeds WebSocket timeout (%v)", 
            estimatedMaxTime, c.WebSocketTimeout)
    }
    
    return nil
}
```

### 3. Integration with Existing Dry-Run Flow

#### Updated UseCase Implementation

```go
func (uc *usecase) DryRunKeywords(ctx context.Context, sc model.Scope, input project.DryRunKeywordsInput) (project.DryRunKeywordsOutput, error) {
    // Validate input
    if len(input.Keywords) == 0 {
        return project.DryRunKeywordsOutput{}, project.ErrInvalidKeywords
    }
    
    // Apply keyword sampling
    samplingInput := SamplingInput{
        Keywords:  input.Keywords,
        UserID:    sc.UserID,
        Strategy:  uc.config.Sampling.DefaultStrategy,
        Config:    uc.config.Sampling,
    }
    
    samplingResult, err := uc.sampler.SampleKeywords(ctx, samplingInput)
    if err != nil {
        uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords: sampling failed: %v", err)
        return project.DryRunKeywordsOutput{}, err
    }
    
    // Log sampling information
    uc.l.Infof(ctx, "Sampled %d/%d keywords for user %s (%.1f%%, method: %s, estimated: %v)",
        len(samplingResult.SelectedKeywords), samplingResult.TotalKeywords,
        sc.UserID, samplingResult.SamplingRatio*100,
        samplingResult.SelectionMethod, samplingResult.EstimatedTime)
    
    // Generate job ID
    jobID := uuid.New().String()
    
    // Create crawl request with sampled keywords
    crawlReq := rabbitmq.DryRunCrawlRequest{
        JobID:    jobID,
        TaskType: "dryrun_keyword",
        Payload: rabbitmq.DryRunPayload{
            Keywords:         samplingResult.SelectedKeywords, // Use sampled keywords
            LimitPerKeyword:  3,
            IncludeComments:  true,
            MaxComments:      5,
        },
        UserID:       sc.UserID,
        EmittedAt:    time.Now(),
        Attempt:      1,
        MaxAttempts:  3,
    }
    
    // Publish to RabbitMQ
    if err := uc.producer.PublishDryRunTask(ctx, crawlReq); err != nil {
        uc.l.Errorf(ctx, "internal.project.usecase.DryRunKeywords: failed to publish task: %v", err)
        return project.DryRunKeywordsOutput{}, err
    }
    
    return project.DryRunKeywordsOutput{
        JobID:             jobID,
        Status:            "processing",
        SampledKeywords:   samplingResult.SelectedKeywords,
        TotalKeywords:     samplingResult.TotalKeywords,
        SamplingRatio:     samplingResult.SamplingRatio,
        EstimatedDuration: samplingResult.EstimatedTime,
    }, nil
}
```

### 4. Basic Performance Logging

#### Simple Performance Tracking

```go
type DryRunPerformanceInfo struct {
    // Essential timing info
    ExecutionTime       time.Duration
    SamplingTime        time.Duration
    
    // Sampling info
    OriginalKeywords    int
    SampledKeywords     int
    SamplingRatio       float64
    SelectionMethod     string
    
    // Success info
    Success             bool
    WebSocketSuccess    bool
    FallbackApplied     bool
    
    // User context
    UserID              string
    Timestamp           time.Time
}

func (uc *usecase) logPerformance(ctx context.Context, info DryRunPerformanceInfo) {
    // Log basic performance information
    uc.l.Infof(ctx, "DryRun performance: duration=%v, sampling=%d/%d (%.1f%%), method=%s, success=%t, fallback=%t",
        info.ExecutionTime, info.SampledKeywords, info.OriginalKeywords,
        info.SamplingRatio*100, info.SelectionMethod, info.Success, info.FallbackApplied)
    
    // Log warnings for performance issues
    if info.ExecutionTime > 80*time.Second {
        uc.l.Warnf(ctx, "DryRun approaching timeout: duration=%v, user=%s", info.ExecutionTime, info.UserID)
    }
}
```

#### Basic Performance Validation

```go
func (uc *usecase) validatePerformance(ctx context.Context, info DryRunPerformanceInfo) {
    // Check execution time compliance
    if info.ExecutionTime > 85*time.Second {
        uc.l.Errorf(ctx, "DryRun exceeded timeout threshold: duration=%v, user=%s", info.ExecutionTime, info.UserID)
    }
    
    // Check WebSocket delivery success
    if !info.WebSocketSuccess {
        uc.l.Errorf(ctx, "WebSocket delivery failed for user %s", info.UserID)
    }
    
    // Log fallback usage for tracking
    if info.FallbackApplied {
        uc.l.Warnf(ctx, "Fallback mechanism activated for user %s", info.UserID)
    }
}
```

### 5. Error Handling and Fallback Mechanisms

#### Emergency Fallback

```go
func (s *percentageSampler) applyEmergencyFallback(keywords []string, config SamplingConfig) []string {
    // If estimated time would exceed emergency threshold, use minimum keywords
    if len(keywords) * int(config.KeywordTimeEstimate.Seconds()) > int(config.EmergencyThreshold.Seconds()) {
        emergencyCount := config.EmergencyKeywords
        if emergencyCount > len(keywords) {
            emergencyCount = len(keywords)
        }
        
        // Use random selection for emergency fallback
        selector := &randomSelector{rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
        selected, _ := selector.Select(keywords, emergencyCount)
        
        return selected
    }
    
    return keywords
}
```

#### Graceful Degradation

```go
func (uc *usecase) handleSamplingError(ctx context.Context, originalKeywords []string, err error) []string {
    uc.l.Warnf(ctx, "Sampling failed, falling back to simple truncation: %v", err)
    
    // Simple fallback: use first N keywords up to maximum
    maxKeywords := uc.config.Sampling.MaxKeywords
    if len(originalKeywords) <= maxKeywords {
        return originalKeywords
    }
    
    return originalKeywords[:maxKeywords]
}
```

### 6. Testing Strategy

#### Unit Test Structure

```go
func TestPercentageSampler_SampleKeywords(t *testing.T) {
    tests := []struct {
        name           string
        keywords       []string
        config         SamplingConfig
        expectedCount  int
        expectedRatio  float64
    }{
        {
            name:          "50 keywords with 10% sampling",
            keywords:      generateKeywords(50),
            config:        SamplingConfig{Percentage: 10, MinKeywords: 3, MaxKeywords: 5},
            expectedCount: 5, // 10% of 50 = 5, capped at max
            expectedRatio: 0.1,
        },
        {
            name:          "10 keywords with 10% sampling",
            keywords:      generateKeywords(10),
            config:        SamplingConfig{Percentage: 10, MinKeywords: 3, MaxKeywords: 5},
            expectedCount: 3, // 10% of 10 = 1, but minimum is 3
            expectedRatio: 0.3,
        },
        // Add more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sampler := NewPercentageSampler(tt.config, mockLogger)
            result, err := sampler.SampleKeywords(context.Background(), SamplingInput{
                Keywords: tt.keywords,
                Config:   tt.config,
            })
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedCount, len(result.SelectedKeywords))
            assert.InDelta(t, tt.expectedRatio, result.SamplingRatio, 0.01)
        })
    }
}
```

## Implementation Phases

### Phase 1: Core Sampling Logic
1. Implement `DryRunKeywordSampler` interface
2. Create percentage-based sampling algorithm
3. Add configuration management
4. Unit tests for sampling logic

### Phase 2: Integration
1. Integrate sampler into existing dry-run usecase
2. Update API response to include sampling information
3. Add basic performance logging
4. Integration tests

### Phase 3: Fallback & Validation
1. Add fallback mechanisms
2. Implement basic performance validation
3. Add performance logging and issue detection
4. Load testing and optimization

### Phase 4: Advanced Features (Future)
1. Priority-based keyword selection
2. Diversity sampling algorithms
3. User preferences for sampling strategy
4. Enhanced monitoring and analytics (if needed)

This design ensures the optimization is robust, maintainable, and provides clear performance benefits while maintaining the existing API contract and user experience.