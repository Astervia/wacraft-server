# Horizontal Scaling - Implementation Plan

This document outlines the step-by-step implementation plan for enabling horizontal scaling. Each phase is designed to be independently deployable and testable.

---

## Phase Overview

```
Phase 1: Core Abstractions (wacraft-core)
   │
   ├── 1.1  Redis client infrastructure
   ├── 1.2  Distributed Lock interface + implementations
   ├── 1.3  Distributed Pub/Sub interface + implementations
   ├── 1.4  Distributed Counter interface + implementations
   ├── 1.5  Distributed Cache interface + implementations
   └── 1.6  Backend factory / registry
   │
Phase 2: Synchronization Migration (wacraft-server)
   │
   ├── 2.1  Message-Status synchronization
   ├── 2.2  Status deduplication (MutexSwapper usages)
   ├── 2.3  Campaign coordination
   └── 2.4  Contact deduplication
   │
Phase 3: WebSocket Cross-Instance Broadcast
   │
   ├── 3.1  Workspace message/status broadcast
   └── 3.2  Campaign real-time updates
   │
Phase 4: Billing & Caching
   │
   ├── 4.1  Throughput counter
   ├── 4.2  Subscription cache
   └── 4.3  Endpoint weight cache
   │
Phase 5: Work Queue (Webhook Delivery)
   │
   └── 5.1  Webhook delivery worker
   │
Phase 6: Development Environment & Testing
   │
   ├── 6.1  Docker Compose profiles
   ├── 6.2  Makefile targets
   └── 6.3  Integration tests
```

---

## Phase 1: Core Abstractions (`wacraft-core`)

All new abstractions go in `wacraft-core` so they are reusable and testable independently.

### 1.1 Redis Client Infrastructure

**New files:**
```
wacraft-core/src/synch/redis/
├── client.go           # Redis client wrapper, connection management
└── config.go           # Configuration struct, env var parsing
```

**What to implement:**
- A `RedisClient` struct wrapping `github.com/redis/go-redis/v9`.
- Configuration parsing from environment variables (`REDIS_URL`, `REDIS_PASSWORD`, etc.).
- Health check method.
- Key prefixing utility (`REDIS_KEY_PREFIX`).
- Graceful shutdown (close connection pool).

**Dependencies to add:**
- `github.com/redis/go-redis/v9` in `wacraft-core/go.mod`.

---

### 1.2 Distributed Lock Interface

**New files:**
```
wacraft-core/src/synch/contract/
├── lock.go             # DistributedLock interface

wacraft-core/src/synch/service/
├── mutex-swapper.go    # (existing) - refactor to implement interface
├── memory-lock.go      # In-memory implementation (wraps MutexSwapper)

wacraft-core/src/synch/redis/
├── redis-lock.go       # Redis implementation
```

**Interface definition:**
```go
// contract/lock.go
package contract

type DistributedLock[T comparable] interface {
    Lock(key T) error
    Unlock(key T) error
}
```

**In-Memory implementation** (`memory-lock.go`):
- Wraps the existing `MutexSwapper[T]`.
- `Lock(key)` calls `MutexSwapper.Lock(key)`, returns `nil`.
- `Unlock(key)` calls `MutexSwapper.Unlock(key)`, returns `nil`.

**Redis implementation** (`redis-lock.go`):
- `Lock(key)` uses `SET key value NX EX ttl` in a retry loop.
- `Unlock(key)` uses a Lua script to atomically check-and-delete (only delete if the value matches the lock owner).
- Lock value includes a unique owner ID (UUID per instance + goroutine ID or random nonce).
- Configurable TTL with automatic renewal (watchdog goroutine) for long-held locks.

**Migration path:**
- All current `MutexSwapper` usages will be replaced with `DistributedLock[T]` interface.
- The 5 instantiation points (see problem statement) will use a factory to pick the implementation.

---

### 1.3 Distributed Pub/Sub Interface

**New files:**
```
wacraft-core/src/synch/contract/
├── pubsub.go           # PubSub interface

wacraft-core/src/synch/service/
├── memory-pubsub.go    # In-memory implementation (Go channels)

wacraft-core/src/synch/redis/
├── redis-pubsub.go     # Redis implementation
```

