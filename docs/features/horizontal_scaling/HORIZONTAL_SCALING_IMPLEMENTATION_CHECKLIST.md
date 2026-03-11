# Horizontal Scaling - Implementation Checklist

Track progress per phase. Each phase is independently deployable and testable. Complete and test one phase before moving to the next.

---

## Phase 1: Core Abstractions (`wacraft-core`)

### 1.1 Redis Client Infrastructure

- [ ] Create `src/config/env/redis.go` - env var loading (`loadRedisEnv()`) following existing project pattern
- [ ] Update `src/config/env/main.go` - add `loadRedisEnv()` call in `init()`
- [ ] Create `wacraft-core/src/synch/redis/config.go` - configuration struct (receives parsed values from `env` package, no env parsing in core)
- [ ] Create `wacraft-core/src/synch/redis/client.go` - Redis client wrapper, connection, health check, key prefixing
- [ ] Add `github.com/redis/go-redis/v9` to `wacraft-core/go.mod`
- [ ] Verify: env vars load correctly with defaults when not set
- [ ] Verify: connect to a local Redis, run health check, confirm key prefix works

### 1.2 Distributed Lock

- [ ] Create `wacraft-core/src/synch/contract/lock.go` - `DistributedLock[T]` interface
- [ ] Create `wacraft-core/src/synch/service/memory-lock.go` - wraps existing `MutexSwapper`
- [ ] Create `wacraft-core/src/synch/redis/redis-lock.go` - `SET NX EX` + Lua unlock script
- [ ] Test: memory lock behaves identically to current `MutexSwapper`
- [ ] Test: Redis lock provides mutual exclusion across two goroutines
- [ ] Test: Redis lock auto-expires after TTL (simulates instance crash)
- [ ] Test: Redis lock Lua unlock only releases if owner matches

### 1.3 Distributed Pub/Sub

- [ ] Create `wacraft-core/src/synch/contract/pubsub.go` - `PubSub` and `Subscription` interfaces
- [ ] Create `wacraft-core/src/synch/service/memory-pubsub.go` - in-memory fan-out via Go channels
- [ ] Create `wacraft-core/src/synch/redis/redis-pubsub.go` - wraps Redis `PUBLISH`/`SUBSCRIBE`
- [ ] Test: memory pub/sub delivers to multiple subscribers
- [ ] Test: Redis pub/sub delivers across two separate `PubSub` instances (simulates two app instances)
- [ ] Test: unsubscribe stops delivery

### 1.4 Distributed Counter

- [ ] Create `wacraft-core/src/synch/contract/counter.go` - `DistributedCounter` interface
- [ ] Create `wacraft-core/src/synch/service/memory-counter.go` - `sync.Map` based
- [ ] Create `wacraft-core/src/synch/redis/redis-counter.go` - `INCRBY` + TTL
- [ ] Test: memory counter increments correctly under concurrent access
- [ ] Test: Redis counter aggregates increments from two clients
- [ ] Test: Redis counter keys expire after TTL

### 1.5 Distributed Cache

- [ ] Create `wacraft-core/src/synch/contract/cache.go` - `DistributedCache` interface
- [ ] Create `wacraft-core/src/synch/service/memory-cache.go` - `sync.Map` + TTL tracking
- [ ] Create `wacraft-core/src/synch/redis/redis-cache.go` - `GET`/`SET`/`DEL`
- [ ] Test: memory cache set/get/delete/TTL expiry
- [ ] Test: Redis cache is shared across two clients
- [ ] Test: Redis cache entries expire after TTL

### 1.6 Backend Factory

- [ ] Create `wacraft-core/src/synch/config.go` - `Backend` type (`memory` | `redis`)
- [ ] Create `wacraft-core/src/synch/factory.go` - creates correct implementation based on config
- [ ] Test: factory returns memory implementations when `backend=memory`
- [ ] Test: factory returns Redis implementations when `backend=redis`
- [ ] Test: factory panics/errors clearly if `backend=redis` but no Redis client provided

### Phase 1 Verification

- [ ] All unit tests pass for both memory and Redis implementations
- [ ] `wacraft-core/go.mod` compiles cleanly with new dependency
- [ ] `go.mod` replace directive in `wacraft-server` still resolves correctly

---

## Phase 2: Synchronization Migration (`wacraft-server`)

### 2.1 Message-Status Synchronization

