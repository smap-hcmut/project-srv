# Optimize Dry-Run Keywords Processing with Percentage-Based Sampling

## Summary

**Change ID:** optimize-dryrun-keywords  
**Type:** Performance Enhancement  
**Scope:** Project Service, Collector Service  
**Priority:** High  

This proposal optimizes the dry-run keywords processing feature to ensure predictable performance and WebSocket compatibility by implementing percentage-based keyword sampling in the Project Service business layer.

## Problem Statement

### Current Performance Issues

Based on real crawler performance data:
- **Processing time**: ~16 seconds per keyword (tesla: 08:30:58 → 08:31:14)
- **Scale implications**: 50 keywords = 800 seconds (13.3 minutes) → **Exceeds 90s WebSocket timeout**
- **Performance variance**: Up to 50x time difference between small (5 keywords = 80s) and large projects (50 keywords = 800s)
- **Critical constraint**: WebSocket connections timeout after 90 seconds, causing users to lose results

### WebSocket Integration Risk

- **Client behavior**: After calling dry-run, client establishes WebSocket connection
- **Timeout limit**: 90 seconds maximum connection time
- **Data delivery**: Results streamed via Redis Pub/Sub through WebSocket
- **User impact**: If dry-run > 90s → connection lost → no results received

### Resource Inefficiency

- Crawler processes entire keyword arrays regardless of business requirements
- No sampling mechanism at business layer
- Technical limits (3 items/keyword) insufficient for time management

## Proposed Solution

### Separation of Concerns Architecture

```
Project Service (Business Layer)
├── Decision: How many keywords for dry-run?
├── Apply percentage-based sampling (Option 2)
├── Keyword selection strategy (priority/diversity)
└── Send optimized request to Collector

Collector Service (Technical Layer)  
├── Receive pre-optimized keyword list
├── Enforce technical limits (3 items/keyword)
└── Forward to Crawler
```

### Percentage-Based Sampling Strategy (Chosen Option)

**Configuration:**
- **Rule**: Select maximum 10% of keywords, with minimum 3, maximum 5
- **Logic**:
  - 50 keywords → 5 keywords (10%, capped at maximum)
  - 10 keywords → 3 keywords (30%, but minimum applied)
  - 100 keywords → 5 keywords (5%, capped at maximum)
- **Performance target**: Always ≤ 80 seconds (5 × 16s), with 10s buffer before WebSocket timeout

### Keyword Selection Strategy

**Priority-based Selection:**
1. **High Priority**: Keywords with high search volume/importance metadata
2. **Medium Priority**: Keywords with low competition scores
3. **Random Selection**: If no metadata available, random sampling
4. **Diversity Ensurance**: Select from different categories to ensure representative sample

## Implementation Approach

### Configuration Parameters

```yaml
# Dry-run Sampling Configuration
DRY_RUN_SAMPLING_STRATEGY: "percentage"
DRY_RUN_PERCENTAGE: 10
DRY_RUN_MIN_KEYWORDS: 3
DRY_RUN_MAX_KEYWORDS: 5
DRY_RUN_TIMEOUT: 85
DRY_RUN_WS_TIMEOUT: 90
DRY_RUN_BUFFER_TIME: 10
DRY_RUN_KEYWORD_TIME_ESTIMATE: 16
```

### Sampling Algorithm Flow

```
Input: Project with N keywords
↓
Apply Percentage Sampling:
- Calculate: min(max(N × 10%, MIN_KEYWORDS), MAX_KEYWORDS)
- Examples:
  * 50 keywords → min(max(5, 3), 5) = 5 keywords
  * 10 keywords → min(max(1, 3), 5) = 3 keywords
  * 2 keywords → min(max(0.2, 3), 5) = 3 keywords (but limited by available)
↓
Apply Selection Strategy:
- Priority-based: Sort by metadata priority → take top K
- Diversity: Ensure representation across keyword categories
- Fallback: Random selection if no metadata
↓
Output: Optimized keyword list (3-5 keywords, ≤80s processing time)
```

## Success Criteria