**Interface definition:**
```go
// contract/pubsub.go
package contract

type PubSub interface {
    Publish(channel string, message []byte) error
    Subscribe(channel string) (Subscription, error)
}

type Subscription interface {
    Channel() <-chan []byte
    Unsubscribe() error
}
```

**In-Memory implementation:**
- Maintains a `map[string][]chan []byte` of subscribers per channel.
- `Publish` fans out to all subscriber channels.
- `Subscribe` creates a buffered channel and appends it to the map.

**Redis implementation:**
- `Publish` calls `PUBLISH channel message`.
- `Subscribe` calls `SUBSCRIBE channel` and wraps the Redis subscription in a `Subscription`.

**Use cases:**
- WebSocket cross-instance broadcast (Phase 3).
- Campaign progress updates (Phase 3).
- Cache invalidation signals (Phase 4).

---

### 1.4 Distributed Counter Interface

**New files:**
```
wacraft-core/src/synch/contract/
├── counter.go          # DistributedCounter interface

wacraft-core/src/synch/service/
├── memory-counter.go   # In-memory implementation

wacraft-core/src/synch/redis/
├── redis-counter.go    # Redis implementation
```

**Interface definition:**
```go
// contract/counter.go
package contract

import "time"

type DistributedCounter interface {
    Increment(key string, delta int64) (int64, error)
    Get(key string) (int64, error)
    SetTTL(key string, ttl time.Duration) error
    Delete(key string) error
}
```

**In-Memory implementation:**
- Wraps current `sync.Map` + mutex pattern from `src/billing/service/throughput.go`.

**Redis implementation:**
- `Increment` uses `INCRBY key delta`.
- `Get` uses `GET key`.
- `SetTTL` uses `EXPIRE key seconds`.
- Atomic increment + TTL set via Lua script for new keys.

---

### 1.5 Distributed Cache Interface

**New files:**
```
wacraft-core/src/synch/contract/
├── cache.go            # DistributedCache interface

wacraft-core/src/synch/service/
├── memory-cache.go     # In-memory implementation

wacraft-core/src/synch/redis/
├── redis-cache.go      # Redis implementation
```

**Interface definition:**
```go
// contract/cache.go
package contract

import "time"

type DistributedCache interface {
    Get(key string) ([]byte, bool, error)
    Set(key string, value []byte, ttl time.Duration) error
    Delete(key string) error
    Invalidate(pattern string) error
}
```

**In-Memory implementation:**
- Wraps `sync.Map` with TTL tracking (similar to current `subscriptionCache`).

**Redis implementation:**
- `Get` uses `GET key`.
- `Set` uses `SET key value EX ttl`.
- `Delete` uses `DEL key`.
- `Invalidate` uses `SCAN` + `DEL` for pattern-based invalidation.

---

### 1.6 Backend Factory / Registry

**New files:**
```
wacraft-core/src/synch/
├── factory.go          # Backend factory, creates implementations based on config
├── config.go           # SyncBackend enum, global config
```

**What to implement:**
```go
// factory.go
package synch

type Backend string

const (
    BackendMemory Backend = "memory"
    BackendRedis  Backend = "redis"
)

type Factory struct {
    backend     Backend
    redisClient *redis.RedisClient  // nil when backend == memory
}

func NewFactory(backend Backend, redisClient *redis.RedisClient) *Factory

func (f *Factory) NewLock[T comparable]() contract.DistributedLock[T]
func (f *Factory) NewPubSub() contract.PubSub
func (f *Factory) NewCounter() contract.DistributedCounter
func (f *Factory) NewCache() contract.DistributedCache
```

The factory instantiates the correct implementation based on the configured backend. This is the single entry point used by the application layer.

> **Note on Go generics limitation:** Since Go does not support generic methods on structs, `NewLock` will need to be a standalone generic function `NewLock[T comparable](f *Factory) contract.DistributedLock[T]` rather than a method.

---

## Phase 2: Synchronization Migration (`wacraft-server`)

### 2.1 Message-Status Synchronization

**Files to modify:**
- `src/message/service/synchronize-message-and-status.go`
- `src/message/service/whatsapp.go`