- [ ] Create `src/message/service/message-status-sync-contract.go` - `MessageStatusSync` interface
- [ ] Create `src/message/service/message-status-sync-memory.go` - wraps current `MessageStatusSynchronizer`
- [ ] Create `src/message/service/message-status-sync-redis.go` - `BLPOP`/`RPUSH` rendezvous
- [ ] Refactor `src/message/service/synchronize-message-and-status.go` - global var uses interface
- [ ] Update `src/message/service/whatsapp.go` - call interface methods instead of concrete type
- [ ] Update `src/webhook-in/handler/whatsapp-message-status.go` - call interface `AddStatus`
- [ ] Test memory mode: send message → receive status webhook → verify status linked to message
- [ ] Test Redis mode: `AddMessage` on goroutine A, `AddStatus` on goroutine B (separate factory instances) → verify handshake completes
- [ ] Test Redis mode: `RollbackMessage` propagates correctly
- [ ] Test Redis mode: timeout fires if no status arrives
- [ ] Test: switch `SYNC_BACKEND` env var and confirm correct implementation is used

### 2.2 Status Deduplication

- [ ] Update `src/webhook-in/service/synchronize-status.go` - use `DistributedLock[string]` from factory
- [ ] Update `src/webhook-in/handler/whatsapp-message-status.go` - replace `MutexSwapper` with `DistributedLock`
- [ ] Test memory mode: two concurrent status webhooks for same `wamID` are serialized
- [ ] Test Redis mode: same test, but locks acquired from different factory instances

### 2.3 Campaign Coordination

- [ ] Update `wacraft-core/src/campaign/model/campaign-results.go` - use `DistributedCounter` for `Sent`/`Successes`/`Errors`
- [ ] Update `wacraft-core/src/campaign/model/campaign-channel.go` - store `Sending` flag via distributed cache/lock
- [ ] Update `wacraft-core/src/campaign/model/channel-pool.go` - integrate with distributed state
- [ ] Update `src/campaign/handler/send-whatsapp.go` - use factory-provided primitives
- [ ] Implement cancel via Pub/Sub: publish cancel signal, receiving instance triggers `context.CancelFunc`
- [ ] Test memory mode: campaign start/progress/cancel works as before
- [ ] Test Redis mode: `Sending` flag visible across instances
- [ ] Test Redis mode: cancel signal reaches executing instance
- [ ] Test Redis mode: counters aggregate correctly across instances

### 2.4 Contact Deduplication

- [ ] Update `src/campaign/service/send-whatsapp-campaign.go` - replace `contactSynchronizer` with `DistributedLock[string]`
- [ ] Test memory mode: concurrent sends to same contact are serialized
- [ ] Test Redis mode: same test across factory instances

### Phase 2 Verification

- [ ] Application starts in memory mode with zero behavior change
- [ ] Application starts in Redis mode and all synchronization works across goroutines simulating separate instances
- [ ] Run existing test suite - no regressions

---

## Phase 3: WebSocket Cross-Instance Broadcast

### 3.1 Workspace Message/Status Broadcast

- [ ] Update `wacraft-core/src/websocket/model/channel.go` - add optional `PubSub`, publish on broadcast, subscribe for remote events
- [ ] Update `src/websocket/workspace-manager/main.go` - accept `PubSub`, wire subscriptions on channel create/destroy
- [ ] Update `src/message/handler/new.go` - pass `PubSub` to `NewMessageWorkspaceManager`
- [ ] Update `src/status/handler/new.go` - pass `PubSub` to `NewStatusWorkspaceManager`
- [ ] Test memory mode: broadcast reaches local clients (no change)
- [ ] Test Redis mode: publish on instance A, WebSocket client on instance B receives the event
- [ ] Test: dynamic subscribe/unsubscribe as WebSocket clients connect/disconnect

### 3.2 Campaign Real-Time Updates

- [ ] Update `wacraft-core/src/campaign/model/campaign-channel.go` - broadcast progress via `PubSub`
- [ ] Update `wacraft-core/src/campaign/model/channel-pool.go` - wire `PubSub` on channel create
- [ ] Update `src/campaign/handler/send-whatsapp.go` - pass `PubSub` to pool
- [ ] Test memory mode: campaign progress reaches connected client
- [ ] Test Redis mode: campaign executes on instance A, client on instance B receives progress

### Phase 3 Verification

