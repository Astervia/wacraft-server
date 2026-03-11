# Horizontal Scaling - Test Plan

Tests for each phase, split into **unit tests** (standard `go test`) and **manual/integration tests** (curl, multi-instance scenarios). The project uses standard Go testing with `go test ./... -v`.

---

## Phase 1: Core Abstractions (`wacraft-core`)

### Unit Tests

#### 1.1 Redis Client — `wacraft-core/src/synch/redis/client_test.go`

```go
func TestNewClient_ParsesValidURL(t *testing.T)
// Create client with valid redis:// URL → no error

func TestNewClient_InvalidURL(t *testing.T)
// Create client with invalid URL → returns error

func TestPrefixKey(t *testing.T)
// PrefixKey("lock:abc") with prefix "wacraft:" → "wacraft:lock:abc"

func TestPingWithTimeout_NoRedis(t *testing.T)
// Ping with no Redis running → returns error within timeout
```

#### 1.2 Distributed Lock

**Memory** — `wacraft-core/src/synch/service/memory_lock_test.go`

```go
func TestMemoryLock_LockUnlock(t *testing.T)
// Lock(key) then Unlock(key) → no error, no deadlock

func TestMemoryLock_ConcurrentSameKey(t *testing.T)
// 10 goroutines Lock/Unlock same key, increment shared counter
// → counter == 10 (no race, mutual exclusion holds)

func TestMemoryLock_DifferentKeysParallel(t *testing.T)
// Lock("a") and Lock("b") concurrently → both acquire immediately (no contention)

func TestMemoryLock_ReentrantDeadlock(t *testing.T)
// Lock(key) twice from different goroutines → second blocks until first unlocks
```

**Redis** — `wacraft-core/src/synch/redis/redis_lock_test.go`

> These tests require a running Redis. Skip with `if os.Getenv("REDIS_URL") == ""`.

```go
func TestRedisLock_LockUnlock(t *testing.T)
// Lock(key) then Unlock(key) → key no longer exists in Redis

func TestRedisLock_MutualExclusion(t *testing.T)
// Two RedisLock instances (simulating two app instances) contend on same key
// → only one holds it at a time, shared counter incremented correctly

func TestRedisLock_TTLExpiry(t *testing.T)
// Lock with short TTL (100ms), don't unlock, wait 150ms
// → another lock acquisition succeeds (simulates instance crash recovery)

func TestRedisLock_UnlockOnlyOwner(t *testing.T)
// Lock from instance A, attempt unlock from instance B (different owner)
// → key still exists, lock NOT released

func TestRedisLock_ConcurrentHighContention(t *testing.T)
// 50 goroutines across 2 RedisLock instances, all contending on same key
// → shared counter incremented correctly to 50
```

#### 1.3 Distributed Pub/Sub

**Memory** — `wacraft-core/src/synch/service/memory_pubsub_test.go`

```go
func TestMemoryPubSub_PublishSubscribe(t *testing.T)
// Subscribe to "ch1", Publish "hello" → subscriber receives "hello"

func TestMemoryPubSub_MultipleSubscribers(t *testing.T)
// 3 subscribers on "ch1", Publish once → all 3 receive the message

func TestMemoryPubSub_Unsubscribe(t *testing.T)
// Subscribe, Unsubscribe, Publish → subscriber does NOT receive, channel closed

func TestMemoryPubSub_IsolatedChannels(t *testing.T)
// Subscribe to "ch1", Publish to "ch2" → "ch1" subscriber receives nothing

func TestMemoryPubSub_BufferFull(t *testing.T)
// Fill subscriber buffer (100 messages), publish one more → no panic, message dropped
```

**Redis** — `wacraft-core/src/synch/redis/redis_pubsub_test.go`

```go
func TestRedisPubSub_CrossInstance(t *testing.T)
// Two RedisPubSub instances (same Redis), subscribe on A, publish on B
// → A receives the message (proves cross-instance delivery)

func TestRedisPubSub_MultipleChannels(t *testing.T)
// Subscribe to "ch1" and "ch2", publish to "ch1" → only "ch1" subscriber receives

func TestRedisPubSub_Unsubscribe(t *testing.T)
// Subscribe, Unsubscribe, Publish → no message received, no error
```

#### 1.4 Distributed Counter

**Memory** — `wacraft-core/src/synch/service/memory_counter_test.go`

