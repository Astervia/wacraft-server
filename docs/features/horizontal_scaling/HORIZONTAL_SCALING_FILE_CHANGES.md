# Horizontal Scaling - File Changes Reference

Detailed map of every file that needs to be created or modified, organized by phase.

---

## Phase 1: Core Abstractions (`wacraft-core`)

### New Files

| File                                               | Description                                                         |
| -------------------------------------------------- | ------------------------------------------------------------------- |
| `wacraft-core/src/synch/contract/lock.go`          | `DistributedLock[T]` interface                                      |
| `wacraft-core/src/synch/contract/pubsub.go`        | `PubSub` and `Subscription` interfaces                              |
| `wacraft-core/src/synch/contract/counter.go`       | `DistributedCounter` interface                                      |
| `wacraft-core/src/synch/contract/cache.go`         | `DistributedCache` interface                                        |
| `wacraft-core/src/synch/service/memory-lock.go`    | In-memory `DistributedLock` (wraps `MutexSwapper`)                  |
| `wacraft-core/src/synch/service/memory-pubsub.go`  | In-memory `PubSub` (Go channels)                                    |
| `wacraft-core/src/synch/service/memory-counter.go` | In-memory `DistributedCounter` (`sync.Map`)                         |
| `wacraft-core/src/synch/service/memory-cache.go`   | In-memory `DistributedCache` (`sync.Map` + TTL)                     |
| `wacraft-core/src/synch/redis/client.go`           | Redis client wrapper and connection management                      |
| `wacraft-core/src/synch/redis/config.go`           | Redis configuration struct (receives parsed values, no env parsing) |
| `wacraft-core/src/synch/redis/redis-lock.go`       | Redis `DistributedLock` (`SET NX EX` + Lua unlock)                  |
| `wacraft-core/src/synch/redis/redis-pubsub.go`     | Redis `PubSub` (`PUBLISH`/`SUBSCRIBE`)                              |
| `wacraft-core/src/synch/redis/redis-counter.go`    | Redis `DistributedCounter` (`INCRBY` + TTL)                         |
| `wacraft-core/src/synch/redis/redis-cache.go`      | Redis `DistributedCache` (`GET`/`SET`/`DEL`)                        |
| `wacraft-core/src/synch/factory.go`                | Backend factory: creates correct implementation based on config     |
| `wacraft-core/src/synch/config.go`                 | `Backend` type, global configuration                                |

### New Files (`wacraft-server`)

| File                      | Description                                                                                    |
| ------------------------- | ---------------------------------------------------------------------------------------------- |
| `src/config/env/redis.go` | Env var loading for all Redis/sync vars (`loadRedisEnv()`), following existing project pattern |

### Modified Files

| File                                              | Change                                                      |
| ------------------------------------------------- | ----------------------------------------------------------- |
| `wacraft-core/go.mod`                             | Add `github.com/redis/go-redis/v9` dependency               |
| `wacraft-core/src/synch/service/mutex-swapper.go` | No breaking changes; still used by `memory-lock.go` wrapper |
| `src/config/env/main.go`                          | Add `loadRedisEnv()` call in `init()`                       |

---

## Phase 2: Synchronization Migration (`wacraft-server`)

### New Files

| File                                                  | Description                                                          |
| ----------------------------------------------------- | -------------------------------------------------------------------- |
| `src/message/service/message-status-sync-contract.go` | `MessageStatusSync` interface                                        |
| `src/message/service/message-status-sync-memory.go`   | In-memory implementation (wraps current `MessageStatusSynchronizer`) |
| `src/message/service/message-status-sync-redis.go`    | Redis implementation (`BLPOP`/`RPUSH` rendezvous)                    |

### Modified Files

