# Horizontal Scaling - Implementation Checklist

Track progress per phase. Each phase is independently deployable and testable. Complete and test one phase before moving to the next.

Run tests with:

- `make test` — unit tests only (no Redis needed)
- `make test-redis` — full suite with ephemeral Redis container

---

## Phase 1: Core Abstractions (`wacraft-core`)

### 1.1 Redis Client Infrastructure

- [x] Create `src/config/env/redis.go` - env var loading (`loadRedisEnv()`) following existing project pattern
- [x] Update `src/config/env/main.go` - add `loadRedisEnv()` call in `init()`
- [x] Create `wacraft-core/src/synch/redis/config.go` - configuration struct (receives parsed values from `env` package, no env parsing in core)
- [x] Create `wacraft-core/src/synch/redis/client.go` - Redis client wrapper, connection, health check, key prefixing
- [x] Add `github.com/redis/go-redis/v9` to `wacraft-core/go.mod`

#### 1.1 Tests — `wacraft-core/src/synch/redis/client_test.go`

- [x] `TestNewClient_ParsesValidURL` — creates client with valid `redis://` URL, no error
- [x] `TestNewClient_InvalidURL` — invalid URL returns error
- [x] `TestPrefixKey` — `PrefixKey("lock:abc")` with prefix `"wacraft:"` returns `"wacraft:lock:abc"`
- [x] `TestPrefixKey_EmptyPrefix` — empty prefix returns key unchanged
- [x] `TestConfig_Accessors` — `Config()` and `Redis()` return correct values
- [x] `TestPingWithTimeout_NoRedis` — ping to non-existent Redis returns error within timeout

### 1.2 Distributed Lock

- [x] Create `wacraft-core/src/synch/contract/lock.go` - `DistributedLock[T]` interface
- [x] Create `wacraft-core/src/synch/service/memory-lock.go` - wraps existing `MutexSwapper`
- [x] Create `wacraft-core/src/synch/redis/redis-lock.go` - `SET NX EX` + Lua unlock script

#### 1.2 Memory Lock Tests — `wacraft-core/src/synch/service/memory_lock_test.go`

- [x] `TestMemoryLock_LockUnlock` — lock then unlock, no error or deadlock
- [x] `TestMemoryLock_ConcurrentSameKey` — 100 goroutines increment shared counter under lock → counter == 100
- [x] `TestMemoryLock_DifferentKeysParallel` — lock "a" and "b" concurrently, both acquire immediately
- [x] `TestMemoryLock_SecondLockBlocks` — second `Lock()` blocks until first `Unlock()`
- [x] `TestMemoryLock_IntKey` — lock works with `int` type parameter

#### 1.2 Redis Lock Tests — `wacraft-core/src/synch/redis/redis_lock_integration_test.go`

- [x] `TestRedisLock_LockUnlock` — lock creates key, unlock removes it
- [x] `TestRedisLock_MutualExclusion` — two `RedisLock` instances contend on same key → counter correct
- [x] `TestRedisLock_TTLExpiry` — lock with short TTL expires, second instance acquires
- [x] `TestRedisLock_UnlockOnlyOwner` — non-owner unlock does not release the lock
- [x] `TestRedisLock_ConcurrentHighContention` — 50 goroutines across 5 lock instances → counter == 50

### 1.3 Distributed Pub/Sub

- [x] Create `wacraft-core/src/synch/contract/pubsub.go` - `PubSub` and `Subscription` interfaces
- [x] Create `wacraft-core/src/synch/service/memory-pubsub.go` - in-memory fan-out via Go channels
- [x] Create `wacraft-core/src/synch/redis/redis-pubsub.go` - wraps Redis `PUBLISH`/`SUBSCRIBE`

#### 1.3 Memory Pub/Sub Tests — `wacraft-core/src/synch/service/memory_pubsub_test.go`

- [x] `TestMemoryPubSub_PublishSubscribe` — subscribe, publish, receive message
- [x] `TestMemoryPubSub_MultipleSubscribers` — 3 subscribers all receive the broadcast
- [x] `TestMemoryPubSub_Unsubscribe` — after unsubscribe, channel is closed, publish doesn't panic
- [x] `TestMemoryPubSub_DoubleUnsubscribe` — second unsubscribe is a no-op
- [x] `TestMemoryPubSub_IsolatedChannels` — ch1 subscriber doesn't receive ch2 messages
- [x] `TestMemoryPubSub_BufferFull` — publish to full buffer doesn't panic or block
- [x] `TestMemoryPubSub_ConcurrentPublishSubscribe` — concurrent publish/subscribe/unsubscribe, no race

