# Throughput Spending & Response Headers

This document describes how the throughput middleware decides which scope to charge per request and the headers it returns.

## Scope Priority

Only **one scope is charged per request**. The middleware uses a cascade: it tries workspace first, and falls back to user if the workspace budget is exhausted or absent.

| Priority | Condition                                                     | Scope charged | Counter key                |
| -------- | ------------------------------------------------------------- | ------------- | -------------------------- |
| 1        | `X-Workspace-ID` provided, workspace exists, budget available | `workspace`   | `workspace:<workspace-id>` |
| 2        | No workspace, or workspace budget exceeded                    | `user`        | `user:<user-id>`           |
| 3        | User budget also exceeded + fallback route                    | `user`        | `user-fallback:<user-id>`  |

### How the cascade works

1. If a workspace is in context, the middleware resolves the workspace subscription and increments the workspace counter.
2. If the workspace counter is **within its limit**, the request is charged to the workspace scope and processing continues.
3. If the workspace counter **exceeds its limit** (or no workspace is present), the middleware falls through to the user scope, resolves the user subscription, and increments the user counter.
4. If the user counter is also exceeded and the request targets a **fallback route**, a separate fallback counter is used with the default free plan limits.
5. If all budgets are exhausted, the response is `429`.

This means a request inside a workspace with a low limit (e.g. 20/600s) will cascade to the user's own subscription (e.g. 100/60s) once the workspace budget runs out, instead of immediately returning 429.

## Fallback Routes

When the user scope's limit is exceeded, most routes return `429`. However, certain routes remain accessible under a **separate fallback budget** using the default free plan limits. This ensures users can always browse plans, manage subscriptions, and upgrade.

| Method | Path Prefix             | Purpose                  |
| ------ | ----------------------- | ------------------------ |
| `*`    | `/billing/plan`         | Browse available plans   |
| `*`    | `/billing/subscription` | Manage subscriptions     |
| `GET`  | `/billing/usage`        | Check current usage      |
| `GET`  | `/workspace`            | List/view workspaces     |
| `GET`  | `/user/me`              | Get current user profile |

The fallback counter key uses a `-fallback` suffix (e.g. `user-fallback:<user-id>`). This counter is tracked independently from the main counter, so regular API usage cannot exhaust the fallback budget.

## Response Headers

Every response from the throughput middleware includes headers that describe the charged scope and its limits. **No headers are sent** when `BILLING_ENABLED=false`.

### Rate Limit Headers

| Header                  | Description                                       |
| ----------------------- | ------------------------------------------------- |
| `X-RateLimit-Limit`     | Throughput limit for the charged scope            |
| `X-RateLimit-Remaining` | Remaining weighted requests in the current window |
| `X-RateLimit-Reset`     | Unix timestamp when the current window resets     |

### Scope Identification Headers

| Header                 | Description                                                                            |
| ---------------------- | -------------------------------------------------------------------------------------- |
| `X-RateLimit-Scope`    | Which scope was charged: `"user"` or `"workspace"`                                     |
| `X-RateLimit-Scope-ID` | UUID of the charged entity (user ID or workspace ID)                                   |
| `X-RateLimit-Fallback` | Present with value `"true"` only when the request was served under the fallback budget |

### Exceeded Limit Header

| Header        | Description                                               |
| ------------- | --------------------------------------------------------- |
| `Retry-After` | Seconds until the window resets (only on `429` responses) |

## Header Examples

### Workspace has budget

```http
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4658
X-RateLimit-Reset: 1760000060
X-RateLimit-Scope: workspace
X-RateLimit-Scope-ID: aa0e8400-e29b-41d4-a716-446655440005
```

### Workspace exceeded, cascaded to user

The workspace limit (20/600s) was exceeded, so the user's own subscription (5000/60s) was charged instead.

```http
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4900
X-RateLimit-Reset: 1760000060
X-RateLimit-Scope: user
X-RateLimit-Scope-ID: 990e8400-e29b-41d4-a716-446655440004
```

### No workspace (user scope directly)

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 58
X-RateLimit-Reset: 1760000060
X-RateLimit-Scope: user
X-RateLimit-Scope-ID: 990e8400-e29b-41d4-a716-446655440004
```

### Unlimited workspace

```http
X-RateLimit-Limit: 0
X-RateLimit-Remaining: -1
X-RateLimit-Reset: 0
X-RateLimit-Scope: workspace
X-RateLimit-Scope-ID: aa0e8400-e29b-41d4-a716-446655440005
```

### All scopes exceeded (429)

Both workspace and user budgets exhausted on a non-fallback route.

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1760000042
X-RateLimit-Scope: user
X-RateLimit-Scope-ID: 990e8400-e29b-41d4-a716-446655440004
```

```json
{
    "context": "billing",
    "description": "Throughput limit exceeded: 100 weighted requests per 60s",
    "message": "Throughput limit exceeded: 100 weighted requests per 60s"
}
```

### Fallback route (user limit exceeded, billing route still accessible)

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1760000060
X-RateLimit-Scope: user
X-RateLimit-Scope-ID: 990e8400-e29b-41d4-a716-446655440004
X-RateLimit-Fallback: true
```

`X-RateLimit-Fallback: true` indicates the request was served under the fallback budget (default free plan limits), not the user's main subscription.

## Middleware Flow

```
Request arrives
    |
    v
Billing disabled? ----yes----> c.Next() (no headers)
    |no
    v
User authenticated? --no-----> c.Next() (no headers)
    |yes
    v
Workspace in context? --no--> [User scope]
    |yes
    v
Resolve workspace throughput
    |
    v
Workspace unlimited? --yes--> Set unlimited headers (scope=workspace) -> c.Next()
    |no
    v
Increment workspace counter
    |
    v
Within workspace limit? --yes--> Set headers (scope=workspace) -> c.Next()
    |no
    v
[User scope]
Resolve user throughput
    |
    v
User unlimited? --yes--> Set unlimited headers (scope=user) -> c.Next()
    |no
    v
Increment user counter
    |
    v
Within user limit? --yes--> Set headers (scope=user) -> c.Next()
    |no
    v
Is fallback route? --no--> 429 (scope=user)
    |yes
    v
Increment fallback counter (user-fallback:<id>)
    |
    v
Within fallback limit? --yes--> Set headers (scope=user, fallback=true) -> c.Next()
    |no
    v
429 (scope=user, fallback=true)
```

## Usage Endpoint

`GET /billing/usage` always returns user-scoped usage. When `X-Workspace-ID` is provided, workspace-scoped usage is appended.

When a scope's limit is exceeded, an additional entry with `"fallback": true` is included showing the separate fallback budget.

**Example (workspace within limits)**:

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 10,
        "remaining": 90,
        "fallback": false
    },
    {
        "scope": "workspace",
        "workspace_id": "aa0e8400-e29b-41d4-a716-446655440005",
        "unlimited": false,
        "throughput_limit": 5000,
        "window_seconds": 60,
        "current_usage": 342,
        "remaining": 4658,
        "fallback": false
    }
]
```

**Example (workspace exceeded, user still has budget)**:

Requests inside this workspace are now cascading to the user subscription.

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 32,
        "remaining": 68,
        "fallback": false
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

**Example (no workspace)**:

```json
[
    {
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 42,
        "remaining": 58,
        "fallback": false
    }
]
```
