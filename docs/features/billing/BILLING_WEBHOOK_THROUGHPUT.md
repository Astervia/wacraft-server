# Billing: Webhook Throughput

This document describes how throughput billing applies to webhooks — both **incoming** (Meta/WhatsApp events delivered to the server) and **outgoing** (events dispatched by the server to user-configured URLs).

---

## Incoming Webhooks (`/webhook-in`)

### Overview

WhatsApp/Meta POST events arrive at `/webhook-in/:waba_id`. Each POST request that passes Meta's HMAC signature check is charged against the **workspace** subscription associated with that phone config.

These requests carry no user JWT, so the standard `ThroughputMiddleware` (which requires an authenticated user) does not apply. A dedicated `WebhookInThroughputMiddleware` handles billing for this path.

### Which Requests Are Counted

| Route                       | Auth mechanism                   | Billed |
| --------------------------- | -------------------------------- | ------ |
| `POST /webhook-in/:waba_id` | Meta HMAC signature (per-config) | Yes    |
| `GET  /webhook-in/:waba_id` | Meta verification token          | No     |
| `POST /webhook-in` (legacy) | `META_APP_SECRET` env var        | No     |
| `GET  /webhook-in` (legacy) | `META_VERIFY_TOKEN` env var      | No     |

Only authenticated POST requests on the per-phone-config route are billed. The legacy route (no workspace association) and GET verification challenges are always free.

A request is **not counted** if the phone config has no `workspace_id` set.

### Scope

Always **workspace**. The workspace is resolved from the `PhoneConfig.WorkspaceID` field — the same workspace that owns the WhatsApp integration.

No user-scope fallback: if the workspace is over quota the request is blocked immediately.

### On Limit Exceeded

Returns `429 Too Many Requests`:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1760000042
X-RateLimit-Scope: workspace
X-RateLimit-Scope-ID: aa0e8400-e29b-41d4-a716-446655440005
```

```json
{
    "context": "billing",
    "description": "Throughput limit exceeded: 1000 weighted requests per 60s",
    "message": "Throughput limit exceeded: 1000 weighted requests per 60s"
}
```

Meta will retry the delivery according to its own retry policy. If the workspace subscription is upgraded before Meta's retry window expires, the retried request will succeed.

### Response Headers

The same rate-limit headers as the standard middleware are returned on every billed request:

| Header                  | Description                                           |
| ----------------------- | ----------------------------------------------------- |
| `X-RateLimit-Limit`     | Workspace throughput limit                            |
| `X-RateLimit-Remaining` | Remaining weighted requests in the current window     |
| `X-RateLimit-Reset`     | Unix timestamp when the current window resets         |
| `X-RateLimit-Scope`     | Always `"workspace"`                                  |
| `X-RateLimit-Scope-ID`  | UUID of the workspace                                 |

No headers are emitted when `BILLING_ENABLED=false`.

---

## Outgoing Webhook Delivery

### Overview

When the server dispatches events to user-configured webhook URLs, each delivery attempt is charged against the **workspace** subscription of the webhook. This happens inside the background `DeliveryWorker`, not during the original HTTP request.

### When Throughput Is Consumed

Throughput is charged **immediately before** the HTTP call is made, after the circuit breaker check. The charge represents actual work the server is about to perform.

```
DeliveryWorker.processDelivery(delivery)
    |
    v
Load webhook (if not preloaded)
    |
    v
Circuit breaker check ──open──> skip (no status update)
    |closed
    v
Workspace throughput check ──exceeded──> UpdateDeliveryStatus(failed, "quota exceeded") -> return
    |allowed
    v
Execute HTTP call
```

Weight is always **1** per delivery attempt, regardless of the event type or payload size.

### Behavior When Quota Is Exceeded

Quota-exceeded deliveries are treated as **failed attempts**, not silent skips. This means:

1. `attempt_count` is incremented.
2. Exponential backoff is applied — `next_attempt_at` is set to `now + RetryDelayMs * 2^attempt_count`.
3. `last_error` is set to a human-readable message.
4. Once `attempt_count >= max_attempts`, the delivery moves to `dead_letter`.

This is intentional: workspaces on low-quota plans accumulate delivery failures and eventually dead-letter entries. The dead-letter state and error message are visible in the webhook logs, giving users a clear reason to upgrade.

### Visibility in Webhook Logs

Every quota-exceeded attempt creates a `WebhookLog` entry with:

| Field         | Value                                                              |
| ------------- | ------------------------------------------------------------------ |
| `last_error`  | `"workspace throughput limit exceeded — upgrade your plan to increase quota"` |
| `http_code`   | `0` (no HTTP call was made)                                        |
| `status`      | `attempted` (retrying) or `dead_letter` (exhausted)               |

Users can inspect their webhook delivery logs to see exactly which deliveries were blocked and why.

### Scope

Always **workspace** — resolved from `Webhook.WorkspaceID`. If a webhook has no workspace association (`workspace_id = null`), throughput is not charged and the delivery proceeds unconditionally.

### Retry / Dead-Letter Summary

| Condition                     | `attempt_count` | `status`      | `next_attempt_at`         |
| ----------------------------- | --------------- | ------------- | ------------------------- |
| Quota exceeded, retries left  | +1              | `attempted`   | `now + backoff`           |
| Quota exceeded, no retries    | +1              | `dead_letter` | `null`                    |
| HTTP failure, retries left    | +1              | `attempted`   | `now + backoff`           |
| HTTP success                  | +1              | `succeeded`   | `null`                    |

---

## Interaction Between Incoming and Outgoing

Both incoming (`/webhook-in/:waba_id`) and outgoing delivery share the **same workspace counter**. A workspace's throughput budget is consumed by both directions:

- Each verified POST from Meta to `/webhook-in/:waba_id` costs 1 unit.
- Each outgoing delivery attempt costs 1 unit.

This means a workspace receiving high WhatsApp message volume and dispatching many webhooks will exhaust its shared budget faster. Workspaces with high bidirectional traffic should subscribe to a plan with a proportionally higher limit.
