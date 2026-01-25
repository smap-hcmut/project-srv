# Keyword Sampling Capability

## Purpose

Implements percentage-based keyword sampling for dry-run operations to ensure predictable performance and WebSocket compatibility.

## ADDED Requirements

### Requirement: Percentage-Based Keyword Sampling

The system SHALL implement percentage-based sampling to limit dry-run processing time. The system MUST ensure WebSocket compatibility by keeping execution time under 90 seconds. The system SHALL apply sampling at the business logic layer before sending keywords to Collector Service.

#### Scenario: Large Project Keyword Sampling

Given a project with 50 keywords for dry-run
And the system is configured with 10% sampling percentage, minimum 3 keywords, maximum 5 keywords
When the user requests a dry-run via `POST /projects/keywords/dry-run`
Then the system calculates target count as min(max(50 × 10% = 5, 3), 5) = 5 keywords
And the system selects 5 representative keywords using the configured selection strategy
And the system sends only the 5 selected keywords to Collector Service
And the estimated processing time is 5 × 16 seconds = 80 seconds (within 90s WebSocket timeout)
And the API response includes sampling metadata:
```json
{
  "job_id": "uuid",
  "status": "processing", 
  "sampled_keywords": ["selected1", "selected2", "selected3", "selected4", "selected5"],
  "total_keywords": 50,
  "sampling_ratio": 0.1,
  "estimated_duration": "80s"
}
```

#### Scenario: Small Project Minimum Keywords

Given a project with 10 keywords for dry-run  
And the system is configured with 10% sampling percentage, minimum 3 keywords, maximum 5 keywords
When the user requests a dry-run
Then the system calculates target count as min(max(10 × 10% = 1, 3), 5) = 3 keywords (minimum applied)
And the system selects 3 keywords from the available 10
And the system sends the 3 selected keywords to Collector Service
And the estimated processing time is 3 × 16 seconds = 48 seconds
And the sampling ratio is 3/10 = 0.3 (30%)

#### Scenario: Very Small Project All Keywords

Given a project with 2 keywords for dry-run
And the system is configured with 10% sampling percentage, minimum 3 keywords, maximum 5 keywords  
When the user requests a dry-run
Then the system calculates target count as min(max(2 × 10% = 0.2, 3), 5) = 3, but limited by available = 2 keywords
And the system selects all 2 available keywords
And the system sends both keywords to Collector Service  
And the estimated processing time is 2 × 16 seconds = 32 seconds
And the sampling ratio is 2/2 = 1.0 (100%)

#### Scenario: Maximum Keywords Capping

Given a project with 100 keywords for dry-run
And the system is configured with 10% sampling percentage, minimum 3 keywords, maximum 5 keywords
When the user requests a dry-run  
Then the system calculates target count as min(max(100 × 10% = 10, 3), 5) = 5 keywords (maximum applied)
And the system selects 5 representative keywords from the 100 available
And the system sends only the 5 selected keywords to Collector Service
And the estimated processing time is 5 × 16 seconds = 80 seconds
And the sampling ratio is 5/100 = 0.05 (5%)

### Requirement: Sampling Configuration Management

The system SHALL provide configurable sampling parameters via environment variables. The system MUST validate configuration on startup and provide sensible defaults. The system SHALL support different sampling strategies through configuration.

#### Scenario: Environment Variable Configuration

Given the system is configured with environment variables:
```env
DRY_RUN_PERCENTAGE=15
DRY_RUN_MIN_KEYWORDS=2  
DRY_RUN_MAX_KEYWORDS=7
DRY_RUN_KEYWORD_TIME_ESTIMATE=18s
DRY_RUN_TIMEOUT=85s
DRY_RUN_SAMPLING_STRATEGY=percentage
```
When the system initializes
Then the sampling configuration is loaded with the specified values
And percentage-based sampling uses 15% as the target ratio
And minimum keywords is set to 2, maximum to 7  
And keyword time estimate is 18 seconds for timing calculations
And the configuration is validated for consistency

#### Scenario: Default Configuration Fallback

Given no sampling environment variables are configured
When the system initializes
Then the system loads default configuration values:
- Percentage: 10%
- Minimum keywords: 3
- Maximum keywords: 5  
- Keyword time estimate: 16 seconds
- Timeout: 85 seconds
- Strategy: "percentage"
And the system logs a warning about using default configuration
And the system functions normally with default values

#### Scenario: Invalid Configuration Validation

Given the system is configured with invalid values:
```env
DRY_RUN_PERCENTAGE=150  # Invalid: > 100
DRY_RUN_MIN_KEYWORDS=0   # Invalid: < 1
DRY_RUN_MAX_KEYWORDS=2   # Invalid: < min_keywords(3 default)
```
When the system initializes
Then the system validates the configuration
And configuration validation fails with specific error messages:
- "percentage must be between 0 and 100, got 150"
- "minimum keywords must be at least 1, got 0"  
- "maximum keywords (2) cannot be less than minimum (3)"
And the system refuses to start with invalid configuration
And operators receive clear guidance on fixing the configuration

