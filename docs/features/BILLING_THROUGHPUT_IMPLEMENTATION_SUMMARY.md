# Billing by Throughput - Implementation Summary

This document describes the throughput-based billing system. Plans define weighted request limits over configurable time windows. Users and workspaces can have multiple active plans (limits stack). A default free plan is always available as fallback.

## Configuration

### Environment Variables

| Variable                       | Required | Default | Description                                                                                                        |
| ------------------------------ | -------- | ------- | ------------------------------------------------------------------------------------------------------------------ |
| `BILLING_ENABLED`              | No       | `false` | Master toggle. When `false`, throughput middleware is a no-op and no enforcement happens. Set to `true` to enable. |
| `STRIPE_SECRET_KEY`            | No       | -       | Stripe API secret key. Required for checkout flow.                                                                 |
| `STRIPE_WEBHOOK_SECRET`        | No       | -       | Stripe webhook signing secret. Required to receive payment confirmations.                                          |
| `DEFAULT_FREE_PLAN_THROUGHPUT` | No       | `100`   | Fallback free plan throughput limit (weighted requests per window). Used if no default plan exists in DB.          |
| `DEFAULT_FREE_PLAN_WINDOW`     | No       | `60`    | Fallback free plan window duration in seconds.                                                                     |

### Billing Toggle Behavior

When `BILLING_ENABLED=false` (the default):

- The throughput middleware is registered but immediately returns `c.Next()` — zero overhead
- Billing API routes still work (admins can manage plans, subscriptions, etc. ahead of time)
- No counters are allocated, no DB queries for subscriptions, no rate limit headers
- The Stripe provider is not initialized

When `BILLING_ENABLED=true`:

- Throughput enforcement is active on all authenticated requests
- Rate limit headers are added to responses
- Requests exceeding limits receive `429 Too Many Requests`
- Fallback routes (billing, workspace list, user profile) remain accessible under a separate budget using the default free plan limits

---

## Core Concepts

### Plans

A plan defines a throughput allowance. Plans live in the `plans` table and are managed via the API.

| Field              | Type    | Description                                                                   |
| ------------------ | ------- | ----------------------------------------------------------------------------- |
| `id`               | uuid    | Unique identifier                                                             |
| `name`             | string  | Display name (e.g. "Starter", "Pro")                                          |
| `slug`             | string  | URL-safe unique identifier (e.g. "starter", "pro")                            |
| `description`      | string? | Optional description                                                          |
| `throughput_limit` | int     | Weighted requests per window. **<= 0 means unlimited (infinite throughput).** |
| `window_seconds`   | int     | Time window in seconds (e.g. 60 = per minute)                                 |
| `duration_days`    | int     | Plan validity in days after activation                                        |
| `price_cents`      | int64   | Price in smallest currency unit (e.g. cents). 0 = free.                       |
| `currency`         | string  | Currency code (e.g. "usd")                                                    |
| `is_default`       | bool    | Whether this is the fallback free plan                                        |
| `is_custom`        | bool    | Admin-created plan for specific users/workspaces                              |
| `active`           | bool    | Whether the plan is available for purchase                                    |

#### Unlimited Plans

Set `throughput_limit` to `0` (or any value `<= 0`) to create an unlimited plan. When a scope has any active subscription with an unlimited plan:

- No request counting is performed (zero overhead)
- No rate limit headers are sent
- The usage endpoint returns `"unlimited": true` and `"remaining": -1`

### Subscriptions

A subscription is an active instance of a plan, scoped to either a user or a workspace.

| Field                 | Type      | Description                                                                       |
| --------------------- | --------- | --------------------------------------------------------------------------------- |
| `id`                  | uuid      | Unique identifier                                                                 |
| `plan_id`             | uuid      | FK to the plan                                                                    |
| `scope`               | enum      | `"user"` or `"workspace"`                                                         |
| `user_id`             | uuid      | Who purchased/owns this subscription                                              |
| `workspace_id`        | uuid?     | Set when `scope=workspace`                                                        |
| `throughput_override` | int?      | Admin override. Overrides plan's `throughput_limit`. Set to `<= 0` for unlimited. |
| `starts_at`           | datetime  | When the subscription becomes active                                              |
| `expires_at`          | datetime  | When it expires                                                                   |
| `cancelled_at`        | datetime? | When cancelled (null if active)                                                   |
| `payment_provider`    | string    | `"stripe"`, `"manual"`, etc.                                                      |
| `payment_external_id` | string?   | Provider-specific ID                                                              |