#### 1.3 Redis Pub/Sub Tests — `wacraft-core/src/synch/redis/redis_pubsub_integration_test.go`

- [x] `TestRedisPubSub_CrossInstance` — subscribe on instance A, publish from instance B → A receives
- [x] `TestRedisPubSub_MultipleChannels` — ch1 message not received by ch2 subscriber
- [x] `TestRedisPubSub_Unsubscribe` — unsubscribe, publish after → no error
- [x] `TestRedisPubSub_MultipleSubscribersSameChannel` — two instances subscribe to same channel, both receive

### 1.4 Distributed Counter

- [x] Create `wacraft-core/src/synch/contract/counter.go` - `DistributedCounter` interface
- [x] Create `wacraft-core/src/synch/service/memory-counter.go` - `sync.Map` based
- [x] Create `wacraft-core/src/synch/redis/redis-counter.go` - `INCRBY` + TTL

#### 1.4 Memory Counter Tests — `wacraft-core/src/synch/service/memory_counter_test.go`

- [x] `TestMemoryCounter_Increment` — increment 3 times → Get == 3
- [x] `TestMemoryCounter_IncrementDelta` — increment by 10, then by 5 → returns 10, then 15
- [x] `TestMemoryCounter_ConcurrentIncrement` — 100 goroutines → Get == 100
- [x] `TestMemoryCounter_TTLExpiry` — set TTL 50ms, wait 60ms → Get returns 0
- [x] `TestMemoryCounter_Delete` — increment, delete → Get returns 0
- [x] `TestMemoryCounter_GetNonExistent` — Get on missing key → 0, no error
- [x] `TestMemoryCounter_SetTTLNonExistent` — SetTTL on missing key → no error
- [x] `TestMemoryCounter_DeleteNonExistent` — Delete missing key → no error
- [x] `TestMemoryCounter_IncrementAfterTTLExpiry` — increment after TTL resets from delta

#### 1.4 Redis Counter Tests — `wacraft-core/src/synch/redis/redis_counter_integration_test.go`

- [x] `TestRedisCounter_CrossInstance` — two instances each increment 5 → Get == 10 from either
- [x] `TestRedisCounter_ConcurrentIncrement` — 100 goroutines → Get == 100
- [x] `TestRedisCounter_TTL` — set TTL 1s, wait 1.1s → Get returns 0
- [x] `TestRedisCounter_Delete` — increment, delete → Get returns 0
- [x] `TestRedisCounter_GetNonExistent` — Get missing key → 0, no error

### 1.5 Distributed Cache

- [x] Create `wacraft-core/src/synch/contract/cache.go` - `DistributedCache` interface
- [x] Create `wacraft-core/src/synch/service/memory-cache.go` - `sync.Map` + TTL tracking
- [x] Create `wacraft-core/src/synch/redis/redis-cache.go` - `GET`/`SET`/`DEL`

#### 1.5 Memory Cache Tests — `wacraft-core/src/synch/service/memory_cache_test.go`

- [x] `TestMemoryCache_SetGet` — set, get → returns data, found
- [x] `TestMemoryCache_Miss` — get nonexistent → nil, not found
- [x] `TestMemoryCache_TTLExpiry` — set with 50ms TTL, wait 60ms → not found
- [x] `TestMemoryCache_Delete` — set, delete, get → not found
- [x] `TestMemoryCache_DeleteNonExistent` — delete missing key → no error
- [x] `TestMemoryCache_Overwrite` — set twice → returns second value
- [x] `TestMemoryCache_Invalidate_TrailingWildcard` — `"prefix:*"` invalidates prefix:a and prefix:b, keeps other:c
- [x] `TestMemoryCache_Invalidate_ExactMatch` — exact pattern doesn't match longer keys
- [x] `TestMemoryCache_Invalidate_EmptyPattern` — empty pattern invalidates nothing

#### 1.5 Redis Cache Tests — `wacraft-core/src/synch/redis/redis_cache_integration_test.go`