| File                                                    | Change                                                                                                                                                                          |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/message/service/synchronize-message-and-status.go` | Refactor: extract current logic into `MemoryMessageStatusSync`. Replace global `StatusSynchronizer` with interface variable initialized by factory.                             |
| `src/message/service/whatsapp.go`                       | Replace `StatusSynchronizer.AddMessage(...)` with `messageStatusSync.AddMessage(...)` (interface call instead of concrete type). Same for `MessageSaved` and `RollbackMessage`. |
| `src/webhook-in/handler/whatsapp-message-status.go`     | Replace `synch_service.MutexSwapper[string]` with `contract.DistributedLock[string]`. Replace `message_service.StatusSynchronizer.AddStatus(...)` with interface call.          |
| `src/webhook-in/service/synchronize-status.go`          | Replace `CreateStatusSynchronizer()` with factory-based lock creation.                                                                                                          |
| `src/campaign/service/send-whatsapp-campaign.go`        | Replace `contactSynchronizer` (`MutexSwapper[string]`) with `DistributedLock[string]` from factory.                                                                             |

---

## Phase 3: WebSocket Cross-Instance Broadcast

### Modified Files

| File                                                  | Change                                                                                                                                                                                                    |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `wacraft-core/src/websocket/model/channel.go`         | Add optional `PubSub` field. When set, `BroadcastJsonMultithread` also publishes to the Pub/Sub channel. Add `SubscribeRemote()` method that listens for remote broadcasts and forwards to local clients. |
| `src/websocket/workspace-manager/main.go`             | Accept `PubSub` in constructor. Wire up remote subscriptions when channels are created. Unsubscribe when channels are removed.                                                                            |
| `src/message/handler/new.go`                          | Pass `PubSub` to `NewMessageWorkspaceManager`.                                                                                                                                                            |
| `src/status/handler/new.go`                           | Pass `PubSub` to `NewStatusWorkspaceManager`.                                                                                                                                                             |
| `wacraft-core/src/campaign/model/campaign-channel.go` | Add `PubSub` support for cross-instance campaign progress broadcasting.                                                                                                                                   |
| `wacraft-core/src/campaign/model/channel-pool.go`     | Wire Pub/Sub when creating channels.                                                                                                                                                                      |
| `src/campaign/handler/send-whatsapp.go`               | Pass `PubSub` to campaign pool creation.                                                                                                                                                                  |

---

## Phase 4: Billing & Caching

### Modified Files

| File                                     | Change                                                                                                                   |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `src/billing/service/throughput.go`      | Replace `Counter` internals with `DistributedCounter`. Remove cleanup goroutine when using Redis (keys have native TTL). |
| `src/billing/service/plan.go`            | Replace `subscriptionCache` with `DistributedCache` + `DistributedLock` for thundering herd protection.                  |
| `src/billing/service/endpoint-weight.go` | Replace `endpointWeightCache` with `DistributedCache`.                                                                   |

---

## Phase 5: Work Queue

### Modified Files

| File                                    | Change                                                                                                                                                                           |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/webhook/worker/delivery-worker.go` | Add distributed lock per delivery ID before processing. In memory mode, continue with current behavior. In Redis mode, acquire `SET NX EX` lock before processing each delivery. |

---

## Phase 6: Development Environment

### Modified Files

| File                     | Change                                                                                                      |
| ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `docker-compose.dev.yml` | Add `redis` service with `distributed` profile. Add `SYNC_BACKEND` and `REDIS_URL` env vars to app service. |
| `Makefile`               | Add `dev-distributed` and `dev-scaled` targets.                                                             |
| `go.mod`                 | Ensure `wacraft-core` replace directive covers new dependencies. Update if needed.                          |

### New Files

| File                                                                 | Description                                        |
| -------------------------------------------------------------------- | -------------------------------------------------- |
| `tests/integration/horizontal_scaling/message_status_sync_test.go`   | Cross-instance message-status synchronization test |
| `tests/integration/horizontal_scaling/websocket_broadcast_test.go`   | Cross-instance WebSocket broadcast test            |
| `tests/integration/horizontal_scaling/campaign_coordination_test.go` | Cross-instance campaign lifecycle test             |
| `tests/integration/horizontal_scaling/distributed_lock_test.go`      | Distributed lock correctness test                  |
| `tests/integration/horizontal_scaling/billing_counter_test.go`       | Distributed counter accuracy test                  |

---

## Summary

| Category                             | New Files | Modified Files |
| ------------------------------------ | --------- | -------------- |
| Phase 1: Core (`wacraft-core` + env) | 17        | 3              |
| Phase 2: Sync Migration              | 3         | 5              |
| Phase 3: WebSocket Broadcast         | 0         | 7              |
| Phase 4: Billing & Caching           | 0         | 3              |
| Phase 5: Work Queue                  | 0         | 1              |
| Phase 6: Dev Environment             | 5         | 3              |
| **Total**                            | **25**    | **22**         |

---

## Dependency Graph (Files)

```
wacraft-core/src/synch/contract/lock.go
  ├── used by: wacraft-core/src/synch/service/memory-lock.go
  ├── used by: wacraft-core/src/synch/redis/redis-lock.go
  ├── used by: src/webhook-in/handler/whatsapp-message-status.go
  ├── used by: src/webhook-in/service/synchronize-status.go
  ├── used by: src/campaign/service/send-whatsapp-campaign.go
  ├── used by: src/billing/service/plan.go
  └── used by: src/webhook/worker/delivery-worker.go

wacraft-core/src/synch/contract/pubsub.go
  ├── used by: wacraft-core/src/synch/service/memory-pubsub.go
  ├── used by: wacraft-core/src/synch/redis/redis-pubsub.go
  ├── used by: wacraft-core/src/websocket/model/channel.go
  ├── used by: src/websocket/workspace-manager/main.go
  └── used by: wacraft-core/src/campaign/model/campaign-channel.go

wacraft-core/src/synch/contract/counter.go
  ├── used by: wacraft-core/src/synch/service/memory-counter.go
  ├── used by: wacraft-core/src/synch/redis/redis-counter.go
  └── used by: src/billing/service/throughput.go

wacraft-core/src/synch/contract/cache.go
  ├── used by: wacraft-core/src/synch/service/memory-cache.go
  ├── used by: wacraft-core/src/synch/redis/redis-cache.go
  ├── used by: src/billing/service/plan.go
  └── used by: src/billing/service/endpoint-weight.go
```