#### Scope Behavior

- **`scope=user`**: The throughput limit applies to ALL API requests made by that user across every workspace they belong to.
- **`scope=workspace`**: The throughput limit applies only to requests made within that specific workspace (identified by `X-Workspace-ID` header).

Both scopes are checked independently. A request can be blocked by either scope.

#### Stacking

Multiple active subscriptions on the same scope stack their limits additively. For example, a user with two active plans of 1000 req/min each gets 2000 req/min total. If any subscription is unlimited, the entire scope becomes unlimited.

#### Expiration

When all paid subscriptions expire, the scope automatically falls back to the default free plan. No service interruption — just lower limits.

### Endpoint Weights

Every API endpoint has a weight (default: 1). Admins can configure custom weights to make expensive endpoints cost more.

| Field          | Type    | Description                              |
| -------------- | ------- | ---------------------------------------- |
| `id`           | uuid    | Unique identifier                        |
| `method`       | string  | HTTP method (GET, POST, etc.)            |
| `path_pattern` | string  | Route path (e.g. "/message", "/contact") |
| `weight`       | int     | Cost per request (default: 1)            |
| `description`  | string? | Optional description                     |

---

## API Reference

### Plans

#### List Plans - `GET /billing/plan`

Returns all plans. Available to any authenticated user.

**Auth**: `Authorization: Bearer <token>`

**Query Parameters**: Standard pagination (`limit`, `offset`) and date ordering (`created_at`, `updated_at`).

**Example Response**:

```json
[
    {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Free",
        "slug": "free",
        "throughput_limit": 100,
        "window_seconds": 60,
        "duration_days": 36500,
        "price_cents": 0,
        "currency": "usd",
        "is_default": true,
        "is_custom": false,
        "active": true,
        "created_at": "2026-02-10T12:00:00Z",
        "updated_at": "2026-02-10T12:00:00Z"
    },
    {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "Pro",
        "slug": "pro",
        "throughput_limit": 5000,
        "window_seconds": 60,
        "duration_days": 30,
        "price_cents": 4900,
        "currency": "usd",
        "is_default": false,
        "is_custom": false,
        "active": true,
        "created_at": "2026-02-10T12:00:00Z",
        "updated_at": "2026-02-10T12:00:00Z"
    },
    {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "name": "Enterprise",
        "slug": "enterprise",
        "description": "Unlimited throughput for large teams",
        "throughput_limit": 0,
        "window_seconds": 60,
        "duration_days": 30,
        "price_cents": 29900,
        "currency": "usd",
        "is_default": false,
        "is_custom": false,
        "active": true,
        "created_at": "2026-02-10T12:00:00Z",
        "updated_at": "2026-02-10T12:00:00Z"
    }
]
```

---

#### Create Plan - `POST /billing/plan`

Creates a new billing plan. Admin only.

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

**Request Body**:

| Field              | Type   | Required | Description                                       |
| ------------------ | ------ | -------- | ------------------------------------------------- |
| `name`             | string | Yes      | Plan display name                                 |
| `slug`             | string | Yes      | URL-safe unique identifier                        |
| `description`      | string | No       | Optional description                              |
| `throughput_limit` | int    | Yes      | Weighted requests per window. `<= 0` = unlimited. |
| `window_seconds`   | int    | Yes      | Time window in seconds                            |
| `duration_days`    | int    | Yes      | Plan duration in days                             |
| `price_cents`      | int64  | Yes      | Price in smallest currency unit                   |
| `currency`         | string | Yes      | Currency code                                     |
| `is_default`       | bool   | No       | Fallback free plan flag                           |
| `is_custom`        | bool   | No       | Custom plan flag                                  |
| `active`           | bool   | No       | Available for purchase                            |

