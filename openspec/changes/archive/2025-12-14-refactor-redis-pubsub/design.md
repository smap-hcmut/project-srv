# Redis Pub/Sub Refactor Design Document

## Architecture Overview

### Current Architecture
```
Crawler/Collector → Project Service → Redis → WebSocket Service → Client
                       ↓ 
              Single Topic Pattern
           "user_noti:{userID}" + type discrimination
```

### Proposed Architecture  
```
Crawler/Collector → Project Service → Redis → WebSocket Service → Client
                       ↓
              Topic-Specific Patterns
      "job:{jobID}:{userID}" | "project:{projectID}:{userID}"
```

## Design Principles

### 1. Topic-Based Routing
**Rationale**: Replace type discrimination with topic-based routing for better scalability and performance.

**Implementation**:
- Dry-run jobs: `job:{jobID}:{userID}`
- Project progress: `project:{projectID}:{userID}`
- WebSocket service subscribes to patterns: `job:*`, `project:*`

**Benefits**:
- Eliminates message parsing overhead for routing
- Allows independent scaling of different message types
- Enables topic-specific subscription strategies

### 2. Type-Safe Message Structures
**Rationale**: Replace generic `map[string]interface{}` with strongly-typed Go structs.

**Design Decisions**:
- Separate message types for different topics (JobMessage, ProjectMessage)
- Shared components (Progress, BatchData) for consistency
- Enums for controlled vocabularies (Platform, Status, MediaType)

**Benefits**:
- Compile-time type checking
- Better IDE support and autocomplete
- Self-documenting code structure
- Reduced runtime errors

### 3. Transformation Layer Architecture
**Rationale**: Maintain backward compatibility while enabling new message formats.

```go
Input (Crawler/Collector) → Transformation → Output (Redis) → WebSocket
     CallbackRequest      → Transform Func → JobMessage/ProjectMessage
```

**Transformation Functions**:
- `TransformDryRunCallback()`: Converts crawler callback to JobMessage
- `TransformProjectCallback()`: Converts progress callback to ProjectMessage
- Helper functions for complex data transformations

## Data Flow Design

### Dry-Run Job Flow
```
1. Crawler sends callback → Project Service
   CallbackRequest {
     JobID, Platform, Status,
     Payload { Content[], Errors[] }
   }

2. Project Service looks up userID from Redis job mapping

3. Transform to JobMessage:
   - Map content to BatchData
   - Map errors to Progress.Errors
   - Set platform and status enums

4. Publish to: job:{jobID}:{userID}
   JobMessage {
     Platform, Status, Batch, Progress
   }
```

### Project Progress Flow  
```
1. Collector sends callback → Project Service  
   ProgressCallbackRequest {
     ProjectID, UserID, Status, Total, Done, Errors
   }

2. Transform to ProjectMessage:
   - Calculate progress percentage
   - Map status to enum
   - Convert error count to messages

3. Publish to: project:{projectID}:{userID}
   ProjectMessage {
     Status, Progress
   }
```

## Message Structure Design

### Hierarchical Structure
```
JobMessage / ProjectMessage (Root)
├── Platform/Status (Enums)  
├── Progress (Shared)
│   ├── Current, Total, Percentage
│   ├── ETA (calculated)
│   └── Errors[] (string messages)
└── Batch (Job-specific)
    ├── Keyword, CrawledAt
    └── ContentList[]
        ├── ID, Text, Permalink
        ├── Author (nested)
        ├── Metrics (nested) 
        └── Media (nested)
```

### Design Rationale
- **Flat vs Nested**: Use nested structures for logical grouping (Author, Metrics, Media)
- **Optional Fields**: Use pointers for optional complex structures (`*BatchData`, `*Progress`)
- **Array Types**: Use slices for lists (ContentList, Errors)
- **Primitive Types**: Use appropriate Go types (int, float64, bool, string)

## Error Handling Strategy