```go
func TestMemoryCounter_Increment(t *testing.T)
// Increment("k", 1) three times → Get("k") == 3

func TestMemoryCounter_ConcurrentIncrement(t *testing.T)
// 100 goroutines each Increment("k", 1) → Get("k") == 100

func TestMemoryCounter_TTLExpiry(t *testing.T)
// Increment, SetTTL(50ms), wait 60ms, Get → returns 0

func TestMemoryCounter_Delete(t *testing.T)
// Increment, Delete, Get → returns 0

func TestMemoryCounter_GetNonExistent(t *testing.T)
// Get("nonexistent") → returns 0, no error
```

**Redis** — `wacraft-core/src/synch/redis/redis_counter_test.go`

```go
func TestRedisCounter_CrossInstance(t *testing.T)
// Two RedisCounter instances, each Increment("k", 5)
// → Get("k") from either instance == 10

func TestRedisCounter_TTL(t *testing.T)
// Increment, SetTTL(100ms), wait 150ms → Get returns 0

func TestRedisCounter_Delete(t *testing.T)
// Increment, Delete → Get returns 0
```

#### 1.5 Distributed Cache

**Memory** — `wacraft-core/src/synch/service/memory_cache_test.go`

```go
func TestMemoryCache_SetGet(t *testing.T)
// Set("k", data, 5s), Get("k") → returns data, true

func TestMemoryCache_Miss(t *testing.T)
// Get("nonexistent") → returns nil, false, no error

func TestMemoryCache_TTLExpiry(t *testing.T)
// Set("k", data, 50ms), wait 60ms, Get("k") → returns nil, false

func TestMemoryCache_Delete(t *testing.T)
// Set, Delete, Get → returns nil, false

func TestMemoryCache_Invalidate(t *testing.T)
// Set "prefix:a", "prefix:b", "other:c"
// Invalidate("prefix:*") → "prefix:a" and "prefix:b" gone, "other:c" still exists
```

**Redis** — `wacraft-core/src/synch/redis/redis_cache_test.go`

```go
func TestRedisCache_CrossInstance(t *testing.T)
// Set from instance A, Get from instance B → returns data

func TestRedisCache_TTL(t *testing.T)
// Set with 100ms TTL, wait 150ms, Get → miss

func TestRedisCache_Invalidate(t *testing.T)
// Set multiple keys with shared prefix, Invalidate("prefix*") → all deleted
```

#### 1.6 Factory — `wacraft-core/src/synch/factory_test.go`

```go
func TestFactory_MemoryBackend(t *testing.T)
// NewFactory(BackendMemory, nil) → NewLock, NewPubSub, NewCounter, NewCache all return non-nil
// Verify returned types are memory implementations (type assertion)

func TestFactory_RedisBackend(t *testing.T)
// NewFactory(BackendRedis, redisClient) → all return Redis implementations

func TestFactory_BackendString(t *testing.T)
// Verify Backend() returns the configured backend
```

### Phase 1 Validation

```bash
# Unit tests (no Redis needed)
go test ./wacraft-core/src/synch/service/... -v

# Redis tests (requires Redis at REDIS_URL)
REDIS_URL=redis://localhost:6379 go test ./wacraft-core/src/synch/redis/... -v

# Factory tests
REDIS_URL=redis://localhost:6379 go test ./wacraft-core/src/synch/... -v

# Race detector
REDIS_URL=redis://localhost:6379 go test ./wacraft-core/src/synch/... -v -race
```

---

## Phase 2: Synchronization Migration

### Unit Tests

#### 2.1 Message-Status Sync

**Memory** — `src/message/service/message_status_sync_memory_test.go`

```go
func TestMemorySync_MessageThenStatus(t *testing.T)
// Goroutine A: AddMessage(wamID, 5s)
// Goroutine B (after 50ms): AddStatus(wamID, "sent", 5s)
// Goroutine A (after AddMessage returns): MessageSaved(wamID, "uuid-123", 5s)
// → AddStatus returns "uuid-123"

func TestMemorySync_StatusThenMessage(t *testing.T)
// Goroutine A: AddStatus(wamID, "sent", 5s) — arrives first, blocks
// Goroutine B (after 50ms): AddMessage(wamID, 5s)
// Goroutine B: MessageSaved(wamID, "uuid-456", 5s)
// → AddStatus returns "uuid-456"

func TestMemorySync_Timeout(t *testing.T)
// AddMessage(wamID, 100ms) with no status arriving → returns timeout error

func TestMemorySync_Rollback(t *testing.T)
// AddMessage → AddStatus → RollbackMessage
// → AddStatus returns empty string / rollback sentinel

func TestMemorySync_ConcurrentDifferentMessages(t *testing.T)
// 10 wamIDs, each with AddMessage + AddStatus in parallel
// → all 10 complete correctly, no cross-contamination
```

