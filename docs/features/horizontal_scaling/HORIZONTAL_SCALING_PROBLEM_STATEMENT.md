# Horizontal Scaling - Problem Statement

The application currently relies on in-memory synchronization primitives (mutexes, channels, maps) that only work within a single process. This prevents horizontal scaling because multiple instances cannot share memory state.

---

## Current Architecture (Single Instance)

```
                    ┌─────────────────────────────────────┐
                    │         Single Go Instance           │
                    │                                      │
  WhatsApp ──────►  │  ┌──────────┐    ┌───────────────┐  │  ◄──── WebSocket
  Webhooks          │  │ Webhook  │───▶│ In-Memory Sync │  │        Clients
                    │  │ Handlers │    │  (mutexes,     │  │
  API Calls ─────►  │  │          │    │   channels,    │  │
                    │  │ API      │───▶│   maps)        │  │
                    │  │ Handlers │    └───────────────┘  │
                    │  └──────────┘                        │
                    └─────────────────────────────────────┘
```

Everything works because all goroutines share the same memory space. A message sent via the API and a status webhook received seconds later both access the same `StatusSynchronizer` channel map.

---

## Target Architecture (Multiple Instances)

```
                    ┌────────────────┐   ┌────────────────┐
  WhatsApp ──────►  │   Instance A   │   │   Instance B   │  ◄──── API Calls
  Webhooks          │                │   │                │
                    │  In-Memory A   │   │  In-Memory B   │
                    └───────┬────────┘   └───────┬────────┘
                            │                    │
                            ▼                    ▼
                    ┌────────────────────────────────────┐
                    │     Shared State (Redis / AMQP)    │
                    └────────────────────────────────────┘
```

When multiple instances exist behind a load balancer:
- Instance A may send a WhatsApp message, but Instance B receives the status webhook.
- Instance A hosts a WebSocket client, but Instance B processes the incoming message webhook.
- Instance A starts a campaign, but status updates arrive at Instance B.

---

## Affected Features

### 1. Message <=> Status Synchronization

**Files:**
- `src/message/service/synchronize-message-and-status.go`
- `src/message/service/whatsapp.go`

**Problem:** The `MessageStatusSynchronizer` uses a `map[string]*chan string` to create a rendezvous point between outbound message sends and inbound status webhooks. The protocol is:

1. `AddMessage(wamID)` - Called when sending a message. Opens a channel and blocks.
2. `AddStatus(wamID)` - Called when a status webhook arrives. Signals the channel.
3. `MessageSaved(wamID, messageID)` - Called after DB insert. Sends the message UUID back.

This two-phase channel handshake only works when both operations happen in the same process. If Instance A calls `AddMessage` and Instance B receives the status webhook calling `AddStatus`, the channel does not exist in Instance B's memory.

**Impact:** Messages may be saved without their corresponding status, or statuses may be lost entirely, causing data inconsistency.

---

### 2. Status Deduplication (MutexSwapper)

**Files:**
- `src/webhook-in/handler/whatsapp-message-status.go`
- `src/webhook-in/service/synchronize-status.go`
- `wacraft-core/src/synch/service/mutex-swapper.go`

**Problem:** The `MutexSwapper[string]` serializes status processing per `wamID` to prevent race conditions (e.g., two status webhooks for the same message arriving simultaneously). With multiple instances, two different instances can process statuses for the same `wamID` concurrently since the mutex only exists per-process.

**Impact:** Duplicate status records, race conditions on DB writes, potential constraint violations.

---

### 3. Campaign Execution and Real-Time Updates

**Files:**
- `wacraft-core/src/campaign/model/campaign-channel.go`
- `wacraft-core/src/campaign/model/campaign-results.go`
- `wacraft-core/src/campaign/model/channel-pool.go`
- `src/campaign/service/send-whatsapp-campaign.go`
- `src/campaign/handler/send-whatsapp.go`