**Example - Standard Plan**:

```json
{
    "name": "Pro",
    "slug": "pro",
    "throughput_limit": 5000,
    "window_seconds": 60,
    "duration_days": 30,
    "price_cents": 4900,
    "currency": "usd",
    "active": true
}
```

**Example - Unlimited Plan**:

```json
{
    "name": "Enterprise",
    "slug": "enterprise",
    "description": "Unlimited throughput",
    "throughput_limit": 0,
    "window_seconds": 60,
    "duration_days": 30,
    "price_cents": 29900,
    "currency": "usd",
    "active": true
}
```

---

#### Update Plan - `PUT /billing/plan?id=<uuid>`

Updates an existing plan. Admin only.

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

**Example Request**:

```json
{
    "throughput_limit": 10000,
    "price_cents": 7900
}
```

---

#### Delete Plan - `DELETE /billing/plan?id=<uuid>`

Deletes a plan. Admin only. Plans with active subscriptions cannot be deleted (FK constraint).

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

---

### Subscriptions

#### List Subscriptions - `GET /billing/subscription`

Returns subscriptions for the authenticated user. If `X-Workspace-ID` is provided, filters by that workspace.

**Auth**: `Authorization: Bearer <token>`
**Optional**: `X-Workspace-ID: <uuid>`

**Example Response**:

```json
[
    {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "plan_id": "660e8400-e29b-41d4-a716-446655440001",
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "starts_at": "2026-02-10T12:00:00Z",
        "expires_at": "2026-03-12T12:00:00Z",
        "payment_provider": "stripe",
        "payment_external_id": "cs_test_abc123",
        "plan": {
            "id": "660e8400-e29b-41d4-a716-446655440001",
            "name": "Pro",
            "slug": "pro",
            "throughput_limit": 5000,
            "window_seconds": 60
        },
        "created_at": "2026-02-10T12:00:00Z",
        "updated_at": "2026-02-10T12:00:00Z"
    }
]
```

---

#### Checkout - `POST /billing/subscription/checkout`

Initiates a Stripe checkout session for purchasing a plan.

**Auth**: `Authorization: Bearer <token>`

**Request Body**:

| Field          | Type   | Required    | Description                           |
| -------------- | ------ | ----------- | ------------------------------------- |
| `plan_id`      | uuid   | Yes         | Plan to purchase                      |
| `scope`        | string | Yes         | `"user"` or `"workspace"`             |
| `workspace_id` | uuid   | Conditional | Required when `scope=workspace`       |
| `success_url`  | string | Yes         | Redirect URL after successful payment |
| `cancel_url`   | string | Yes         | Redirect URL if payment is cancelled  |

**Example Request**:

```json
{
    "plan_id": "660e8400-e29b-41d4-a716-446655440001",
    "scope": "user",
    "success_url": "https://app.example.com/billing/success",
    "cancel_url": "https://app.example.com/billing/cancel"
}
```

**Example Response**:

```json
{
    "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_abc123...",
    "external_id": "cs_test_abc123"
}
```

The frontend should redirect the user to `checkout_url`. After payment, Stripe redirects to `success_url`. The subscription is activated asynchronously via the Stripe webhook.

---

#### Create Manual Subscription - `POST /billing/subscription/manual`

Admin creates a subscription manually (for custom plans, enterprise deals, etc.).

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

**Request Body**:

| Field                 | Type   | Required    | Description                                             |
| --------------------- | ------ | ----------- | ------------------------------------------------------- |
| `plan_id`             | uuid   | Yes         | Plan to assign                                          |
| `scope`               | string | Yes         | `"user"` or `"workspace"`                               |
| `user_id`             | uuid   | Yes         | User to assign the subscription to                      |
| `workspace_id`        | uuid   | Conditional | Required when `scope=workspace`                         |
| `throughput_override` | int    | No          | Override plan's throughput limit. `<= 0` for unlimited. |