**Current mechanism:**
```
Instance A (sends message)          Instance A (receives status)
─────────────────────────           ────────────────────────────
AddMessage(wamID)                   AddStatus(wamID)
  creates chan, blocks ◄──────────────signals chan, blocks
  receives signal                     receives messageID
MessageSaved(wamID, msgID)
  sends msgID on chan ────────────►  returns msgID
```

**Distributed mechanism (Redis):**
```
Instance A (sends message)          Instance B (receives status)
─────────────────────────           ────────────────────────────
AddMessage(wamID)                   AddStatus(wamID)
  SUBSCRIBE wacraft:msg:wamID         PUBLISH wacraft:msg:wamID:signal ""
  PUBLISH wacraft:msg:wamID:ready     SUBSCRIBE wacraft:msg:wamID:saved
  waits for signal ◄───────────────── publishes signal
  receives signal                     waits for msgID
MessageSaved(wamID, msgID)
  PUBLISH wacraft:msg:wamID:saved ──► receives msgID
  cleanup                             cleanup
```

**Implementation approach:**
- Define a `MessageStatusSync` interface:
```go
type MessageStatusSync interface {
    AddMessage(wamID string, timeout time.Duration) error
    MessageSaved(wamID string, messageID string, timeout time.Duration)
    RollbackMessage(wamID string, timeout time.Duration)
    AddStatus(wamID string, status string, timeout time.Duration) (string, error)
}
```
- The in-memory implementation wraps the current `MessageStatusSynchronizer`.
- The Redis implementation uses Redis Pub/Sub with correlation keys.
- Replace the global `StatusSynchronizer` singleton with an instance provided via dependency injection or a factory call at init.

**Key design decisions:**
- The Redis Pub/Sub approach requires both sides to be subscribed before the other publishes. Use a "ready" signal pattern: the message side subscribes first, then publishes a "ready" signal. The status side publishes its signal only after seeing "ready" (or uses `BLPOP` on a list as a simpler rendezvous).
- Alternative: Use Redis lists with `BLPOP` for the rendezvous. `AddMessage` does `BLPOP wacraft:msg:wamID:status timeout` (blocks until a status pushes). `AddStatus` does `RPUSH wacraft:msg:wamID:status ""` then `BLPOP wacraft:msg:wamID:saved timeout`. This is simpler and avoids race conditions with Pub/Sub subscription timing.

**Recommended: BLPOP/RPUSH approach.**

```
Instance A (sends message)          Instance B (receives status)
─────────────────────────           ────────────────────────────
AddMessage(wamID)                   AddStatus(wamID)
  BLPOP msg:{wamID}:status 30s       RPUSH msg:{wamID}:status ""
     (blocks) ◄─────────────────────  BLPOP msg:{wamID}:saved 30s
  unblocks, got signal                   (blocks)
MessageSaved(wamID, msgID)
  RPUSH msg:{wamID}:saved msgID ───►  unblocks, got msgID
  DEL msg:{wamID}:status               DEL msg:{wamID}:saved
  DEL msg:{wamID}:saved
```

This mirrors the current channel semantics exactly, with Redis lists acting as single-use channels.

---

### 2.2 Status Deduplication

**Files to modify:**
- `src/webhook-in/handler/whatsapp-message-status.go`
- `src/webhook-in/service/synchronize-status.go`

**Change:** Replace `synch_service.MutexSwapper[string]` with `contract.DistributedLock[string]` from the factory. No logic changes needed beyond swapping the lock implementation.

---

### 2.3 Campaign Coordination

**Files to modify:**
- `wacraft-core/src/campaign/model/campaign-channel.go`
- `wacraft-core/src/campaign/model/campaign-results.go`
- `wacraft-core/src/campaign/model/channel-pool.go`
- `src/campaign/handler/send-whatsapp.go`

**Changes:**

1. **Sending flag** - Store in Redis: `SET campaign:{id}:sending true EX 3600`. Check before starting. Clear on completion.

2. **Campaign results** - Use `DistributedCounter`:
   - `INCRBY campaign:{id}:sent 1`
   - `INCRBY campaign:{id}:successes 1`
   - `INCRBY campaign:{id}:errors 1`
   - Progress callback reads counters and broadcasts via Pub/Sub.