- [x] `TestRedisCache_CrossInstance` — set from instance A, get from instance B → found
- [x] `TestRedisCache_TTL` — set with 100ms TTL, wait 150ms → not found
- [x] `TestRedisCache_Delete` — set, delete, get → not found
- [x] `TestRedisCache_Miss` — get nonexistent → nil, not found
- [x] `TestRedisCache_Overwrite` — set twice → returns second value
- [x] `TestRedisCache_Invalidate` — `"group:*"` deletes group:a and group:b, keeps other:c

### 1.6 Backend Factory

- [x] Create `wacraft-core/src/synch/config.go` - `Backend` type (`memory` | `redis`)
- [x] Create `wacraft-core/src/synch/factory.go` - creates correct implementation based on config

#### 1.6 Factory Tests — `wacraft-core/src/synch/factory_test.go`

- [x] `TestFactory_MemoryBackend` — `Backend()` returns `"memory"`, `RedisClient()` returns nil
- [x] `TestFactory_MemoryLock` — creates working lock from memory factory
- [x] `TestFactory_MemoryLockIntKey` — creates `DistributedLock[int]` from memory factory
- [x] `TestFactory_MemoryPubSub` — creates working pub/sub from memory factory
- [x] `TestFactory_MemoryCounter` — creates working counter from memory factory
- [x] `TestFactory_MemoryCache` — creates working cache from memory factory
- [x] `TestFactory_BackendConstants` — `BackendMemory == "memory"`, `BackendRedis == "redis"`

### 1.7 Test Infrastructure

- [x] Create `wacraft-core/src/synch/redis/test_helper_test.go` — shared `testRedisClient(t)` helper with DB 15, auto-flush, cleanup
- [x] Update `Makefile` — add `test` and `test-redis` targets
- [x] Update `.github/workflows/quality-and-security.yml` — add Redis service container, set `REDIS_URL`

### Phase 1 Verification

- [x] All 57 tests pass (25 memory + 7 factory + 6 client + 19 Redis integration)
- [x] Race detector passes on all tests (`-race` flag)
- [x] `wacraft-core/go.mod` compiles cleanly with new dependency
- [x] `go.mod` replace directive in `wacraft-server` resolves correctly

---

## Phase 2: Synchronization Migration (`wacraft-server`)

### 2.0 Central Wiring

- [x] Create `src/synch/main.go` — central factory init, Redis client creation, wires all sync primitives via `init()`
- [x] Update `src/config/main.go` — imports `src/synch` to trigger wiring

### 2.1 Message-Status Synchronization

- [x] Create `src/message/service/message-status-sync-contract.go` - `MessageStatusSync` interface
- [x] Create `src/message/service/message-status-sync-memory.go` - wraps current `MessageStatusSynchronizer`
- [x] Create `src/message/service/message-status-sync-redis.go` - `BLPOP`/`RPUSH` rendezvous
- [x] Refactor `src/message/service/synchronize-message-and-status.go` - global var uses interface, `SetStatusSynchronizer()` setter
- [x] Update `src/message/service/whatsapp.go` - call interface methods instead of concrete type
- [x] Update `src/webhook-in/handler/whatsapp-message-status.go` - call interface `AddStatus`

#### 2.1 Tests — `src/message/service/message_status_sync_test.go`

- [x] `TestMemorySync_MessageThenStatus` — AddMessage blocks, AddStatus unblocks it, MessageSaved returns UUID
- [x] `TestMemorySync_StatusThenMessage` — AddStatus arrives first and blocks, AddMessage unblocks, MessageSaved delivers UUID
- [x] `TestMemorySync_Timeout` — AddMessage with 100ms timeout, no status → timeout error
- [x] `TestMemorySync_Rollback` — AddMessage → AddStatus → RollbackMessage → AddStatus returns rollback sentinel
- [x] `TestMemorySync_ConcurrentDifferentMessages` — 10 wamIDs in parallel, all complete correctly
- [x] `TestRedisSync_CrossInstance` — AddMessage on instance A, AddStatus on instance B → handshake completes
- [x] `TestRedisSync_Timeout` — AddMessage with timeout, no status → error
- [x] `TestRedisSync_Rollback` — cross-instance rollback propagation
- [x] `TestRedisSync_KeyCleanup` — after sync, Redis keys are cleaned up (no leaks)

### 2.2 Status Deduplication

- [x] Update `src/webhook-in/service/synchronize-status.go` - use `DistributedLock[string]` from factory
- [x] Update `src/webhook-in/handler/whatsapp-message-status.go` - replace `MutexSwapper` with `DistributedLock`