**Example - Custom unlimited subscription for a workspace**:

```json
{
    "plan_id": "660e8400-e29b-41d4-a716-446655440001",
    "scope": "workspace",
    "user_id": "990e8400-e29b-41d4-a716-446655440004",
    "workspace_id": "aa0e8400-e29b-41d4-a716-446655440005",
    "throughput_override": 0
}
```

---

#### Cancel Subscription - `DELETE /billing/subscription?id=<uuid>`

Cancels an active subscription. Users can only cancel their own subscriptions.

**Auth**: `Authorization: Bearer <token>`

---

### Usage

#### Get Usage - `GET /billing/usage`

Returns current throughput usage for the authenticated user. If `X-Workspace-ID` is provided, also returns workspace-scoped usage. When a scope's limit is exceeded, an additional entry with `"fallback": true` is included showing the separate budget available for billing/upgrade routes.

**Auth**: `Authorization: Bearer <token>`
**Optional**: `X-Workspace-ID: <uuid>` — when provided, the workspace is resolved via `OptionalWorkspaceMiddleware` (validates membership) and workspace-scoped usage is included in the response.

**Response Fields**:

| Field              | Type   | Description                                                          |
| ------------------ | ------ | -------------------------------------------------------------------- |
| `scope`            | string | `"user"` or `"workspace"`                                           |
| `user_id`          | uuid?  | Present when `scope=user`                                            |
| `workspace_id`     | uuid?  | Present when `scope=workspace`                                       |
| `unlimited`        | bool   | `true` if scope has infinite throughput                               |
| `throughput_limit`  | int    | Weighted requests per window (0 when unlimited)                      |
| `window_seconds`   | int    | Window duration in seconds                                           |
| `current_usage`    | int64  | Weighted requests used in current window                             |
| `remaining`        | int64  | Requests remaining (-1 when unlimited)                               |
| `fallback`         | bool   | `true` if this entry represents the fallback budget for billing routes |

**Example Response (Limited)**:

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 5000,
        "window_seconds": 60,
        "current_usage": 342,
        "remaining": 4658,
        "fallback": false
    },
    {
        "scope": "workspace",
        "workspace_id": "aa0e8400-e29b-41d4-a716-446655440005",
        "unlimited": false,
        "throughput_limit": 1000,
        "window_seconds": 60,
        "current_usage": 127,
        "remaining": 873,
        "fallback": false
    }
]
```

**Example Response (Limit Exceeded — includes fallback budget)**:

When the main subscription limit is exceeded, an additional entry with `"fallback": true` is included for each exceeded scope. The fallback entry uses a separate counter and the default free plan limits, matching the budget enforced by the throughput middleware on fallback routes.

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 5000,
        "window_seconds": 60,
        "current_usage": 5200,
        "remaining": 0,
        "fallback": false
    },
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 13,
        "remaining": 87,
        "fallback": true
    },
    {
        "scope": "workspace",
        "workspace_id": "aa0e8400-e29b-41d4-a716-446655440005",
        "unlimited": false,
        "throughput_limit": 20,
        "window_seconds": 600,
        "current_usage": 25,
        "remaining": 0,
        "fallback": false
    },
    {
        "scope": "workspace",
        "workspace_id": "aa0e8400-e29b-41d4-a716-446655440005",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 3,
        "remaining": 97,
        "fallback": true
    }
]
```

