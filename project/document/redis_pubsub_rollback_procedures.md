# Redis Pub/Sub Refactor - Rollback Procedures

## Overview

This document describes the rollback procedures for the Redis Pub/Sub refactor. The refactor changes topic patterns from `user_noti:{userID}` to topic-specific patterns (`job:{jobID}:{userID}` and `project:{projectID}:{userID}`).

## When to Rollback

Consider rollback if any of the following occur:

1. **Message Delivery Failures**: WebSocket clients not receiving messages
2. **High Error Rates**: Significant increase in Redis publish errors
3. **Performance Degradation**: Message processing latency exceeds acceptable thresholds
4. **Data Integrity Issues**: Messages malformed or missing required fields

## Rollback Strategies

### Strategy 1: Kubernetes Rollback (Recommended)

**Use when**: Deployment issues detected within minutes of deployment.

```bash
# Check current deployment status
kubectl rollout status deployment/project-service -n production

# Rollback to previous revision
kubectl rollout undo deployment/project-service -n production

# Verify rollback completed
kubectl rollout status deployment/project-service -n production

# Check pod status
kubectl get pods -n production -l app=project-service
```

### Strategy 2: Manual Version Rollback

**Use when**: Need to rollback to a specific version.

```bash
# List deployment history
kubectl rollout history deployment/project-service -n production

# Rollback to specific revision
kubectl rollout undo deployment/project-service -n production --to-revision=<revision-number>
```

### Strategy 3: Feature Flag Disable (If Implemented)

**Use when**: Feature flag system is in place.

```bash
# Disable new topic patterns via configuration
kubectl set env deployment/project-service -n production USE_NEW_TOPIC_PATTERNS=false
```

## Rollback Verification

After rollback, verify the following:

### 1. Service Health

```bash
# Check pod status
kubectl get pods -n production -l app=project-service

# Check service logs for errors
kubectl logs -n production -l app=project-service --tail=100
```

### 2. Message Delivery

- [ ] Trigger a test dry-run job
- [ ] Verify message published to `user_noti:{userID}` topic
- [ ] Confirm WebSocket client receives the message

### 3. Error Rates

- [ ] Check error rate metrics in monitoring dashboard
- [ ] Verify no increase in Redis connection errors
- [ ] Confirm no message serialization errors

## Post-Rollback Actions

1. **Document the Issue**

   - Record the symptoms observed
   - Note the time of detection and rollback
   - Capture relevant logs and metrics

2. **Notify Stakeholders**

   - Inform WebSocket service team
   - Update operations team
   - Notify product team if user-facing impact

3. **Root Cause Analysis**

   - Review logs for error patterns
   - Check message structure differences
   - Verify Redis connectivity and performance

4. **Plan Remediation**
   - Identify fixes needed
   - Update tests to cover the failure case
   - Schedule re-deployment after fixes

## Compatibility Notes

### WebSocket Service Compatibility

The WebSocket service should support both old and new topic patterns during the transition period:

```go
// WebSocket service should handle both patterns
func handleMessage(channel string, data []byte) {
    if strings.HasPrefix(channel, "job:") || strings.HasPrefix(channel, "project:") {
        // Handle new format
        handleNewFormat(channel, data)
    } else if strings.HasPrefix(channel, "user_noti:") {
        // Handle legacy format
        handleLegacyFormat(channel, data)
    }
}
```

### Message Format Compatibility

Old format (legacy):

```json
{
  "type": "dryrun_result",
  "payload": { ... }
}
```

New format:

```json
{
  "platform": "TIKTOK",
  "status": "COMPLETED",
  "batch": { ... },
  "progress": { ... }
}
```

## Emergency Contacts

- **On-Call Engineer**: [Contact Info]
- **Project Service Lead**: [Contact Info]
- **WebSocket Service Lead**: [Contact Info]
- **DevOps Team**: [Contact Info]