#### 2.2 Tests — `src/webhook-in/handler/whatsapp_message_status_test.go`

- [x] `TestStatusDedup_SerializeSameWamID` — two goroutines process same wamID → serialized execution
- [x] `TestStatusDedup_ParallelDifferentWamIDs` — two goroutines with different wamIDs → concurrent execution

### 2.3 Campaign Coordination

- [x] Update `wacraft-core/src/campaign/model/campaign-results.go` - use `DistributedCounter` for `Sent`/`Successes`/`Errors`
- [x] Update `wacraft-core/src/campaign/model/campaign-channel.go` - store `Sending` flag via distributed cache, cancel via PubSub
- [x] Update `wacraft-core/src/campaign/model/channel-pool.go` - integrate with distributed state
- [x] Update `src/campaign/handler/send-whatsapp.go` - use `IsSending()`/`SetSending()`, `SetSendCampaignPool()`
- [x] Update `src/campaign/service/send-whatsapp-campaign.go` - use `IsSending()`/`SetSending()`, `SubscribeCancel()`/`UnsubscribeCancel()`
- [x] Wire `SendCampaignPool` with distributed primitives in `src/synch/main.go`
- [x] Implement cancel via Pub/Sub: publish cancel signal, receiving instance triggers `context.CancelFunc`

#### 2.3 Tests — `src/campaign/service/campaign_coordination_test.go`

- [x] `TestCampaignResults_MemoryCounters` — increment Sent/Successes/Errors, verify counts
- [x] `TestCampaignResults_RedisCounters` — two instances increment counters → aggregated correctly

#### 2.3 Tests — `wacraft-core/src/campaign/model/campaign_channel_test.go`

- [x] `TestCampaignChannel_SetSending_Memory` — set Sending, check IsSending → true
- [x] `TestCampaignChannel_SetSending_Distributed` — set on A, check on B → true (via shared cache)
- [x] `TestCampaignChannel_IsSending_DifferentCampaigns` — campaign A sending, campaign B not
- [x] `TestCampaignChannel_Cancel_Memory` — cancel triggers context cancellation
- [x] `TestCampaignChannel_Cancel_NilCancel` — cancel on nil returns error
- [x] `TestCampaignChannel_Cancel_Distributed` — cancel from A, executing on B → B's context cancelled via PubSub
- [x] `TestCampaignChannel_Cancel_DistributedDifferentCampaigns` — cancel campaign Y, campaign X unaffected
- [x] `TestCampaignChannel_SubscribeCancel_NoopWithoutPubSub` — no-op on memory-only channel
- [x] `TestCampaignChannel_UnsubscribeCancel_StopsListening` — after unsubscribe, cancel signals ignored
- [x] `TestCampaignChannel_ConcurrentSetSending` — 50 goroutines toggle sending, no race

#### 2.3 Tests — `wacraft-core/src/campaign/model/channel_pool_test.go`

- [x] `TestChannelPool_CreateMemory` — creates non-nil pool
- [x] `TestChannelPool_CreateDistributed` — creates pool with cache and pubsub
- [x] `TestChannelPool_AddUser_MemoryChannel` — memory pool creates channel without distributed primitives
- [x] `TestChannelPool_AddUser_DistributedChannel` — distributed pool creates channel with cache and pubsub
- [x] `TestChannelPool_AddUser_SameCampaignReturnsExisting` — same campaign returns same channel with 2 clients
- [x] `TestChannelPool_RemoveUser` — remove one user, channel still exists
- [x] `TestChannelPool_RemoveUser_DeletesEmptyChannel` — remove last user, channel deleted
- [x] `TestChannelPool_RemoveUser_NonExistentCampaign` — no panic
- [x] `TestChannelPool_DistributedSendingCrossChannel` — set sending via pool A, check via pool B → true

### 2.4 Contact Deduplication

- [x] Update `src/campaign/service/send-whatsapp-campaign.go` - replace `contactSynchronizer` with `DistributedLock[string]`

#### 2.4 Tests — `src/campaign/service/contact_dedup_test.go`

- [x] `TestContactDedup_SamePhone` — two goroutines lock same phone → serialized
- [x] `TestContactDedup_DifferentPhones` — two goroutines lock different phones → concurrent

### Phase 2 Verification

- [x] All Phase 2 tests pass with `make test-redis`
- [x] Application starts in memory mode with zero behavior change
- [x] Application starts in Redis mode and all synchronization works
- [x] Existing test suite passes — no regressions

