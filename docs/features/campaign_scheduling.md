# Campaign Scheduling

## Overview

Campaign scheduling lets users prepare a campaign and have it automatically execute at a specified UTC time — no active WebSocket connection required at send time. The feature integrates with both the existing memory and Redis sync backends.

---

## State Machine

```
draft
  │
  ├─ POST /campaign/schedule ──► scheduled ──► running ──► completed
  │                                  │                  └──► failed
  └─ (unchanged)    DELETE /campaign/schedule ◄──────────────┘
                              (only from scheduled / non-running states)
```

| Status      | Meaning                                                                                    |
| ----------- | ------------------------------------------------------------------------------------------ |
| `draft`     | Default. Campaign exists but is not scheduled and not running.                             |
| `scheduled` | A future `scheduled_at` time has been set. The scheduler worker will execute it.           |
| `running`   | The scheduler worker has picked it up and is currently sending messages.                   |
| `completed` | All messages were attempted (may include individual errors, but the run itself completed). |
| `failed`    | An unrecoverable error prevented the run from completing.                                  |
| `cancelled` | The campaign was cancelled mid-run via the WebSocket `cancel` message.                     |

---

## Database Changes

Two columns added to the `campaigns` table (migration `20260404000001_campaign_schedule`):

| Column         | Type                   | Default   | Notes                             |
| -------------- | ---------------------- | --------- | --------------------------------- |
| `status`       | `VARCHAR(20) NOT NULL` | `'draft'` | See state machine above.          |
| `scheduled_at` | `TIMESTAMPTZ`          | `NULL`    | UTC time to execute the campaign. |

Indexes:

- `idx_campaigns_scheduled` — partial on `(scheduled_at, status) WHERE status = 'scheduled'` (fast scheduler poll)
- `idx_campaigns_status` — on `(status, workspace_id)` (fast status filtering per workspace)

---

## Architecture

### Scheduler Worker (`src/campaign/worker/scheduler-worker.go`)

Mirrors the webhook delivery worker pattern (`src/webhook/worker/delivery-worker.go`).

**Poll loop:**

- Reads `CAMPAIGN_SCHEDULE_POLL_INTERVAL` env var (default `30s`).
- Fetches up to 10 campaigns where `status = 'scheduled' AND scheduled_at <= NOW()`.
- Processes up to 5 campaigns concurrently via `errgroup`.

**Per-campaign execution (`processCampaign`):**

1. Acquire distributed lock `campaign_schedule:{id}` (Redis mode only). Skip if not acquired.
2. Atomic status update: `UPDATE … SET status='running' WHERE status='scheduled'`. Skip if 0 rows affected (another instance claimed it).
3. Call `poolGetOrCreate(campaignID)` to register in the global `ChannelPool`, so WebSocket clients can subscribe and receive real-time progress.
4. Call `campaign_service.SendWhatsAppCampaign` with `BroadcastProgress` as the callback — identical to the manual WebSocket-triggered execution.
5. Mark `status='completed'` (success) or `status='failed'` (error).
6. Call `poolRelease(campaignID)` to release the channel pool hold.

### WebSocket Real-Time Integration

The scheduler worker registers the campaign in the global `ChannelPool` before calling `SendWhatsAppCampaign`. A WebSocket client who connects to `/websocket/campaign/whatsapp/send/{campaignID}` during execution calls `AddUser` on the same pool, joining the existing channel. `BroadcastProgress` then reaches all connected clients in real time — exactly the same experience as manually-triggered sends.

**Memory mode:** The `Clients` map in the `CampaignChannel` struct is a Go map (reference type) shared across struct copies. The scheduler's channel and the WebSocket client's channel therefore share the same `Clients` map, so `BroadcastProgress` delivers progress to connected clients.

**Redis mode:** `BroadcastProgress` publishes to `campaign:{id}:progress` pub/sub. Every instance subscribed to that channel (including those with WebSocket clients) delivers the data locally.

### Channel Pool: Hold/Release Pattern (`src/campaign/model/channel-pool.go`)

To prevent `RemoveUser` from cleaning up the channel while the scheduler is still using it, `ChannelPool` now tracks a `workerRefs` count per campaign:

- `GetOrCreateChannel(campaignID)` — creates/returns channel, increments `workerRefs`.
- `ReleaseChannel(campaignID)` — decrements `workerRefs`; deletes channel only when both `len(Clients) == 0` AND `workerRefs == 0`.
- `RemoveUser` checks `workerRefs` before deleting.

### Restart Resilience

Schedule state is stored in PostgreSQL — it survives any number of restarts regardless of backend.

On `SchedulerWorker.Start()`, any campaigns in `running` state are reset to `scheduled`:

```
UPDATE campaigns SET status = 'scheduled' WHERE status = 'running'
```

This re-queues campaigns that were mid-execution when an instance crashed. In Redis mode the distributed lock TTL (`REDIS_LOCK_TTL`, default 30s) auto-expires if the lock holder dies, so the next instance can acquire it.

---

## New API Endpoints

### `POST /campaign/schedule`

Sets `scheduled_at` and transitions status to `scheduled`.

**Request:**

```json
{
    "id": "<campaign UUID>",
    "scheduled_at": "2026-04-05T15:00:00Z"
}
```

**Responses:**

- `200` — updated `Campaign` object
- `400` — invalid body
- `404` — campaign not found or not in your workspace
- `409` — campaign is already `running` or `completed`

**Auth:** `ApiKeyAuth` + `WorkspaceAuth` (requires `campaign.manage` policy)

---

### `DELETE /campaign/schedule`

Cancels a pending schedule, resetting status to `draft`.

**Request:**

```json
{
    "id": "<campaign UUID>"
}
```

**Responses:**

- `200` — updated `Campaign` object
- `400` — invalid body
- `404` — campaign not found or not in your workspace
- `409` — campaign is currently `running`

**Auth:** `ApiKeyAuth` + `WorkspaceAuth` (requires `campaign.manage` policy)

---

## Environment Variables

| Variable                          | Default | Description                                                                                          |
| --------------------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `CAMPAIGN_SCHEDULE_POLL_INTERVAL` | `30s`   | How often the scheduler polls for due campaigns. Accepts Go duration strings (`30s`, `1m`, `2m30s`). |

---

## Testing

Tests are in:

- `src/campaign/worker/scheduler_worker_test.go` — lock logic, graceful shutdown, poll interval
- `src/campaign/handler/schedule_test.go` — HTTP handler: success, conflict states, wrong workspace
- `wacraft-core/src/campaign/model/channel_pool_test.go` — `GetOrCreateChannel`, `ReleaseChannel`, hold/release lifecycle, broadcaster reaches client

Run with:

```sh
make test-memory       # PostgreSQL only (no Redis)
make test-distributed  # PostgreSQL + Redis
```