- [ ] Connect two WebSocket clients (simulated on different instances)
- [ ] Send a message webhook to instance A → both clients receive the new message event
- [ ] Send a status webhook to instance B → both clients receive the status update event
- [ ] Start campaign on instance A with client on instance B → client receives progress updates

---

## Phase 4: Billing & Caching

### 4.1 Throughput Counter

- [ ] Update `src/billing/service/throughput.go` - replace `Counter` with `DistributedCounter`
- [ ] Remove cleanup goroutine when using Redis (TTL handles expiry)
- [ ] Test memory mode: rate limiting works as before
- [ ] Test Redis mode: increments from two instances aggregate correctly
- [ ] Test Redis mode: counter keys expire after TTL

### 4.2 Subscription Cache

- [ ] Update `src/billing/service/plan.go` - replace `subscriptionCache` with `DistributedCache` + `DistributedLock`
- [ ] Test memory mode: cache hit/miss/TTL expiry works as before
- [ ] Test Redis mode: cache is shared across instances
- [ ] Test Redis mode: thundering herd protection (only one DB query under concurrent cache miss)

### 4.3 Endpoint Weight Cache

- [ ] Update `src/billing/service/endpoint-weight.go` - replace with `DistributedCache`
- [ ] Test memory mode: lazy-load + invalidation works as before
- [ ] Test Redis mode: invalidation on instance A clears cache for instance B

### Phase 4 Verification

- [ ] Rate limiting enforced globally across instances
- [ ] Cache lookups return consistent data across instances
- [ ] No performance regression in memory mode

---

## Phase 5: Work Queue (Webhook Delivery)

### 5.1 Webhook Delivery Worker

- [ ] Update `src/webhook/worker/delivery-worker.go` - acquire distributed lock per delivery ID before processing
- [ ] In memory mode: no change (single instance, no contention)
- [ ] In Redis mode: `SET NX EX` lock per delivery, skip if already locked
- [ ] Test memory mode: deliveries processed normally
- [ ] Test Redis mode: two workers polling simultaneously → each delivery processed exactly once
- [ ] Test Redis mode: lock expires if worker crashes (delivery retried by another worker)

### Phase 5 Verification

- [ ] Create multiple pending webhook deliveries
- [ ] Start two simulated workers
- [ ] Verify each delivery is processed exactly once
- [ ] Verify no duplicate external webhook calls

---

## Phase 6: Development Environment & Testing

### 6.1 Docker Compose

- [ ] Add `redis` service to `docker-compose.dev.yml` with `distributed` profile
- [ ] Add `SYNC_BACKEND` and `REDIS_URL` environment variables to app service
- [ ] Verify `docker compose up` starts without Redis (memory mode)
- [ ] Verify `docker compose --profile distributed up` starts with Redis

### 6.2 Makefile

- [ ] Add `dev` target (memory mode, current behavior)
- [ ] Add `dev-distributed` target (Redis mode, single instance)
- [ ] Add `dev-scaled` target (Redis mode, multiple instances via `--scale`)
- [ ] Verify each target works end-to-end

### 6.3 Integration Tests

- [ ] Create `tests/integration/horizontal_scaling/message_status_sync_test.go`
- [ ] Create `tests/integration/horizontal_scaling/websocket_broadcast_test.go`
- [ ] Create `tests/integration/horizontal_scaling/campaign_coordination_test.go`
- [ ] Create `tests/integration/horizontal_scaling/distributed_lock_test.go`
- [ ] Create `tests/integration/horizontal_scaling/billing_counter_test.go`
- [ ] All integration tests pass against a live Redis instance

### Phase 6 Verification

- [ ] `make dev` works (memory mode, no Redis required)
- [ ] `make dev-distributed` works (Redis mode, single instance)
- [ ] `make dev-scaled` works (Redis mode, 3 instances)
- [ ] Manual test: send message on instance A, receive WebSocket update on instance B
- [ ] Manual test: start campaign on instance A, see progress on instance B's WebSocket

---

## Final Acceptance Criteria

- [ ] Default mode (`SYNC_BACKEND=memory`) is fully backward compatible - zero behavior change
- [ ] Redis mode passes all integration tests
- [ ] Existing test suite passes in both modes
- [ ] Docker Compose supports both profiles with a single command switch
- [ ] No new database migrations required
- [ ] Documentation in `docs/features/horizontal_scaling/` is complete and up to date