---

## Phase 3: WebSocket Cross-Instance Broadcast

### 3.1 Workspace Message/Status Broadcast

- [x] Update `src/websocket/workspace-manager/main.go` - added `PubSub` field + `SetPubSub()`, `subscribeWorkspace()` on first connect, unsubscribe on last disconnect; `BroadcastToWorkspace` publishes to PubSub (all instances deliver locally via subscriber goroutine); memory-only mode falls back to direct local broadcast
- [x] Update `src/synch/main.go` - wire `NewMessageWorkspaceManager` and `NewStatusWorkspaceManager` with Redis PubSub when `SYNC_BACKEND=redis`
- Note: `wacraft-core/src/websocket/model/channel.go` unchanged — PubSub logic lives in the manager, not the channel primitive

#### 3.1 Tests — `src/websocket/workspace-manager/main_test.go`

- [x] `TestWorkspaceManager_LocalBroadcast` — memory PubSub, broadcast → published to correct PubSub channel
- [x] `TestWorkspaceManager_CrossInstanceBroadcast` — two managers sharing PubSub, broadcast from B → received on A's PubSub channel
- [x] `TestWorkspaceManager_SubscribeOnConnect` — first client creates subscription, second reuses it
- [x] `TestWorkspaceManager_UnsubscribeOnLastDisconnect` — remove last client → subscription and channel cleaned up
- [x] `TestWorkspaceManager_IsolatedWorkspaces` — broadcast to workspace B → workspace A's channel receives nothing

### 3.2 Campaign Real-Time Updates

- [x] Update `wacraft-core/src/campaign/model/campaign-channel.go` - added `BroadcastProgress()` (publishes to `campaign:{id}:progress` PubSub channel or local broadcast in memory mode), `subscribeProgress()` (goroutine forwards PubSub messages to local clients), `UnsubscribeProgress()`; `CreateCampaignChannelWithDistributed` now calls `subscribeProgress()` automatically
- [x] Update `wacraft-core/src/campaign/model/channel-pool.go` - `RemoveUser` calls `UnsubscribeProgress()` and `UnsubscribeCancel()` when last client disconnects
- [x] Update `src/campaign/handler/send-whatsapp.go` - progress callback uses `BroadcastProgress()` instead of `BroadcastJsonMultithread()`

#### 3.2 Tests — `wacraft-core/src/campaign/model/campaign_channel_test.go`

- [x] `TestCampaignChannel_LocalProgress` — memory PubSub, BroadcastProgress → published to correct PubSub channel with correct payload
- [x] `TestCampaignChannel_CrossInstanceProgress` — channel A broadcasts progress → received on shared PubSub channel (simulating instance B)

### Phase 3 Verification

- [x] All Phase 3 tests pass with `make test-redis`
- [x] Connect two WebSocket clients (simulated on different instances)
- [x] Send a message webhook to instance A → both clients receive the new message event
- [x] Send a status webhook to instance B → both clients receive the status update event
- [x] Start campaign on instance A with client on instance B → client receives progress updates

---

## Phase 4: Billing & Caching

### 4.1 Throughput Counter

- [x] Update `src/billing/service/throughput.go` - replace `Counter` with `ThroughputCounter` wrapping `DistributedCounter`; time-bucket keys provide automatic window expiry via TTL (no cleanup goroutine needed)
- [x] Remove cleanup goroutine when using Redis (TTL handles expiry)

#### 4.1 Tests — `src/billing/service/throughput_test.go`

- [x] `TestThroughput_MemoryIncrement` — increment 10 times, Get → 10
- [x] `TestThroughput_RedisCrossInstance` — instance A increments 5, B increments 5 → Get == 10
- [x] `TestThroughput_TTLExpiry` — counter resets after TTL
- [x] `TestThroughput_RateLimitEnforced` — limit 10, send 15 → last 5 rejected

### 4.2 Subscription Cache

- [x] Update `src/billing/service/plan.go` - replace `subscriptionCache` with `DistributedCache` + `DistributedLock`; `queryThroughputFn` injectable for tests

#### 4.2 Tests — `src/billing/service/plan_test.go`

- [x] `TestSubscriptionCache_Hit` — first call queries DB, second returns cached
- [x] `TestSubscriptionCache_CrossInstance` — A caches, B reads → cache hit (Redis)
- [x] `TestSubscriptionCache_TTLRefresh` — entry expires → next call queries DB
- [x] `TestSubscriptionCache_ThunderingHerd` — 50 concurrent misses → only 1 DB query