**Redis** — `src/message/service/message_status_sync_redis_test.go`

```go
func TestRedisSync_CrossInstance(t *testing.T)
// Two separate MessageStatusSync instances (simulating two app instances)
// Instance A: AddMessage(wamID)
// Instance B: AddStatus(wamID)
// Instance A: MessageSaved(wamID, "uuid-789")
// → Instance B's AddStatus returns "uuid-789"

func TestRedisSync_Timeout(t *testing.T)
// AddMessage with 200ms timeout, no status → returns error

func TestRedisSync_Rollback(t *testing.T)
// Cross-instance rollback propagation

func TestRedisSync_KeyCleanup(t *testing.T)
// After successful sync, verify Redis keys are cleaned up (no leaks)
```

#### 2.2 Status Deduplication — `src/webhook-in/handler/whatsapp_message_status_test.go`

```go
func TestStatusDedup_SerializeSameWamID(t *testing.T)
// Two goroutines process statuses for the same wamID
// → second waits for first to finish (verified via ordering)

func TestStatusDedup_ParallelDifferentWamIDs(t *testing.T)
// Two goroutines process statuses for different wamIDs
// → both run concurrently (no unnecessary blocking)
```

#### 2.4 Contact Deduplication — `src/campaign/service/send_whatsapp_campaign_test.go`

```go
func TestContactDedup_SamePhone(t *testing.T)
// Two goroutines call GetContactOrSave for the same phone
// → only one DB insert, second gets the existing record

func TestContactDedup_DifferentPhones(t *testing.T)
// Two goroutines call GetContactOrSave for different phones
// → both run concurrently
```

### Manual Tests

```bash
# Verify memory mode still works (no behavior change)
SYNC_BACKEND=memory go run . &
# Send a WhatsApp message via API → check status is linked in DB

# Verify Redis mode
SYNC_BACKEND=redis REDIS_URL=redis://localhost:6379 go run . &
# Send a WhatsApp message via API → check status is linked in DB
```

### Phase 2 Validation

```bash
# Memory implementation tests
go test ./src/message/service/... -v -run "MemorySync"

# Redis implementation tests
REDIS_URL=redis://localhost:6379 go test ./src/message/service/... -v -run "RedisSync"

# All Phase 2 with race detector
REDIS_URL=redis://localhost:6379 go test ./src/message/... ./src/webhook-in/... ./src/campaign/... -v -race
```

---

## Phase 3: WebSocket Cross-Instance Broadcast

### Unit Tests

#### 3.1 Workspace Broadcast — `src/websocket/workspace-manager/main_test.go`

```go
func TestWorkspaceManager_LocalBroadcast(t *testing.T)
// Memory PubSub: add client, broadcast to workspace → client receives message

func TestWorkspaceManager_CrossInstanceBroadcast(t *testing.T)
// Redis PubSub: two WorkspaceChannelManager instances
// Add client to manager A, broadcast from manager B
// → client on A receives the message

func TestWorkspaceManager_SubscribeOnConnect(t *testing.T)
// Add first client for workspace X → PubSub subscription created
// Add second client for workspace X → reuses existing subscription

func TestWorkspaceManager_UnsubscribeOnLastDisconnect(t *testing.T)
// Add client, remove client (last one for workspace)
// → PubSub subscription cleaned up

func TestWorkspaceManager_IsolatedWorkspaces(t *testing.T)
// Client on workspace A, broadcast to workspace B → client receives nothing
```

#### 3.2 Campaign Broadcast — `wacraft-core/src/campaign/model/campaign_channel_test.go`

```go
func TestCampaignChannel_LocalProgress(t *testing.T)
// Memory PubSub: connect client, send progress → client receives update

func TestCampaignChannel_CrossInstanceProgress(t *testing.T)
// Redis PubSub: client on instance A, campaign executes on instance B
// → client on A receives progress updates
```

### Manual Tests