3. **Cancel** - Use Pub/Sub: `PUBLISH campaign:{id}:cancel ""`. The executing instance subscribes to this channel and triggers `context.CancelFunc` on message.

4. **Real-time updates** - Covered in Phase 3.

---

### 2.4 Contact Deduplication

**Files to modify:**
- `src/campaign/service/send-whatsapp-campaign.go`

**Change:** Replace `contactSynchronizer` (`MutexSwapper[string]`) with `DistributedLock[string]`. Same as 2.2.

---

## Phase 3: WebSocket Cross-Instance Broadcast

### 3.1 Workspace Message/Status Broadcast

**Files to modify:**
- `src/websocket/workspace-manager/main.go`
- `wacraft-core/src/websocket/model/channel.go`

**Current flow:**
```
Webhook arrives at Instance A
  → handleMessages() saves to DB
  → WorkspaceChannelManager.BroadcastToWorkspace(workspaceID, data)
  → Channel.BroadcastJsonMultithread(data)
  → Only clients on Instance A receive the event
```

**Distributed flow:**
```
Webhook arrives at Instance A
  → handleMessages() saves to DB
  → PubSub.Publish("workspace:{id}:messages", jsonData)
  → All instances subscribed to "workspace:{id}:messages" receive it
  → Each instance broadcasts to its local WebSocket clients
```

**Implementation:**
- Modify `WorkspaceChannelManager` to accept a `PubSub` interface.
- `BroadcastToWorkspace` publishes to the Pub/Sub channel instead of (or in addition to) local broadcast.
- On startup, each instance subscribes to the relevant workspace channels and forwards messages to local clients.
- Subscribe/unsubscribe dynamically as WebSocket clients connect/disconnect.
- In-memory mode: Pub/Sub publish directly calls local broadcast (no change from current behavior).

---

### 3.2 Campaign Real-Time Updates

**Files to modify:**
- `src/campaign/handler/send-whatsapp.go`
- `wacraft-core/src/campaign/model/campaign-channel.go`

**Implementation:**
- Campaign progress updates are published via `PubSub.Publish("campaign:{id}:progress", data)`.
- Each instance with connected campaign WebSocket clients subscribes to the relevant campaign channel.
- Local `CampaignChannel.BroadcastJsonMultithread` is called when a Pub/Sub message arrives.

---

## Phase 4: Billing & Caching

### 4.1 Throughput Counter

**Files to modify:**
- `src/billing/service/throughput.go`

**Change:** Replace `Counter` with `DistributedCounter`. The Redis `INCRBY` is atomic and handles concurrent increments from multiple instances natively.

The cleanup goroutine (`time.Ticker`) is no longer needed with Redis since keys have native TTL.

---

### 4.2 Subscription Cache

**Files to modify:**
- `src/billing/service/plan.go`

**Change:** Replace `subscriptionCache` with `DistributedCache`. The double-checked locking pattern becomes:
1. `cache.Get(key)` - fast path.
2. `lock.Lock(key)` - thundering herd protection.
3. `cache.Get(key)` - recheck after lock.
4. Query DB, `cache.Set(key, value, ttl)`.
5. `lock.Unlock(key)`.

With Redis, step 1 is a single `GET` and step 4 is `SET EX`. The lock in steps 2-5 uses `DistributedLock`.

---

### 4.3 Endpoint Weight Cache

**Files to modify:**
- `src/billing/service/endpoint-weight.go`

**Change:** Replace `endpointWeightCache` with `DistributedCache`. Simpler than subscription cache since it's a one-time lazy load. With Redis, `GET` returns the cached map (JSON-serialized), and `SET` writes it with a TTL. `InvalidateEndpointWeightCache` calls `cache.Delete(key)`.

---

## Phase 5: Work Queue (Webhook Delivery)

### 5.1 Webhook Delivery Worker

**Files to modify:**
- `src/webhook/worker/delivery-worker.go`

**Current behavior:** Each instance runs a `DeliveryWorker` that polls the DB for pending deliveries. Multiple instances cause duplicate processing.