### 4.3 Endpoint Weight Cache

- [x] Update `src/billing/service/endpoint-weight.go` - replace `sync.RWMutex`+map with `DistributedCache`; `loadWeightsFn` injectable for tests

#### 4.3 Tests — `src/billing/service/endpoint_weight_test.go`

- [x] `TestEndpointWeightCache_LazyLoad` — first call loads from DB, second cached
- [x] `TestEndpointWeightCache_Invalidate` — invalidate → next Get reloads
- [x] `TestEndpointWeightCache_CrossInstance` — invalidate on A → B's next Get reloads (Redis)

### Phase 4 Verification

- [x] All Phase 4 tests pass with `make test-redis`
- [x] Rate limiting enforced globally across instances
- [x] Cache lookups return consistent data across instances
- [x] No performance regression in memory mode

---

## Phase 5: Work Queue (Webhook Delivery)

### 5.1 Webhook Delivery Worker

- [x] Update `src/webhook/worker/delivery-worker.go` - `TryLock` per delivery ID before processing; skip if already locked
- [x] In memory mode: `lock` field is nil — no distributed locking, no behaviour change
- [x] In Redis mode: `TryLock` uses `SET NX EX`; skip delivery if another instance holds the lock
- [x] Added `TryLock(key T) (bool, error)` to `DistributedLock` contract + implemented in `MemoryLock` and `RedisLock`
- [x] `SetDeliveryLock()` setter wired from `src/synch/main.go` when `SYNC_BACKEND=redis`

#### 5.1 Tests — `src/webhook/worker/delivery_worker_test.go`

- [x] `TestDeliveryWorker_MemoryMode` — single worker, 5 deliveries → all 5 processed
- [x] `TestDeliveryWorker_RedisNoDuplicates` — two workers, 10 deliveries → each processed exactly once
- [x] `TestDeliveryWorker_LockExpiry` — worker A crashes (holds lock), B picks up after TTL
- [x] `TestDeliveryWorker_GracefulShutdown` — shutdown signal → in-flight completes, worker stops
- [x] `TestDeliveryWorker_ProcessDeliverySkipsIfLocked` — second worker skips when lock is held
- [x] `TestDeliveryWorker_NilLockProcessesAll` — nil lock (memory mode) → both workers process all

### Phase 5 Verification

- [x] All Phase 5 tests pass with `make test-redis`
- [x] Create multiple pending deliveries, start two workers → no duplicates
- [x] Verify no duplicate external webhook calls

---

## Phase 6: Development Environment & Testing

### 6.1 Docker Compose

- [x] Add `redis` service to `docker-compose.dev.yml` with `distributed` profile
- [x] Add `SYNC_BACKEND` and `REDIS_URL` environment variables to app service
- [x] Verify `docker compose up` starts without Redis (memory mode)
- [x] Verify `docker compose --profile distributed up` starts with Redis

### 6.2 Makefile

- [x] Add `test` target (unit tests, no Redis)
- [x] Add `test-redis` target (full suite with ephemeral Redis container)
- [x] Add `dev-distributed` target — sets `SYNC_BACKEND=redis` automatically, starts Redis profile
- [x] Add `dev-scaled` target — use `make dev-distributed REPLICAS=N`
- [x] Verify each target works end-to-end

### 6.3 CI

- [x] Add Redis service container to GitHub Actions workflow
- [x] Set `REDIS_URL` env var in test step
- [x] Enable `-race` flag in CI test run

### Phase 6 Verification

- [x] `make dev` works (memory mode, no Redis required)
- [x] `make dev-distributed` works (Redis mode, single instance)
- [x] `make dev-scaled` works (Redis mode, 3 instances)
- [x] Manual test: send message on instance A, receive WebSocket update on instance B
- [x] Manual test: start campaign on instance A, see progress on instance B's WebSocket

---

## Final Acceptance Criteria

- [x] Default mode (`SYNC_BACKEND=memory`) is fully backward compatible — zero behavior change
- [x] Redis mode passes all integration tests
- [x] `make test-distributed` passes all tests across both modules with race detector
- [x] Docker Compose supports both profiles with a single command switch
- [x] No new database migrations required
- [x] Documentation in `docs/features/horizontal_scaling/` is complete and up to date