```bash
# Test 1: Cross-instance WebSocket message delivery
# Terminal 1 - Start instance A
SYNC_BACKEND=redis REDIS_URL=redis://localhost:6379 PORT=3000 go run . &

# Terminal 2 - Start instance B
SYNC_BACKEND=redis REDIS_URL=redis://localhost:6379 PORT=3001 go run . &

# Terminal 3 - Connect WebSocket to instance A
websocat ws://localhost:3000/ws/message/new

# Terminal 4 - Send a webhook to instance B (simulating WhatsApp callback)
curl -X POST http://localhost:3001/webhook-in/whatsapp \
  -H "Content-Type: application/json" \
  -d '{"entry":[{"changes":[{"value":{"messages":[...]}}]}]}'

# Expected: WebSocket on instance A receives the new message event

# Test 2: Cross-instance campaign progress
# Connect campaign WebSocket to instance A
websocat ws://localhost:3000/ws/campaign/{id}

# Start campaign via instance B
curl -X POST http://localhost:3001/campaign/{id}/send \
  -H "Authorization: Bearer $TOKEN"

# Expected: WebSocket on instance A receives progress updates
```

### Phase 3 Validation

```bash
# Unit tests
REDIS_URL=redis://localhost:6379 go test ./src/websocket/... ./wacraft-core/src/campaign/... -v

# Race detector
REDIS_URL=redis://localhost:6379 go test ./src/websocket/... -v -race
```

---

## Phase 4: Billing & Caching

### Unit Tests

#### 4.1 Throughput Counter — `src/billing/service/throughput_test.go`

```go
func TestThroughput_MemoryMode(t *testing.T)
// Increment counter 10 times, Get → returns 10

func TestThroughput_RedisMode_CrossInstance(t *testing.T)
// Two counter instances (simulating two app instances)
// Instance A increments 5, Instance B increments 5
// → Get from either returns 10

func TestThroughput_TTLExpiry(t *testing.T)
// Increment with short TTL, wait → counter resets

func TestThroughput_RateLimitEnforced(t *testing.T)
// Set limit to 10, send 15 requests → last 5 rejected (429)
```

#### 4.2 Subscription Cache — `src/billing/service/plan_test.go`

```go
func TestSubscriptionCache_Hit(t *testing.T)
// First call queries DB, second call returns cached → only 1 DB query

func TestSubscriptionCache_CrossInstance(t *testing.T)
// Instance A caches subscription, Instance B reads → cache hit (Redis mode)

func TestSubscriptionCache_TTLRefresh(t *testing.T)
// Cache entry expires after TTL → next call queries DB again

func TestSubscriptionCache_ThunderingHerd(t *testing.T)
// 50 concurrent cache misses for same key → only 1 DB query
```

#### 4.3 Endpoint Weight Cache — `src/billing/service/endpoint_weight_test.go`

```go
func TestEndpointWeightCache_LazyLoad(t *testing.T)
// First call loads from DB, second returns cached

func TestEndpointWeightCache_Invalidate(t *testing.T)
// Load, Invalidate, next Get → reloads from DB

func TestEndpointWeightCache_CrossInstance(t *testing.T)
// Invalidate on instance A → instance B's next Get reloads (Redis mode)
```

### Manual Tests

```bash
# Test rate limiting across instances
# Start two instances with Redis backend

# Use `hey` to send 100 requests split across both instances
hey -n 50 -c 10 -H "Authorization: Bearer $TOKEN" http://localhost:3000/contact &
hey -n 50 -c 10 -H "Authorization: Bearer $TOKEN" http://localhost:3001/contact &

# Expected: Total allowed requests across both instances respects the global limit
# Check X-RateLimit-Remaining headers — should decrease globally
```

### Phase 4 Validation

```bash
# Unit tests
REDIS_URL=redis://localhost:6379 go test ./src/billing/... -v

# Race detector
REDIS_URL=redis://localhost:6379 go test ./src/billing/... -v -race
```

---

## Phase 5: Work Queue (Webhook Delivery)

### Unit Tests — `src/webhook/worker/delivery_worker_test.go`

```go
func TestDeliveryWorker_MemoryMode(t *testing.T)
// Single worker, 5 pending deliveries → all 5 processed

func TestDeliveryWorker_RedisMode_NoDuplicates(t *testing.T)
// Two workers (simulating two instances), 10 pending deliveries
// → each delivery processed exactly once (total = 10, no duplicates)

func TestDeliveryWorker_LockExpiry(t *testing.T)
// Worker A acquires lock, simulates crash (doesn't release)
// → Worker B picks up the delivery after lock TTL expires

func TestDeliveryWorker_GracefulShutdown(t *testing.T)
// Start worker, send shutdown signal → in-flight deliveries complete, worker stops
```