**Problem:** Campaigns use multiple in-memory structures:
- `CampaignChannel` - WebSocket channel with `Sending` flag and `cancel` function.
- `CampaignResults` - Atomic counters for `Total`, `Sent`, `Successes`, `Errors`.
- `ChannelPool` - Maps campaign IDs to their WebSocket channels.
- `contactSynchronizer` - MutexSwapper preventing duplicate contact creation.
- `offsetMu` - Mutex serializing DB query offset increments.

With multiple instances:
- A user connected via WebSocket to Instance A won't receive progress updates if Instance B is executing the campaign.
- The `Sending` flag on one instance doesn't prevent another instance from starting the same campaign.
- Campaign cancel requests only reach the instance that has the `context.CancelFunc`.
- Contact deduplication fails across instances.

**Impact:** Users see no real-time updates, campaigns can be started twice, cancel doesn't work, duplicate contacts.

---

### 4. WebSocket Event Propagation

**Files:**
- `wacraft-core/src/websocket/model/channel.go`
- `wacraft-core/src/websocket/model/client-pool.go`
- `wacraft-core/src/websocket/model/client-id-manager.go`
- `src/websocket/workspace-manager/main.go`
- `src/message/handler/new.go` (`NewMessageWorkspaceManager`)
- `src/status/handler/new.go` (`NewStatusWorkspaceManager`)

**Problem:** When a webhook event arrives (new message or status update), the handler broadcasts to all connected WebSocket clients via the workspace `Channel`. But the `Channel` only knows about clients connected to the current instance. Clients connected to other instances receive nothing.

**Impact:** Users miss real-time message and status updates unless they happen to be connected to the exact instance that processes the webhook.

---

### 5. Billing Rate Limiting and Caching

**Files:**
- `src/billing/service/throughput.go`
- `src/billing/service/plan.go`
- `src/billing/service/endpoint-weight.go`

**Problem:**
- `Counter` (throughput) - Uses `sync.Map` + `MutexSwapper` to count API calls per scope. Each instance maintains its own count, so the actual rate is N times the limit (where N is the number of instances).
- `subscriptionCache` - Each instance caches subscription data independently. Cache invalidation on one instance doesn't propagate to others, leading to stale data.
- `endpointWeightCache` - Same issue as subscription cache.

**Impact:** Rate limits are not enforced correctly. Billing data may be inconsistent. Cache staleness can cause incorrect authorization decisions.

---

### 6. Webhook Delivery Worker

**Files:**
- `src/webhook/worker/delivery-worker.go`

**Problem:** The delivery worker polls the database for pending webhook deliveries. With multiple instances, all workers poll simultaneously, potentially picking up the same delivery and sending duplicate outbound webhooks.

**Impact:** Duplicate webhook deliveries to external systems.

---

## Summary of In-Memory State That Must Be Distributed

| Component | Primitive | Current State | Needs |
|---|---|---|---|
| Message-Status Sync | `map[string]*chan string` + `sync.Mutex` | Process-local channel handshake | Distributed pub/sub with request-reply |
| Status Dedup | `MutexSwapper[string]` | Process-local per-key mutex | Distributed lock |
| Campaign Channel | `CampaignChannel` + `ChannelPool` | Process-local WebSocket + state | Distributed pub/sub + shared state |
| Campaign Results | `CampaignResults` | Process-local atomic counters | Distributed counters |
| Campaign Contact Dedup | `MutexSwapper[string]` | Process-local per-key mutex | Distributed lock |
| WebSocket Broadcast | `Channel[T,U,V]` + `WorkspaceChannelManager` | Process-local client map | Cross-instance pub/sub |
| Billing Counter | `sync.Map` + `MutexSwapper` | Process-local counter | Distributed counter (atomic increment) |
| Subscription Cache | `sync.Map` + `MutexSwapper` | Process-local TTL cache | Distributed cache |
| Endpoint Weight Cache | `sync.RWMutex` + `map` | Process-local lazy-load cache | Distributed cache |
| Webhook Worker | DB polling + `errgroup` | Competing consumers | Distributed work queue |