### Performance Metrics
- **Execution time variance**: < 10% between dry-runs (due to fixed keyword count)
- **95th percentile**: < 80 seconds
- **99th percentile**: < 85 seconds (with network delays)
- **WebSocket success rate**: > 99.5%
- **Average processing time**: ~16 seconds per keyword (monitoring target)

### Business Metrics
- **User satisfaction**: Improved due to predictable timing
- **Feature adoption**: Increased dry-run usage
- **Conversion rate**: Improved from dry-run to full project execution

### Technical Metrics
- **System stability**: No degradation during concurrent dry-runs
- **Error rate**: < 1%
- **Resource utilization**: Stable CPU/memory usage

## Risk Assessment

### Sampling Accuracy Risk
- **Risk**: Sample may not be representative of full keyword set
- **Mitigation**: 
  - Priority-based selection ensures important keywords included
  - Diversity sampling across categories
  - User communication about sampling methodology

### Performance Regression Risk
- **Risk**: New sampling logic could introduce latency
- **Mitigation**:
  - Lightweight sampling algorithms
  - Performance benchmarking before deployment
  - Feature flag for gradual rollout

### WebSocket Timeout Risk
- **Risk**: Despite optimization, some dry-runs might still timeout
- **Mitigation**:
  - Hard limit at 75s with emergency fallback to 3 keywords
  - Progress indicators for users
  - Retry mechanism with smaller samples
  - Fallback to partial results delivery

## Migration Strategy

### Phase 1: Business Logic Implementation (1 week)
- Implement keyword sampling algorithms in Project Service
- Add configuration management
- Create sampling strategy interfaces

### Phase 2: Integration & Testing (1 week)
- Integrate with existing dry-run flow
- Performance testing with various keyword set sizes
- A/B testing against current implementation

### Phase 3: Rollout (1 week)
- Feature flag controlled deployment
- Monitor performance metrics
- User feedback collection

### Phase 4: Optimization (Ongoing)
- Fine-tune sampling parameters based on real usage
- Optimize selection algorithms
- Scale monitoring and alerting

## Dependencies

### External Systems
- **Collector Service**: Already supports keyword arrays, no changes needed
- **WebSocket Service**: Already handles Redis messages, no changes needed
- **Redis Pub/Sub**: Current implementation sufficient

### Configuration Systems
- Environment variable management
- Feature flag system for gradual rollout
- Monitoring and alerting infrastructure

## Backwards Compatibility

- **API Compatibility**: No breaking changes to dry-run API
- **Payload Compatibility**: Collector Service receives same message format
- **User Experience**: Transparent optimization, users see improved performance
- **Feature Flag**: Allows rollback to original implementation if needed

## Monitoring and Observability

### Key Metrics to Track
- Dry-run execution time distribution
- Keyword sampling ratio (actual vs requested)
- WebSocket connection success rate
- User satisfaction scores
- System resource utilization during dry-runs

### Alerting Thresholds
- Dry-run time > 85 seconds (approaching timeout)
- WebSocket connection failure rate > 1%
- Keyword sampling failures
- Performance regression indicators

## Testing Strategy

### Unit Testing
- Keyword sampling algorithm correctness
- Edge cases (empty arrays, single keywords)
- Configuration parameter validation
- Selection strategy algorithms

### Integration Testing  
- End-to-end dry-run flow with sampled keywords
- WebSocket timeout scenarios
- Performance under load
- Error handling and fallback mechanisms

### Performance Testing
- Benchmark sampling overhead
- Load testing with concurrent dry-runs
- WebSocket connection sustainability
- Memory and CPU impact assessment

## Success Definition

This proposal succeeds when:

1. **Predictable Performance**: All dry-runs complete within 80 seconds with <10% variance
2. **WebSocket Compatibility**: >99.5% of connections successfully receive results
3. **Maintained Quality**: Sampled results provide sufficient insight for user decision-making
4. **Scalability**: System handles concurrent dry-runs without performance degradation
5. **User Satisfaction**: Improved user experience with faster, more reliable dry-run results

The optimization transforms dry-run from an unpredictable, potentially failing operation into a reliable, fast preview feature that scales consistently regardless of project size.