# Horizontal Scaling - Requirements

This document defines the functional and non-functional requirements for enabling horizontal scaling of the wacraft-server application.

---

## Functional Requirements

### FR-1: Dual-Mode Operation (Memory / External)

The system MUST support two operational modes:

1. **In-Memory Mode** (default) - Current behavior. All synchronization uses local Go primitives. No external dependencies beyond the database. Suitable for single-instance deployments or clients with limited budget.

2. **External Mode** - Synchronization uses Redis and/or RabbitMQ. Required for multi-instance deployments. Requires additional infrastructure.

The mode MUST be configurable via environment variables at startup. The application MUST NOT require code changes to switch between modes.

```
SYNC_BACKEND=memory   # default, current behavior
SYNC_BACKEND=redis    # use Redis for distributed state
```

### FR-2: Interface-Based Abstraction

All synchronization components MUST be defined as Go interfaces. Each interface MUST have at least two implementations:
- An in-memory implementation (wrapping current behavior)
- A distributed implementation (using Redis/RabbitMQ)

This applies to:
- Distributed locks (replacing `MutexSwapper`)
- Pub/Sub messaging (replacing channel-based handshakes and WebSocket broadcast)
- Distributed counters (replacing `sync.Map`-based counters)
- Distributed cache (replacing `sync.Map`-based caches)
- Work queue (replacing competing DB pollers)

### FR-3: Message-Status Synchronization

The distributed implementation MUST preserve the current semantics:
- When a message is sent, the system waits for a status webhook before saving.
- When a status arrives before the message is saved, the system waits for the message to be saved.
- Timeout behavior is preserved.
- Rollback signaling works across instances.

### FR-4: WebSocket Cross-Instance Broadcast

When a webhook event (message or status) is processed by any instance, ALL connected WebSocket clients across ALL instances MUST receive the update, provided they are subscribed to the relevant workspace.

### FR-5: Campaign Distributed Coordination

- Campaign execution state (`Sending` flag) MUST be visible across instances.
- Campaign progress counters (`Sent`, `Successes`, `Errors`) MUST be accurate across instances.
- Campaign cancel requests MUST reach the instance executing the campaign.
- Real-time WebSocket updates MUST reach clients on any instance.

### FR-6: Distributed Locking

All `MutexSwapper` usages MUST be replaceable with distributed locks that provide:
- Per-key mutual exclusion across instances.
- Automatic lock expiry (to handle instance crashes).
- The same semantics as the current `Lock`/`Unlock` API.

### FR-7: Billing Counter Accuracy

Throughput counters MUST aggregate across all instances. Rate limits MUST be enforced globally, not per-instance.

### FR-8: Cache Consistency

Caches (subscription, endpoint weight) MUST be shared or invalidated across instances to prevent stale reads.

### FR-9: Webhook Delivery Deduplication

The webhook delivery worker MUST ensure each pending delivery is processed by exactly one instance (at-least-once with dedup, or exactly-once via distributed work queue).

---

## Non-Functional Requirements

### NFR-1: Zero Downtime Switching

Switching between memory and external mode MUST only require a restart with different environment variables. No database migration or data transformation should be needed.

### NFR-2: Graceful Degradation

If the external backend (Redis/RabbitMQ) becomes temporarily unavailable:
- The system SHOULD log errors clearly.
- The system SHOULD NOT crash.
- It is acceptable for synchronization to degrade (e.g., missed WebSocket broadcasts, temporary lock failures) as long as data integrity is preserved at the database level.

### NFR-3: Minimal Latency Overhead

The distributed implementations SHOULD add no more than ~5ms of latency per synchronization operation under normal conditions. Redis operations are typically sub-millisecond on local networks.

### NFR-4: Development Environment Parity

The `docker-compose.dev.yml` MUST support both modes:
- A minimal profile without Redis/RabbitMQ for in-memory development.
- A full profile with Redis/RabbitMQ for distributed testing.

Switching between profiles SHOULD be a single command (e.g., `make dev` vs `make dev-distributed`).

### NFR-5: Testing

- Unit tests MUST cover both in-memory and distributed implementations.
- Integration tests SHOULD verify cross-instance scenarios (e.g., message on instance A, status on instance B).

### NFR-6: Backward Compatibility

- The default mode (`SYNC_BACKEND=memory`) MUST be fully backward compatible.
- No existing API contracts, database schemas, or client behavior should change.
- Existing single-instance deployments MUST work without any configuration changes.

---

## Technology Choice

### Redis (Recommended Primary Backend)

Redis is the recommended backend for the distributed implementations because it provides all the required primitives in a single dependency:

| Requirement | Redis Primitive |
|---|---|
| Distributed Lock | `SET NX EX` (Redlock for multi-node) or Redisson-style |
| Pub/Sub | `PUBLISH` / `SUBSCRIBE` |
| Request-Reply | Pub/Sub with correlation IDs or `BLPOP`-based queues |
| Distributed Counter | `INCR` / `INCRBY` with TTL |
| Distributed Cache | `GET` / `SET` / `DEL` with TTL |
| Work Queue | `BRPOPLPUSH` or Redis Streams (`XADD` / `XREADGROUP`) |

**Advantages:**
- Single additional dependency covers all use cases.
- Sub-millisecond latency.
- Well-supported Go client libraries (`go-redis/redis`).
- Simple to operate (single node is sufficient for most deployments).
- Lightweight resource footprint.

### RabbitMQ (Optional, for Work Queues)

RabbitMQ MAY be used for the webhook delivery worker if more robust message delivery guarantees are needed (acknowledgments, dead-letter queues, retries). However, Redis Streams can cover this use case adequately for most deployments.

**Decision:** Start with Redis-only. Add RabbitMQ as a future option if Redis Streams prove insufficient for the webhook worker.

---

## Environment Variables

| Variable | Type | Default | Description |
|---|---|---|---|
| `SYNC_BACKEND` | `string` | `memory` | Synchronization backend: `memory` or `redis` |
| `REDIS_URL` | `string` | `redis://localhost:6379` | Redis connection URL (only used when `SYNC_BACKEND=redis`) |
| `REDIS_PASSWORD` | `string` | _(empty)_ | Redis password (optional) |
| `REDIS_DB` | `int` | `0` | Redis database number |
| `REDIS_KEY_PREFIX` | `string` | `wacraft:` | Prefix for all Redis keys (namespace isolation) |
| `REDIS_LOCK_TTL` | `duration` | `30s` | Default TTL for distributed locks |
| `REDIS_CACHE_TTL` | `duration` | `5m` | Default TTL for cache entries |