### Manual Tests

```bash
# Test 1: Duplicate prevention
# Start two instances with Redis backend
# Create 5 webhook subscriptions
# Trigger 5 events that generate webhook deliveries
# Check external endpoint logs → each delivery received exactly once

# Test 2: Crash recovery
# Start two instances
# Stop instance A mid-delivery (kill -9)
# Wait for lock TTL to expire
# → Instance B picks up the unfinished deliveries
```

### Phase 5 Validation

```bash
REDIS_URL=redis://localhost:6379 go test ./src/webhook/worker/... -v -race
```

---

## Phase 6: Development Environment

### Docker Compose Tests

```bash
# Test 1: Memory mode (default)
make dev
# Verify: application starts, no Redis connection errors
# Verify: SYNC_BACKEND=memory in startup logs
# Verify: all features work as before

# Test 2: Distributed mode (single instance + Redis)
make dev-distributed
# Verify: Redis container starts
# Verify: application connects to Redis (check startup logs)
# Verify: SYNC_BACKEND=redis in startup logs
# Verify: Redis PING succeeds

# Test 3: Scaled mode (multiple instances + Redis)
make dev-scaled
# Verify: Redis container starts
# Verify: 3 app instances start
# Verify: all instances connect to Redis
```

### Cross-Instance Integration Tests

These are the definitive tests that prove horizontal scaling works. Run with `make dev-scaled`.

```bash
# Test A: Message-Status Sync across instances
# 1. Send WhatsApp message via instance A (port 3000)
curl -X POST http://localhost:3000/message \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Content-Type: application/json" \
  -d '{"to": "5511999999999", "type": "text", "text": {"body": "test"}}'
# 2. WhatsApp sends status webhook → load balancer routes to instance B
# 3. Check DB: message has linked status record

# Test B: WebSocket broadcast across instances
# 1. Connect WebSocket to instance A
# 2. Trigger webhook on instance B
# 3. Verify WebSocket on instance A receives the event

# Test C: Campaign across instances
# 1. Connect campaign WebSocket to instance A
# 2. Start campaign via instance B
# 3. Verify progress updates arrive on instance A's WebSocket
# 4. Cancel campaign via instance A
# 5. Verify campaign stops on instance B

# Test D: Billing counter across instances
# 1. Send requests alternating between instance A and B
# 2. Verify rate limit is enforced globally (not per-instance)

# Test E: Webhook delivery deduplication
# 1. Create webhook subscription
# 2. Trigger events that generate deliveries
# 3. Verify each delivery sent exactly once (not duplicated by multiple workers)
```

### Phase 6 Validation

```bash
# Full test suite in both modes
SYNC_BACKEND=memory go test ./... -v
SYNC_BACKEND=redis REDIS_URL=redis://localhost:6379 go test ./... -v

# Race detector on everything
SYNC_BACKEND=redis REDIS_URL=redis://localhost:6379 go test ./... -v -race
```

---

## Test Summary

| Phase | Unit Tests | Manual/Integration Tests | Files |
|---|---|---|---|
| 1: Core Abstractions | 30 | 4 commands | 10 test files |
| 2: Sync Migration | 14 | 2 scenarios | 4 test files |
| 3: WebSocket Broadcast | 7 | 2 scenarios | 2 test files |
| 4: Billing & Caching | 11 | 1 scenario | 3 test files |
| 5: Work Queue | 4 | 2 scenarios | 1 test file |
| 6: Dev Environment | — | 8 scenarios | — |
| **Total** | **66** | **19** | **20 test files** |

---

## Running All Tests

```bash
# Quick: memory-only unit tests (no Redis needed)
go test ./... -v

# Full: all tests including Redis implementations
REDIS_URL=redis://localhost:6379 go test ./... -v

# Full with race detector
REDIS_URL=redis://localhost:6379 go test ./... -v -race

# Specific phase
REDIS_URL=redis://localhost:6379 go test ./wacraft-core/src/synch/... -v          # Phase 1
REDIS_URL=redis://localhost:6379 go test ./src/message/... ./src/webhook-in/... -v # Phase 2
REDIS_URL=redis://localhost:6379 go test ./src/websocket/... -v                    # Phase 3
REDIS_URL=redis://localhost:6379 go test ./src/billing/... -v                      # Phase 4
REDIS_URL=redis://localhost:6379 go test ./src/webhook/worker/... -v               # Phase 5
```
