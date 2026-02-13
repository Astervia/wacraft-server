# Billing Rate Limit Detection Guide

This guide explains how frontend clients can detect whether billing throughput limits are enabled and how to interpret rate limit headers, including unlimited scopes.

## Quick Rules

1) **Billing disabled**
- No `X-RateLimit-*` headers are present on responses.

2) **Billing enabled + limited**
- `X-RateLimit-Limit` and `X-RateLimit-Remaining` are present with positive values.
- `X-RateLimit-Reset` is a Unix timestamp for when the window resets.

3) **Billing enabled + unlimited (user scope)**
- `X-RateLimit-Limit: 0`
- `X-RateLimit-Remaining: -1`
- `X-RateLimit-Reset: 0`

4) **Billing enabled + unlimited (workspace scope)**
- `X-RateLimit-Limit-Workspace: 0`
- `X-RateLimit-Remaining-Workspace: -1`

5) **Billing enabled + rate limited (429)**
- `X-RateLimit-Limit` and `X-RateLimit-Remaining: 0` are present.
- `Retry-After` header contains seconds until the window resets.

6) **Fallback routes (when main limit is exceeded)**
- Certain routes get a **separate throughput budget** using the default free plan limits, even when the user's subscription limit is exceeded. This ensures users can always access routes needed to upgrade.
- Fallback routes: `/billing/plan`, `/billing/subscription`, `/billing/usage`, `GET /workspace`, `GET /user/me`.
- When operating under fallback limits, the `X-RateLimit-*` headers reflect the **fallback budget** (default free plan), not the main subscription.
- If both the main limit and the fallback limit are exhausted, the response is `429`.

## Recommended Frontend Checks

- Treat **absence of any `X-RateLimit-*` headers** as billing disabled.
- Treat **`Limit = 0` and `Remaining = -1`** as unlimited for that scope.
- On **`429` responses**, read the `Retry-After` header and show a countdown or upgrade prompt.
- When a `429` is received on a **non-fallback** route, billing/upgrade routes remain accessible — redirect the user to the billing page.
- Call `GET /billing/usage` with `X-Workspace-ID` to get **both** user and workspace-scoped usage. Without the header, only user-scoped usage is returned.
- When the response contains entries with `"fallback": true`, the user has exceeded their main limit — show the exceeded subscription alongside the fallback budget remaining for billing routes.
- Otherwise, treat the scope as limited and display remaining/limit as usual.

## Example

Limited user scope:

```http
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4658
X-RateLimit-Reset: 1760000000
```

Unlimited user scope:

```http
X-RateLimit-Limit: 0
X-RateLimit-Remaining: -1
X-RateLimit-Reset: 0
```

Unlimited workspace scope:

```http
X-RateLimit-Limit-Workspace: 0
X-RateLimit-Remaining-Workspace: -1
```

Rate limited (429):

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1760000042
```

```json
{
    "context": "billing",
    "description": "Throughput limit exceeded: 5000 weighted requests per 60s",
    "message": "Throughput limit exceeded: 5000 weighted requests per 60s"
}
```

Fallback route (user exceeded main limit, but billing route still accessible):

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1760000060
```

The lower `Limit` value (100 vs the subscription's 5000) indicates the response is served under fallback limits.

## Fallback Routes

When a user exceeds their subscription's throughput limit, most routes return `429`. However, the following routes remain accessible under a separate budget equal to the default free plan limits (e.g. 100 req/min):

| Method | Path                     | Purpose                    |
| ------ | ------------------------ | -------------------------- |
| `*`    | `/billing/plan`          | Browse available plans     |
| `*`    | `/billing/subscription`  | Manage subscriptions       |
| `GET`  | `/billing/usage`         | Check current usage        |
| `GET`  | `/workspace`             | List/view workspaces       |
| `GET`  | `/user/me`               | Get current user profile   |

This separate budget is tracked independently from the main subscription counter, so regular API usage cannot exhaust the fallback budget. If both the main and fallback budgets are exceeded, the fallback routes also return `429`.

## Usage Endpoint

`GET /billing/usage` accepts an optional `X-Workspace-ID` header. When provided, the workspace is validated (membership check) and workspace-scoped usage is included in the response.

**Important**: Without `X-Workspace-ID`, only user-scoped usage is returned. If a `429` came from the workspace scope, the frontend must pass `X-Workspace-ID` to see the workspace usage that caused the limit.

When a scope's limit is exceeded, the response includes two entries for that scope:

| Entry            | `fallback` | Description                                        |
| ---------------- | ---------- | -------------------------------------------------- |
| Subscription     | `false`    | Main subscription usage (shows exceeded limit)     |
| Fallback budget  | `true`     | Separate budget for billing routes (free plan limits) |

Example (workspace limit exceeded):

```json
[
    {
        "scope": "user",
        "user_id": "a2f83722-f7e9-46a9-bbf3-691b5ab941ce",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 10,
        "remaining": 90,
        "fallback": false
    },
    {
        "scope": "workspace",
        "workspace_id": "6e859def-125d-4b1d-92b6-68f028444dcb",
        "unlimited": false,
        "throughput_limit": 20,
        "window_seconds": 600,
        "current_usage": 25,
        "remaining": 0,
        "fallback": false
    },
    {
        "scope": "workspace",
        "workspace_id": "6e859def-125d-4b1d-92b6-68f028444dcb",
        "unlimited": false,
        "throughput_limit": 100,
        "window_seconds": 60,
        "current_usage": 3,
        "remaining": 97,
        "fallback": true
    }
]
```

The frontend should:
1. Find entries with `remaining: 0` to identify which scope is exceeded
2. Find matching `fallback: true` entries to show the remaining fallback budget
3. Prompt the user to upgrade their plan