**Example Response (Unlimited)**:

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": true,
        "throughput_limit": 0,
        "window_seconds": 0,
        "current_usage": 0,
        "remaining": -1,
        "fallback": false
    }
]
```

---

### Endpoint Weights (Admin)

#### List Endpoint Weights - `GET /billing/endpoint-weight`

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

#### Create Endpoint Weight - `POST /billing/endpoint-weight`

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

**Example Request**:

```json
{
    "method": "POST",
    "path_pattern": "/message",
    "weight": 5,
    "description": "Sending messages costs 5x"
}
```

#### Delete Endpoint Weight - `DELETE /billing/endpoint-weight?id=<uuid>`

Removes a custom weight. The endpoint reverts to the default weight of 1.

**Auth**: `Authorization: Bearer <token>`, `X-Workspace-ID: <uuid>`
**Policy**: `billing.admin`

---

### Payment Webhooks

#### Stripe Webhook - `POST /billing/webhook/stripe`

Receives Stripe webhook events. No authentication — Stripe validates via the `Stripe-Signature` header.

Handled events:

- `checkout.session.completed` — Activates the subscription automatically.

---

## Response Headers

When billing is enabled and the user has limited throughput, responses include rate limit headers:

| Header                            | Description                              |
| --------------------------------- | ---------------------------------------- |
| `X-RateLimit-Limit`               | User-scoped throughput limit             |
| `X-RateLimit-Remaining`           | Remaining requests in current window     |
| `X-RateLimit-Reset`               | Unix timestamp when the window resets    |
| `X-RateLimit-Limit-Workspace`     | Workspace-scoped limit (when applicable) |
| `X-RateLimit-Remaining-Workspace` | Workspace remaining (when applicable)    |

When a limit is exceeded, the response includes:

| Header        | Description                     |
| ------------- | ------------------------------- |
| `Retry-After` | Seconds until the window resets |

**No headers are sent** when billing is disabled.

When a scope is unlimited, headers are still sent to indicate unlimited throughput:

| Header                            | Value |
| --------------------------------- | ----- |
| `X-RateLimit-Limit`               | `0`   |
| `X-RateLimit-Remaining`           | `-1`  |
| `X-RateLimit-Reset`               | `0`   |
| `X-RateLimit-Limit-Workspace`     | `0`   |
| `X-RateLimit-Remaining-Workspace` | `-1`  |

---

## Throughput Exceeded Response

**Status**: `429 Too Many Requests`

```json
{
    "context": "billing",
    "description": "Throughput limit exceeded: 1000 weighted requests per 60s",
    "message": "Throughput limit exceeded: 1000 weighted requests per 60s"
}
```

---

## Fallback Routes

When a user or workspace exceeds their subscription's throughput limit, most routes return `429 Too Many Requests`. However, certain routes remain accessible under a **separate fallback budget** equal to the default free plan limits. This ensures users can always browse plans, manage subscriptions, and upgrade — even when rate-limited.

### Fallback Route Table

| Method | Path Prefix              | Purpose                    |
| ------ | ------------------------ | -------------------------- |
| `*`    | `/billing/plan`          | Browse available plans     |
| `*`    | `/billing/subscription`  | Manage subscriptions       |
| `GET`  | `/billing/usage`         | Check current usage        |
| `GET`  | `/workspace`             | List/view workspaces       |
| `GET`  | `/user/me`               | Get current user profile   |

### How It Works

1. The throughput middleware increments the **main counter** (`user:<uuid>` or `workspace:<uuid>`) as usual.
2. If the main counter exceeds the subscription limit and the request matches a fallback route, a **separate fallback counter** (`user-fallback:<uuid>` or `workspace-fallback:<uuid>`) is checked against the default free plan limits.
3. If the fallback counter is within limits, the request proceeds. Rate limit headers reflect the **fallback budget**.
4. If both the main and fallback budgets are exhausted, the response is `429`.
5. The fallback budget is tracked independently — regular API usage cannot exhaust it.

### Usage Endpoint Awareness

The `GET /billing/usage` endpoint returns both the subscription usage and the fallback budget when a scope's limit is exceeded. Entries with `"fallback": true` represent the fallback budget. This allows the frontend to show "your plan limit is exceeded" alongside "you can still manage your subscription."

---

## Policies

Three new workspace policies control access to billing features:

| Policy           | Description                                                             |
| ---------------- | ----------------------------------------------------------------------- |
| `billing.read`   | View plans, own subscriptions, usage                                    |
| `billing.manage` | Purchase plans, cancel subscriptions                                    |
| `billing.admin`  | Create/edit plans, manage endpoint weights, create manual subscriptions |

The `workspace.admin` policy implicitly grants access to all billing operations.

---

## Payment Provider Architecture

The billing system uses a `PaymentProvider` interface, making it easy to swap or add payment processors:

```go
type Provider interface {
    Name() string
    CreateCheckoutSession(plan, userID, scope, workspaceID, successURL, cancelURL) (checkoutURL, externalID, error)
    CancelSubscription(externalID string) error
    VerifyWebhookSignature(payload, signature) error
    ParseWebhookEvent(payload, signature) (WebhookEvent, error)
}
```

Currently implemented: **Stripe**. To add another provider (e.g. PayPal), implement this interface and register it.

---

## Database Schema

New tables created automatically via GORM AutoMigrate:

- `plans` — Plan catalog
- `subscriptions` — Active plan instances
- `endpoint_weights` — Per-endpoint weight configuration
- `usage_logs` — Historical usage snapshots (for analytics)

A goose migration seeds the default free plan on first startup.

---

## Frontend UI Recommendations

### Billing Page Sections

#### 1. Current Plan & Usage

- Display active subscriptions with plan name, limits, and expiration
- Show usage gauge/progress bar: `current_usage / throughput_limit`
- For unlimited plans, show an "Unlimited" badge instead of a gauge
- Show both user-scoped and workspace-scoped usage if applicable
- Call `GET /billing/usage` (optionally with `X-Workspace-ID`) to populate

#### 2. Available Plans

- List all active plans from `GET /billing/plan`
- Display as pricing cards with: name, description, throughput limit, window, duration, price
- For plans with `throughput_limit <= 0`, show "Unlimited" instead of a number
- Highlight the current plan
- "Subscribe" button triggers `POST /billing/subscription/checkout`
- Redirect user to `checkout_url` from the response

#### 3. Subscription History

- List all subscriptions from `GET /billing/subscription`
- Show: plan name, scope, start/end dates, status (active/expired/cancelled)
- "Cancel" button calls `DELETE /billing/subscription?id=<uuid>`
- Show payment provider badge (Stripe, Manual)

#### 4. Scope Selector

- When purchasing a plan, let the user choose scope:
    - **User scope**: "Apply to all my workspaces"
    - **Workspace scope**: "Apply to this workspace only" (show workspace picker)
- For workspace scope, the user must select which workspace to bind

#### 5. Admin Panel (billing.admin policy)

- Plan management: create, edit, toggle active, delete
- Manual subscription creation for custom deals
- Endpoint weight configuration table
- Option to create unlimited plans (toggle or input `0` for throughput limit)

### Checkout Flow

1. User clicks "Subscribe" on a plan card
2. Frontend calls `POST /billing/subscription/checkout` with plan ID, scope, and redirect URLs
3. Frontend redirects to `checkout_url` (Stripe hosted page)
4. After payment, Stripe redirects to `success_url`
5. Stripe sends webhook to `POST /billing/webhook/stripe`
6. Backend activates subscription
7. Frontend can poll `GET /billing/subscription` to confirm activation

### Usage Display

- Poll `GET /billing/usage` (with `X-Workspace-ID` if applicable) periodically or on page load
- Show percentage bar for limited plans
- Show "Unlimited" badge for unlimited plans
- Color coding: green (< 70%), yellow (70-90%), red (> 90%)
- When `remaining` reaches 0, show a warning banner with upgrade CTA
- When entries with `"fallback": true` appear in the response, the user has exceeded their main limit:
    - Show the exceeded subscription usage (the `fallback: false` entry with `remaining: 0`)
    - Show a notice that billing routes are still accessible, with the fallback budget remaining
    - Prompt the user to upgrade their plan

---

## Backwards Compatibility

- Billing is disabled by default (`BILLING_ENABLED=false`) — existing deployments are unaffected
- When enabled, all users fall back to the default free plan if they have no subscriptions
- Billing routes are always registered regardless of the toggle (admins can set up plans before enabling)
- No changes to existing API request/response formats for non-billing endpoints (only new headers are added)