**Distributed approach (Redis Streams):**
```
Producer:  XADD wacraft:webhooks:pending * delivery_id {id} payload {json}
Consumer:  XREADGROUP GROUP workers consumer-{instance} COUNT 10 BLOCK 5000 STREAMS wacraft:webhooks:pending >
Ack:       XACK wacraft:webhooks:pending workers {message-id}
```

- Each instance joins a consumer group (`workers`).
- Redis ensures each message is delivered to exactly one consumer in the group.
- Failed deliveries are retried via `XPENDING` + `XCLAIM` after a timeout.

**Alternative (simpler, no Redis Streams):**
- Keep DB polling but use a distributed lock per delivery ID:
  ```
  SET webhook:delivery:{id}:lock {instance} NX EX 60
  ```
- If the lock is acquired, process the delivery. Otherwise, skip it.
- This is simpler and doesn't require changing the polling architecture.

**Recommended:** Start with the distributed lock approach (simpler). Migrate to Redis Streams if throughput demands it.

---

## Phase 6: Development Environment & Testing

### 6.1 Docker Compose Profiles

**File to modify:** `docker-compose.dev.yml`

Add a Redis service and use Docker Compose profiles:

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    profiles:
      - distributed
    volumes:
      - redis-data:/data

  app:
    # existing app service
    environment:
      SYNC_BACKEND: ${SYNC_BACKEND:-memory}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}

volumes:
  redis-data:
```

- `docker compose --profile distributed up` starts Redis alongside the app.
- `docker compose up` starts without Redis (in-memory mode).

### 6.2 Makefile Targets

**File to modify:** `Makefile`

```makefile
# Development with in-memory sync (default)
dev:
	docker compose -f docker-compose.dev.yml up

# Development with distributed sync (Redis)
dev-distributed:
	SYNC_BACKEND=redis docker compose -f docker-compose.dev.yml --profile distributed up

# Development with multiple instances (horizontal scaling test)
dev-scaled:
	SYNC_BACKEND=redis docker compose -f docker-compose.dev.yml --profile distributed up --scale app=3
```

### 6.3 Integration Tests

Create integration tests that verify cross-instance scenarios:

```
tests/integration/horizontal_scaling/
├── message_status_sync_test.go    # Send on one instance, status on another
├── websocket_broadcast_test.go    # Connect to A, event on B, receive on A
├── campaign_coordination_test.go  # Start on A, progress on B, cancel from A
├── distributed_lock_test.go       # Concurrent lock from multiple goroutines
└── billing_counter_test.go        # Increment from multiple instances
```

These tests SHOULD use Redis (not mocks) and simulate multi-instance behavior by creating multiple `Factory` instances with the same Redis connection.

---

## Implementation Order and Dependencies

```
                    ┌──────────────────────┐
                    │ 1.1 Redis Client     │
                    └──────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                 ▼
    ┌────────────────┐ ┌─────────────┐ ┌──────────────┐
    │ 1.2 Dist Lock  │ │ 1.3 PubSub  │ │ 1.4 Counter  │
    └───────┬────────┘ └──────┬──────┘ └──────┬───────┘
            │                 │               │
            │           ┌─────┘               │
            ▼           ▼                     ▼
    ┌──────────────────────┐          ┌──────────────┐
    │ 1.5 Cache            │          │ 1.6 Factory  │
    └──────────────────────┘          └──────┬───────┘
                                             │
              ┌──────────────────────────────┤
              ▼                              ▼
    ┌──────────────────────┐    ┌──────────────────────┐
    │ Phase 2: Sync Migr.  │    │ Phase 3: WS Bcast    │
    └──────────┬───────────┘    └──────────┬───────────┘
               │                           │
               ▼                           ▼
    ┌──────────────────────┐    ┌──────────────────────┐
    │ Phase 4: Billing     │    │ Phase 5: Work Queue  │
    └──────────┬───────────┘    └──────────┬───────────┘
               │                           │
               └───────────┬───────────────┘
                           ▼
                ┌──────────────────────┐
                │ Phase 6: Dev & Test  │
                └──────────────────────┘
```

**Critical path:** 1.1 → 1.2 + 1.3 → 1.6 → 2.1 (message-status sync is the most complex and highest-impact item).

**Parallelizable:** Phases 2, 3, 4, and 5 can be worked on concurrently once Phase 1 is complete.
