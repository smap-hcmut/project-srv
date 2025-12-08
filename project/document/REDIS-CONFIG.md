# Redis Configuration Guide

## 📦 Deployed Redis Service

Redis standalone instance deployed in `smap` namespace with:

- **Version**: Redis 7.2-alpine
- **Persistence**: RDB + AOF enabled
- **Storage**: 5Gi PersistentVolume (local-path)
- **Memory**: 512Mi limit, 410mb maxmemory
- **Password**: `21042004` (change in production!)

---

## 🔗 Connection Endpoints

### 1. **From within cluster (apps)**

```yaml
REDIS_HOST: redis-service:6379
REDIS_PASSWORD: 21042004
REDIS_DB: 0
REDIS_STATE_DB: 1
```

**Redis URI format:**

```
redis://:21042004@redis-service:6379/0  # Main DB
redis://:21042004@redis-service:6379/1  # State DB
```

### 2. **From outside cluster (dev/test)**

```yaml
REDIS_HOST: 172.16.21.231:6379
REDIS_PASSWORD: 21042004
```

**Redis URI format:**

```
redis://:21042004@172.16.21.231:6379/0
```

### 3. **Port Forward (local development)**

```bash
kubectl port-forward -n smap svc/redis-service 6379:6379

# Then connect to localhost
redis-cli -h localhost -p 6379 -a 21042004
```

---

## 🎯 Services Available

| Service Name         | Type             | IP            | Port | Purpose                       |
| -------------------- | ---------------- | ------------- | ---- | ----------------------------- |
| `redis-headless`     | ClusterIP (None) | -             | 6379 | StatefulSet management        |
| `redis-service`      | ClusterIP        | 10.96.193.196 | 6379 | App connections (recommended) |
| `redis-loadbalancer` | LoadBalancer     | 172.16.21.231 | 6379 | External access (dev/test)    |

---

## 📝 Service Configurations

### Project Service

**File**: `project/k8s/configmap.yaml` + `project/k8s/secret.yaml.template`

```yaml
# ConfigMap (non-sensitive)
REDIS_HOST: "redis-service:6379"
REDIS_DB: "0"
REDIS_STATE_DB: "1"
REDIS_MIN_IDLE_CONNS: "10"
REDIS_POOL_SIZE: "100"
REDIS_POOL_TIMEOUT: "30"

# Secret (sensitive)
REDIS_PASSWORD: "21042004"
```

### Identity Service

**TODO**: Add Redis config when identity service needs Redis

### Collector Service

**TODO**: Add Redis config when collector service needs Redis

### WebSocket Service

**File**: `websocket/.env`

```yaml
REDIS_HOST: redis-service
REDIS_PORT: 6379
REDIS_PASSWORD: 21042004
REDIS_DB: 0
REDIS_USE_TLS: false
```

---

## 🧪 Testing Connection

### From within cluster:

```bash
kubectl run -it --rm redis-test --image=redis:alpine --restart=Never -n smap -- \
  redis-cli -h redis-service -p 6379 -a 21042004 ping
```

### From outside cluster:

```bash
redis-cli -h 172.16.21.231 -p 6379 -a 21042004 ping
```

### Test commands:

```bash
redis-cli -h 172.16.21.231 -p 6379 -a 21042004

> PING
PONG

> SET test "hello"
OK

> GET test
"hello"

> SELECT 1
OK

> KEYS *
(empty array)

> INFO server
# Server info...
```

---

## 🔒 Security Notes

1. **Change password in production!**

   - Update `redis-manifest.yaml` secret
   - Update all service configs
   - Restart Redis pod

2. **Network isolation:**

   - ClusterIP services only accessible within cluster
   - LoadBalancer IP only accessible from same network (172.16.21.x)
   - Consider NetworkPolicy for additional security

3. **Disable dangerous commands (optional):**
   Edit `redis-manifest.yaml` ConfigMap:
   ```yaml
   rename-command FLUSHDB ""
   rename-command FLUSHALL ""
   rename-command CONFIG ""
   ```

---

## 📊 Monitoring

### Check Redis status:

```bash
kubectl get statefulset redis -n smap
kubectl get pods -n smap | grep redis
kubectl get pvc -n smap
kubectl get svc -n smap | grep redis
```

### View logs:

```bash
kubectl logs redis-0 -n smap
kubectl logs redis-0 -n smap -f  # Follow logs
```

### Check metrics:

```bash
kubectl exec -it redis-0 -n smap -- redis-cli -a 21042004 INFO stats
kubectl exec -it redis-0 -n smap -- redis-cli -a 21042004 INFO memory
```

---

## 🔄 Maintenance

### Backup data:

```bash
kubectl exec redis-0 -n smap -- redis-cli -a 21042004 BGSAVE
kubectl cp smap/redis-0:/data/dump.rdb ./redis-backup-$(date +%Y%m%d).rdb
```

### Restore data:

```bash
kubectl cp ./redis-backup.rdb smap/redis-0:/data/dump.rdb
kubectl delete pod redis-0 -n smap  # Pod will restart and load backup
```

### Scale (not recommended for standalone):

```bash
# Redis standalone should stay at 1 replica
kubectl scale statefulset redis -n smap --replicas=1
```

---

## 📚 References

- Redis manifest: `redis-manifest.yaml`
- Project config: `project/k8s/configmap.yaml`, `project/k8s/secret.yaml.template`
- MetalLB pool: `smap-pool` (172.16.21.230-239)
- StorageClass: `local-path`