### Transformation Errors
```go
// Graceful degradation approach
func TransformDryRunCallback(req CallbackRequest) JobMessage {
    message := JobMessage{
        Platform: mapPlatform(req.Platform), // Default fallback if unknown
        Status:   mapDryRunStatus(req.Status), // Default to PROCESSING
    }
    
    // Optional transformations - don't fail if problematic
    if batch, err := transformBatch(req.Payload); err == nil {
        message.Batch = batch
    }
    
    return message
}
```

### Publishing Errors
- Log transformation failures with full context
- Continue with partial message if possible
- Use structured logging for debugging
- Include original callback data in error logs

## Performance Considerations

### Message Size Optimization
- **Current**: Generic payload with type field overhead
- **Proposed**: Topic-specific messages without type discrimination
- **Savings**: ~10-15% message size reduction

### Processing Efficiency  
- **Current**: Parse every message to determine type
- **Proposed**: Topic-based routing, no parsing needed
- **Improvement**: ~20-30% processing time reduction

### Memory Usage
- **Current**: `map[string]interface{}` allocations
- **Proposed**: Pre-allocated struct fields
- **Improvement**: ~15-25% memory usage reduction

## Compatibility Strategy

### Transition Period
1. **Phase 1**: Deploy new publishing logic, WebSocket supports both formats
2. **Phase 2**: Monitor new topic patterns, verify correct message delivery
3. **Phase 3**: Gradually migrate WebSocket subscriptions to new patterns
4. **Phase 4**: Remove old topic support after full migration

### Fallback Mechanism
```go
// Example WebSocket subscriber compatibility
func handleMessage(channel string, data []byte) {
    if strings.Contains(channel, "job:") || strings.Contains(channel, "project:") {
        // Handle new format
        handleNewFormat(channel, data)
    } else if strings.Contains(channel, "user_noti:") {
        // Handle legacy format
        handleLegacyFormat(channel, data)
    }
}
```

## Security Considerations

### Topic Pattern Security
- **User Isolation**: Topic includes userID, ensuring messages only go to correct user
- **No Information Leakage**: JobID and ProjectID in topic don't expose sensitive data
- **Access Control**: WebSocket service validates userID before delivering messages

### Data Sanitization
- Validate all enum values with fallback defaults
- Sanitize content fields (text, URLs) if needed
- Ensure error messages don't leak sensitive information

## Monitoring & Observability

### New Metrics
- Message transformation success/failure rates
- Topic-specific publishing rates (`job:*` vs `project:*`)
- Message size distribution by topic type
- Transformation latency percentiles

### Logging Strategy
```go
// Structured logging for new patterns
uc.l.Infof(ctx, "Published to topic: topic=%s, message_type=%s, size=%d", 
    channel, messageType, len(body))

// Error context for debugging
uc.l.Errorf(ctx, "Transformation failed: job_id=%s, error=%v, payload=%+v", 
    req.JobID, err, req.Payload)
```

### Health Checks
- Verify Redis connectivity for new topic patterns
- Monitor message delivery success rates
- Track transformation error rates and patterns

## Testing Strategy

### Unit Testing
- Test each transformation function independently
- Mock external dependencies (Redis, callbacks)
- Test edge cases and error conditions
- Verify enum mapping correctness

### Integration Testing  
- End-to-end flow from callback to Redis publish
- Real Redis instance testing
- WebSocket service integration testing
- Load testing with realistic message volumes

### Compatibility Testing
- Test old and new formats side-by-side
- Verify graceful degradation scenarios
- Test rollback procedures
- Validate monitoring and alerting

## Future Extensibility

### New Message Types
- Framework supports easy addition of new topic patterns
- Transformation layer can be extended for new callback types
- Enum system allows new platforms, statuses, media types

### Enhanced Features
- Message versioning support in structures
- Advanced filtering and routing capabilities
- Message analytics and insights features
- Integration with external monitoring systems