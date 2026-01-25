# Redis Pub/Sub Refactor - Troubleshooting Guide

## Overview

This guide helps diagnose and resolve issues with the refactored Redis Pub/Sub implementation.

## Topic Patterns

### New Topic Patterns

- **Dry-run jobs**: `job:{jobID}:{userID}`
- **Project progress**: `project:{projectID}:{userID}`

### Legacy Topic Pattern (deprecated)

- **All messages**: `user_noti:{userID}`

## Common Issues

### Issue 1: Messages Not Reaching WebSocket Clients

**Symptoms:**

- WebSocket clients not receiving real-time updates
- No errors in project service logs
- Redis publish appears successful

**Diagnosis:**

1. Check WebSocket service subscription patterns:

   ```bash
   # Verify WebSocket service is subscribing to new patterns
   redis-cli PUBSUB CHANNELS "job:*"
   redis-cli PUBSUB CHANNELS "project:*"
   ```

2. Verify message is being published:

   ```bash
   # Subscribe to channels to see messages
   redis-cli PSUBSCRIBE "job:*"
   redis-cli PSUBSCRIBE "project:*"
   ```

3. Check project service logs:
   ```bash
   kubectl logs -n production -l app=project-service | grep "published to Redis"
   ```

**Resolution:**

- Ensure WebSocket service is updated to subscribe to new patterns
- Verify Redis connectivity between services
- Check for network policies blocking pub/sub traffic

---

### Issue 2: Job Mapping Not Found

**Symptoms:**

- Error: `job mapping not found for jobID: {jobID}`
- Dry-run callbacks failing
- Log message: `webhook.HandleDryRunCallback.getJobMapping failed`

**Diagnosis:**

1. Check if job mapping was stored:

   ```bash
   redis-cli GET "job:mapping:{jobID}"
   ```

2. Verify job creation flow stores mapping:

   ```bash
   kubectl logs -n production -l app=project-service | grep "Stored job mapping"
   ```

3. Check TTL on job mapping:
   ```bash
   redis-cli TTL "job:mapping:{jobID}"
   ```

**Resolution:**

- Ensure `StoreJobMapping()` is called when jobs are created
- Verify Redis connectivity during job creation
- Check if TTL (7 days) is appropriate for job lifecycle

---

### Issue 3: Message Serialization Errors

**Symptoms:**

- Error: `failed to marshal JobMessage` or `failed to marshal ProjectMessage`
- Messages not being published
- Log message: `webhook.Handle*.Marshal failed`

**Diagnosis:**

1. Check for invalid data in callback:

   ```bash
   kubectl logs -n production -l app=project-service | grep "Marshal failed"
   ```

2. Verify callback payload structure matches expected format

**Resolution:**

- Validate callback data before transformation
- Check for nil pointer issues in nested structures
- Ensure all required fields are present

---

### Issue 4: Redis Publish Failures

**Symptoms:**

- Error: `failed to publish to Redis`
- Messages not reaching subscribers
- Log message: `webhook.Handle*.Publish failed`

**Diagnosis:**

1. Check Redis connectivity:

   ```bash
   redis-cli PING
   ```

2. Check Redis memory usage:

   ```bash
   redis-cli INFO memory
   ```

3. Check for connection pool exhaustion:
   ```bash
   kubectl logs -n production -l app=project-service | grep "connection"
   ```

**Resolution:**

- Verify Redis server is healthy
- Check connection pool configuration
- Review Redis cluster status if using cluster mode

---

### Issue 5: Incorrect Message Format

**Symptoms:**

- WebSocket clients receiving malformed data
- JSON parsing errors on client side
- Missing fields in messages

**Diagnosis:**

1. Capture actual message content:

   ```bash
   redis-cli PSUBSCRIBE "job:*" | head -20
   ```

2. Compare with expected format:
   ```json
   // JobMessage
   {
     "platform": "TIKTOK",
     "status": "COMPLETED",
     "batch": {
       "keyword": "...",
       "content_list": [...],
       "crawled_at": "..."
     },
     "progress": {
       "current": 1,
       "total": 1,
       "percentage": 100.0,
       "eta": 0.0,
       "errors": []
     }
   }
   ```

**Resolution:**

- Verify transformation functions are working correctly
- Check JSON tags on struct fields
- Run unit tests to validate message structure

---

### Issue 6: Performance Degradation

**Symptoms:**

- Increased latency in message delivery
- Higher CPU/memory usage
- Slower callback processing

**Diagnosis:**

1. Check transformation performance:

   ```bash
   go test -bench=. ./internal/webhook/usecase/...
   ```

2. Monitor Redis latency:

   ```bash
   redis-cli --latency
   ```

3. Check message sizes:
   ```bash
   kubectl logs -n production -l app=project-service | grep "message_size"
   ```

**Resolution:**

- Review transformation logic for inefficiencies
- Check for memory leaks in message construction
- Consider message compression if sizes are large

---

## Log Patterns

### Successful Operations

```
webhook.HandleDryRunCallback: published to Redis: topic=job:{jobID}:{userID}, job_id={jobID}, platform=TIKTOK, status=COMPLETED, content_count=5, message_size=1234
webhook.HandleProgressCallback: published to Redis: topic=project:{projectID}:{userID}, project_id={projectID}, status=PROCESSING, progress=50/100 (50.0%), message_size=256
```

### Error Patterns

```
webhook.HandleDryRunCallback.getJobMapping failed: job_id={jobID}, platform=tiktok, status=success, error=job mapping not found
webhook.HandleDryRunCallback.Publish failed: topic=job:{jobID}:{userID}, job_id={jobID}, message_size=1234, error=connection refused
webhook.HandleProgressCallback: validation failed: project_id=, user_id={userID}
```

## Monitoring Queries

### Check Message Publishing Rate

```promql
rate(redis_pubsub_messages_published_total{topic=~"job:.*|project:.*"}[5m])
```

### Check Error Rate

```promql
rate(webhook_callback_errors_total[5m])
```

### Check Message Size Distribution

```promql
histogram_quantile(0.95, rate(webhook_message_size_bytes_bucket[5m]))
```

## Support Escalation

If issues persist after troubleshooting:

1. **Level 1**: Check logs and metrics, apply known fixes
2. **Level 2**: Engage project service team for code-level investigation
3. **Level 3**: Engage platform team for infrastructure issues
