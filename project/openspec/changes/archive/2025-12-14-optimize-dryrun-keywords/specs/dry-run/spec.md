# Dry-Run Performance Optimization

## Purpose

Enhances the existing dry-run capability with predictable performance guarantees and WebSocket timeout compatibility through intelligent keyword sampling.

## MODIFIED Requirements

### Requirement: Fetch Sample Data with Performance Optimization

The system SHALL fetch a sample of real data based on optimized keywords by calling the Collector Service. The system MUST apply keyword sampling before calling Collector Service to ensure execution time remains under WebSocket timeout limits. The system SHALL maintain data quality while optimizing performance.

#### Scenario: Optimized Dry Run Execution with Large Keyword Set

Given a project with 50 keywords ["VinFast", "VF3", "tesla", "honda", ...] (50 total)
And the system is configured with percentage-based sampling (10%, min 3, max 5)
When the user requests a dry run via `POST /projects/keywords/dry-run`
Then the system applies keyword sampling to select 5 representative keywords
And validates the sampled keywords using existing keyword validation rules  
And calls Collector Service with the 5 sampled keywords and limit=10
And Collector Service returns 5-10 recent posts per sampled keyword (25-50 posts total)
And the total execution time is approximately 5 × 16s = 80s (within 90s WebSocket timeout)
And the system returns the posts to the user with sampling metadata:
```json
{
  "job_id": "uuid",
  "status": "processing",
  "sampled_keywords": ["VinFast", "VF3", "tesla", "honda", "toyota"], 
  "total_keywords": 50,
  "sampling_ratio": 0.1,
  "estimated_duration": "80s"
}
```

#### Scenario: Small Project No Sampling Required

Given a project with 3 keywords ["VinFast", "VF3", "tesla"]
And the system is configured with percentage-based sampling (10%, min 3, max 5)
When the user requests a dry run
Then the system calculates sampling: min(max(3 × 10% = 0.3, 3), 5) = 3 keywords
And all 3 keywords are selected (no sampling needed)
And validates all 3 keywords using existing validation rules
And calls Collector Service with all 3 keywords and limit=10
And the execution time is approximately 3 × 16s = 48s (well within timeout)
And the API response indicates all keywords were used (sampling_ratio = 1.0)

#### Scenario: Dry Run with Invalid Keywords in Sample  

Given a project with 20 keywords including some invalid entries (too short, invalid characters)
And the system applies sampling to select 5 keywords
When the sampled set contains invalid keywords
Then the system validates the sampled keywords first
And invalid keywords in the sample are rejected with appropriate error messages
And the system attempts to select replacement keywords from the remaining valid keywords
And if insufficient valid keywords remain, adjusts the sample size accordingly
And the Collector Service is only called with valid keywords
And the API returns validation errors for the invalid keywords that were sampled

#### Scenario: Performance-Optimized Error Handling

Given a project with optimized keywords for dry-run 
When the Collector Service is unavailable or times out
Then the error handling follows existing patterns (logging, user-friendly messages)
But the error detection happens faster due to reduced keyword processing load
And the system can handle more concurrent dry-run requests due to predictable resource usage
And monitoring can better predict and alert on system capacity issues

### Requirement: Visualize Results with Sampling Context

The system SHALL display the fetched sample data with clear indication of sampling applied. The system MUST provide transparency about which keywords were selected and why. The system SHALL maintain user confidence in result representativeness despite sampling.

#### Scenario: Display Sample with Sampling Information

Given a successful dry run response from a sampled keyword set
When the data is received from Collector Service  
Then the system formats the posts into the existing consistent structure
And each post includes the standard fields: content, source, date
And the API response includes additional sampling context:
```json
{
  "posts": [
    {
      "content": "Post content...",
      "source": "tiktok", 
      "date": "2025-01-27T10:00:00Z",
      "keyword": "VinFast",
      // ... other existing fields
    }
  ],
  "sampling_info": {
    "sampled_keywords": ["VinFast", "VF3", "tesla"],
    "total_keywords": 50,
    "sampling_ratio": 0.06,
    "selection_method": "random",
    "excluded_keywords": ["honda", "toyota", ...],
    "estimated_time_saved": "720s"
  }
}
```

#### Scenario: Full Keywords Used Indication

Given a dry run where no sampling was applied (small project)
When the results are returned to the user
Then the response clearly indicates that all keywords were used:
```json
{
  "sampling_info": {
    "sampled_keywords": ["VinFast", "VF3", "tesla"],
    "total_keywords": 3,
    "sampling_ratio": 1.0, 
    "selection_method": "all",
    "excluded_keywords": [],
    "estimated_time_saved": "0s"
  }
}
```

## ADDED Requirements

### Requirement: WebSocket Timeout Compatibility

The system SHALL ensure all dry-run operations complete within WebSocket connection timeouts. The system MUST provide predictable execution times regardless of project size. The system SHALL implement safeguards to prevent WebSocket connection loss.

#### Scenario: WebSocket Timeout Prevention

Given any dry-run request regardless of original keyword count
When the system processes the dry-run
Then the estimated execution time must be ≤ 85 seconds (5s buffer before 90s WebSocket timeout)
And the actual execution time must not exceed 90 seconds in any normal scenario
And if execution approaches the timeout threshold, emergency fallback activates
And WebSocket connections maintain >99.5% success rate for result delivery

#### Scenario: Predictable Timing Across Project Sizes

Given projects of varying sizes:
- Small project: 5 keywords
- Medium project: 25 keywords  
- Large project: 100 keywords
When dry-run requests are made for each project
Then the execution time variance between all projects is <10 seconds
And all projects complete in 48-80 seconds range (3-5 keywords × 16s)
And users experience consistent performance regardless of their project size
And system resource usage is predictable for capacity planning

#### Scenario: Emergency Timeout Prevention

Given a dry-run operation that unexpectedly takes longer than estimated
When the system detects execution time approaching 85 seconds
Then the system immediately triggers emergency protocols:
- Cancels remaining keyword processing
- Publishes partial results to Redis
- Logs timeout prevention action with user and timing context
- Ensures WebSocket delivery happens before 90s timeout
And users receive partial results rather than losing connection
And the system maintains reliability metrics despite the emergency action

### Requirement: Basic Performance Validation

The system SHALL provide basic performance validation for dry-run optimization. The system MUST verify that sampling achieves timing objectives. The system SHALL log sufficient information for debugging and validation.

#### Scenario: Performance Target Validation

Given dry-run operations with keyword sampling enabled
When each dry-run completes (successfully or with errors)
Then the system validates performance targets:
- Execution time stays under 85 seconds
- WebSocket delivery completes successfully
- Sampling overhead remains minimal (<5ms)
- Fallback mechanisms activate when needed
And basic performance information is logged:
```
INFO: Dry-run performance - duration=78s, websocket_success=true, fallback=false
```

#### Scenario: Performance Issue Detection

Given a dry-run operation that exhibits performance problems
When execution time approaches limits or errors occur
Then the system detects and logs issues:
- Execution times approaching WebSocket timeout
- Fallback mechanism activation frequency
- User-impacting errors or failures
And provides actionable information for investigation

#### Scenario: Success Rate Validation

Given multiple dry-run operations over time
When validating system performance
Then basic success indicators are tracked:
- WebSocket connection success rate (target >99.5%)
- Execution time compliance (target <85s consistently)
- Fallback activation frequency (should be minimal)
- Error rates (should remain stable or improve)
And performance validation helps ensure the optimization meets its goals