### Requirement: Keyword Selection Strategies  

The system SHALL support multiple keyword selection strategies for sampling. The system MUST provide random selection as the default strategy. The system SHALL design interfaces to support future priority-based and diversity-based selection.

#### Scenario: Random Selection Strategy

Given a project with keywords ["tesla", "vinfast", "honda", "toyota", "bmw", "audi", "ford", "chevy"] (8 keywords)
And the system is configured to select 3 keywords using random selection strategy
When the user requests a dry-run
Then the system randomly shuffles the keyword list
And selects the first 3 keywords from the shuffled list
And the selection is uniform and unbiased across multiple requests
And the API response indicates selection method as "random"
And the selected keywords vary between different dry-run requests for the same project

#### Scenario: All Keywords Selection

Given a project with 3 keywords ["brand1", "brand2", "brand3"]
And the system is configured to select up to 5 keywords
When the user requests a dry-run
Then the system selects all 3 available keywords
And no sampling is applied since available keywords ≤ target count
And the API response indicates selection method as "all"
And all original keywords are preserved in the request to Collector Service

#### Scenario: Future Strategy Interface Compatibility

Given the system implements the KeywordSelector interface
When future priority-based or diversity-based strategies are developed
Then they can be integrated without breaking existing functionality  
And the interface supports strategy-specific configuration parameters
And the selection method is reported accurately in API responses
And users can configure which strategy to use via environment variables

### Requirement: Basic Performance Logging

The system SHALL provide basic logging of sampling operations and dry-run performance. The system MUST track essential timing and sampling information for debugging. The system SHALL log critical performance issues for investigation.

#### Scenario: Basic Sampling Logging

Given a successful dry-run operation with sampling
When the sampling process completes
Then the system logs essential information:
- Original keyword count: 50
- Sampled keyword count: 5
- Sampling ratio: 0.1 (10%)
- Selection method: "random"
- Estimated total time: 80s
And information is logged in structured format for debugging:
```
INFO: Dry-run sampling - original=50, sampled=5, ratio=10%, method=random, estimated=80s
```

#### Scenario: Performance Issue Logging

Given a dry-run execution that approaches timeout thresholds
When execution time exceeds 80 seconds or other issues occur
Then the system logs a warning with context:
```
WARN: Dry-run execution approaching timeout - duration=82s, user=user123, keywords=5
```
And when fallback mechanisms are triggered, logs the action taken:
```
WARN: Emergency fallback applied - original=50, reduced=3, reason=timeout_risk
```

#### Scenario: Error Logging for Investigation

Given sampling operations encounter errors or performance degradation
When issues occur that affect user experience
Then the system logs sufficient information for debugging:
- Error context and stack traces
- User ID and request parameters
- Timing information and system state
- Actions taken (fallbacks, retries, etc.)
And logs are accessible for operations team investigation

### Requirement: Emergency Fallback Mechanisms

The system SHALL implement emergency fallback mechanisms when normal sampling fails. The system MUST provide graceful degradation to ensure dry-run functionality remains available. The system SHALL automatically apply more aggressive sampling when timeout risk is detected.

#### Scenario: Sampling Algorithm Failure Fallback

Given the sampling algorithm encounters an error (null pointer, configuration issue, etc.)
When the dry-run request is being processed  
Then the system logs the sampling error with full context
And falls back to simple keyword truncation: use first N keywords up to maximum limit
And continues processing with the fallback keywords
And the API response indicates fallback was applied:
```json
{
  "job_id": "uuid",
  "status": "processing",
  "sampled_keywords": ["first", "second", "third", "fourth", "fifth"],
  "total_keywords": 50,
  "sampling_ratio": 0.1,
  "estimated_duration": "80s",
  "fallback_applied": true,
  "fallback_reason": "sampling_algorithm_error"
}
```

#### Scenario: Emergency Timeout Risk Fallback

Given a dry-run request where estimated time exceeds the emergency threshold (70s)
When the system calculates that normal sampling would result in >70s processing time
Then the system automatically applies emergency fallback:
- Reduces target keywords to emergency minimum (3 keywords)
- Uses fastest selection method (first N keywords)
- Logs emergency fallback activation
And the estimated time is recalculated to be under the emergency threshold
And users receive a notification that emergency optimization was applied

#### Scenario: WebSocket Timeout Prevention

Given a dry-run operation that is approaching the 90-second WebSocket timeout
When the system detects execution time >75 seconds during processing
Then the system immediately applies emergency measures:
- Cancels any remaining keyword processing beyond minimum required
- Publishes partial results to Redis for WebSocket delivery
- Logs timeout prevention action
And users receive partial results before WebSocket timeout
And the system maintains connection reliability >99.5%