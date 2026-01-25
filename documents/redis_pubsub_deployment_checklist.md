# Redis Pub/Sub Refactor - Deployment Checklist

## Pre-Deployment Verification

### 1. Code Verification

- [ ] All unit tests pass (`go test ./internal/webhook/usecase/...`)
- [ ] All integration tests pass
- [ ] Performance benchmarks show no regression
- [ ] Code review completed and approved
- [ ] No linting errors or warnings

### 2. Environment Preparation

- [ ] Staging Redis configuration verified
- [ ] Staging environment accessible
- [ ] WebSocket service team notified of deployment schedule
- [ ] Operations team notified of new topic patterns

### 3. Monitoring Setup

- [ ] Log aggregation configured for new log patterns
- [ ] Alerts configured for Redis publish failures
- [ ] Dashboard updated with new topic metrics

## Deployment Steps

### Stage 1: Staging Deployment

1. **Deploy to Staging**

   ```bash
   # Deploy the updated project service to staging
   kubectl apply -f k8s/staging/project-service.yaml
   ```

2. **Verify Staging Deployment**

   - [ ] Service starts without errors
   - [ ] Health check endpoints respond correctly
   - [ ] Logs show expected startup messages

3. **Test New Topic Patterns**

   - [ ] Trigger a dry-run job and verify `job:{jobID}:{userID}` topic
   - [ ] Trigger project progress and verify `project:{projectID}:{userID}` topic
   - [ ] Verify WebSocket service receives messages correctly

4. **Verify Message Structure**
   - [ ] JobMessage contains correct Platform, Status, Batch, Progress fields
   - [ ] ProjectMessage contains correct Status, Progress fields
   - [ ] JSON serialization matches specification

### Stage 2: Production Deployment

1. **Pre-Production Checks**

   - [ ] Staging tests completed successfully
   - [ ] WebSocket service ready for new patterns
   - [ ] Rollback procedure tested in staging
   - [ ] On-call team notified

2. **Deploy to Production**

   ```bash
   # Deploy the updated project service to production
   kubectl apply -f k8s/production/project-service.yaml
   ```

3. **Post-Deployment Verification**
   - [ ] Service starts without errors
   - [ ] No increase in error rates
   - [ ] Message delivery confirmed
   - [ ] WebSocket clients receiving updates

## Rollback Procedure

If issues are detected:

1. **Immediate Rollback**

   ```bash
   # Rollback to previous deployment
   kubectl rollout undo deployment/project-service -n production
   ```

2. **Verify Rollback**

   - [ ] Service returns to previous version
   - [ ] Old topic patterns working
   - [ ] No message loss during rollback

3. **Post-Rollback Actions**
   - [ ] Document the issue encountered
   - [ ] Notify stakeholders
   - [ ] Plan remediation

## Success Criteria

- [ ] Zero message loss during deployment
- [ ] No increase in error rates
- [ ] WebSocket clients receive messages correctly
- [ ] Performance metrics within acceptable range
- [ ] No customer-reported issues

## Contacts

- **Project Service Team**: [Team Contact]
- **WebSocket Service Team**: [Team Contact]
- **Operations Team**: [Team Contact]
- **On-Call**: [On-Call Contact]
