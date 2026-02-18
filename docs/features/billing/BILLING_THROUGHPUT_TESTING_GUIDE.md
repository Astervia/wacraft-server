# Billing by Throughput - Testing Guide

This document maps every testable scenario for the billing system, organized by component. It covers API testing (HTTP), service-level logic, middleware behavior, concurrency, and edge cases. Use it as a checklist for manual QA or as a specification for writing automated tests.

## Prerequisites

Before testing, ensure the server is running with the correct environment:

```bash
# Minimum required for billing enforcement
BILLING_ENABLED=true
DEFAULT_FREE_PLAN_THROUGHPUT=10    # Low value to trigger limits easily
DEFAULT_FREE_PLAN_WINDOW=60

# For Stripe checkout tests (use Stripe test mode)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_test_...
```

You'll need:

- An authenticated user (JWT token)
- A workspace with the user as a member
- The `billing.admin` policy assigned to the user's workspace membership (for admin endpoints)

---

## 1. Billing Toggle (`BILLING_ENABLED`)

### Scenario Map

| #   | Scenario                                                    | Method                              | Expected                                                                |
| --- | ----------------------------------------------------------- | ----------------------------------- | ----------------------------------------------------------------------- |
| 1.1 | Billing disabled (default) — make any authenticated request | Any endpoint                        | 200 OK, **no** `X-RateLimit-*` headers in response                      |
| 1.2 | Billing disabled — send 1000 rapid requests                 | Any endpoint                        | All succeed (no 429), no counter overhead                               |
| 1.3 | Billing disabled — billing API routes still work            | `GET /billing/plan`                 | 200 OK, returns plans list                                              |
| 1.4 | Billing disabled — create plan via API                      | `POST /billing/plan`                | 201 Created, plan is persisted                                          |
| 1.5 | Billing disabled — create manual subscription               | `POST /billing/subscription/manual` | 201 Created                                                             |
| 1.6 | Billing enabled — make request within limit                 | Any endpoint                        | 200 OK, `X-RateLimit-Limit` and `X-RateLimit-Remaining` headers present |
| 1.7 | Enable billing at runtime (restart required)                | Change env + restart                | Enforcement starts immediately                                          |

### How to Test

```bash
# 1.1 — Billing disabled, no headers
# Start server with BILLING_ENABLED=false (or unset)
curl -s -D - -H "Authorization: Bearer $TOKEN" http://localhost:3000/contact \
  | grep -i "x-ratelimit"
# Expected: no output (no rate limit headers)

# 1.6 — Billing enabled, headers present
# Restart server with BILLING_ENABLED=true
curl -s -D - -H "Authorization: Bearer $TOKEN" http://localhost:3000/contact \
  | grep -i "x-ratelimit"
# Expected: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
```

---

## 2. Plan CRUD

### Scenario Map

| #    | Scenario                                       | Method                               | Expected                                                      |
| ---- | ---------------------------------------------- | ------------------------------------ | ------------------------------------------------------------- |
| 2.1  | List plans — unauthenticated                   | `GET /billing/plan`                  | 400 (no auth header)                                          |
| 2.2  | List plans — authenticated, no admin           | `GET /billing/plan`                  | 200 OK, returns plans                                         |
| 2.3  | List plans — with pagination                   | `GET /billing/plan?limit=1&offset=0` | 200 OK, 1 plan returned                                       |
| 2.4  | Create plan — no admin policy                  | `POST /billing/plan`                 | 403 Forbidden                                                 |
| 2.5  | Create standard plan                           | `POST /billing/plan`                 | 201 Created, all fields match                                 |
| 2.6  | Create unlimited plan (`throughput_limit: 0`)  | `POST /billing/plan`                 | 201 Created, `throughput_limit=0`                             |
| 2.7  | Create unlimited plan (`throughput_limit: -1`) | `POST /billing/plan`                 | 201 Created, `throughput_limit=-1`                            |
| 2.8  | Create plan — missing required fields          | `POST /billing/plan`                 | 400 Bad Request                                               |
| 2.9  | Create plan — duplicate slug                   | `POST /billing/plan`                 | 500 (unique constraint violation)                             |
| 2.10 | Update plan                                    | `PUT /billing/plan?id=<uuid>`        | 200 OK, fields updated                                        |
| 2.11 | Update plan — non-admin                        | `PUT /billing/plan?id=<uuid>`        | 403 Forbidden                                                 |
| 2.12 | Delete plan — no subscriptions                 | `DELETE /billing/plan?id=<uuid>`     | 204 No Content                                                |
| 2.13 | Delete plan — has active subscriptions         | `DELETE /billing/plan?id=<uuid>`     | 500 (FK constraint)                                           |
| 2.14 | Default free plan seeded on startup            | Check DB                             | `plans` table has one row with `is_default=true, slug='free'` |

### How to Test

```bash
# 2.5 — Create standard plan
curl -s -X POST http://localhost:3000/billing/plan \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro",
    "slug": "pro",
    "throughput_limit": 1000,
    "window_seconds": 60,
    "duration_days": 30,
    "price_cents": 4900,
    "currency": "usd",
    "active": true
  }'
# Expected: 201 with plan JSON

# 2.6 — Create unlimited plan
curl -s -X POST http://localhost:3000/billing/plan \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Enterprise",
    "slug": "enterprise",
    "throughput_limit": 0,
    "window_seconds": 60,
    "duration_days": 30,
    "price_cents": 29900,
    "currency": "usd",
    "active": true
  }'
# Expected: 201, throughput_limit = 0
```

---

## 3. Subscription Management

### Scenario Map

| #    | Scenario                                                                   | Method                                         | Expected                                           |
| ---- | -------------------------------------------------------------------------- | ---------------------------------------------- | -------------------------------------------------- |
| 3.1  | List subscriptions — no subscriptions                                      | `GET /billing/subscription`                    | 200 OK, empty array `[]`                           |
| 3.2  | List subscriptions — has subscriptions                                     | `GET /billing/subscription`                    | 200 OK, array with plan preloaded                  |
| 3.3  | List subscriptions — filter by workspace                                   | `GET /billing/subscription` + `X-Workspace-ID` | Only workspace-scoped subscriptions                |
| 3.4  | Create manual subscription — user scope                                    | `POST /billing/subscription/manual`            | 201 Created, `scope=user`, `workspace_id=null`     |
| 3.5  | Create manual subscription — workspace scope                               | `POST /billing/subscription/manual`            | 201 Created, `scope=workspace`, `workspace_id` set |
| 3.6  | Create manual subscription — workspace scope without workspace_id          | `POST /billing/subscription/manual`            | 500 error (`workspace_id required`)                |
| 3.7  | Create manual subscription — with throughput_override                      | `POST /billing/subscription/manual`            | 201, `throughput_override` saved                   |
| 3.8  | Create manual subscription — unlimited override (`throughput_override: 0`) | `POST /billing/subscription/manual`            | 201, override is 0                                 |
| 3.9  | Create manual subscription — invalid plan_id                               | `POST /billing/subscription/manual`            | 500 (`plan not found`)                             |
| 3.10 | Create manual subscription — no admin policy                               | `POST /billing/subscription/manual`            | 403 Forbidden                                      |
| 3.11 | Cancel own subscription                                                    | `DELETE /billing/subscription?id=<uuid>`       | 204, `cancelled_at` is set                         |
| 3.12 | Cancel someone else's subscription                                         | `DELETE /billing/subscription?id=<uuid>`       | 500 (`unauthorized`)                               |
| 3.13 | Cancel already-cancelled subscription                                      | `DELETE /billing/subscription?id=<uuid>`       | 500 (`already cancelled`)                          |
| 3.14 | Cancel nonexistent subscription                                            | `DELETE /billing/subscription?id=<random>`     | 500 (`not found`)                                  |
| 3.15 | Subscription `expires_at` is `starts_at + plan.duration_days`              | Create + inspect                               | Dates match                                        |

### How to Test

```bash
# 3.4 — Manual subscription (user scope)
curl -s -X POST http://localhost:3000/billing/subscription/manual \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "'$PLAN_ID'",
    "scope": "user",
    "user_id": "'$USER_ID'"
  }'
# Expected: 201, scope="user", workspace_id=null

# 3.11 — Cancel subscription
curl -s -X DELETE "http://localhost:3000/billing/subscription?id=$SUB_ID" \
  -H "Authorization: Bearer $TOKEN"
# Expected: 204
```

---

## 4. Throughput Enforcement (Middleware)

This is the core billing logic. Test with `BILLING_ENABLED=true` and a low `DEFAULT_FREE_PLAN_THROUGHPUT` (e.g. 5).

### Scenario Map

| #    | Scenario                                                                         | Expected                                                           |
| ---- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| 4.1  | User with no subscriptions — within free plan limit                              | 200 OK, `X-RateLimit-Limit` matches free plan                      |
| 4.2  | User with no subscriptions — exceed free plan limit                              | 429, `Retry-After` header present                                  |
| 4.3  | User with active subscription — within limit                                     | 200 OK, `X-RateLimit-Limit` matches subscription                   |
| 4.4  | User with active subscription — exceed limit                                     | 429 Too Many Requests                                              |
| 4.5  | User with unlimited subscription (`throughput_limit=0`) — any number of requests | All 200 OK, **no** `X-RateLimit-*` headers                         |
| 4.6  | User with unlimited override (`throughput_override=0`) on limited plan           | All 200 OK, no headers (override wins)                             |
| 4.7  | Unauthenticated request (no JWT)                                                 | Middleware skipped, normal auth error from route                   |
| 4.8  | Window expiration — wait for window to pass, then request                        | Counter resets, request succeeds                                   |
| 4.9  | `X-RateLimit-Remaining` decreases with each request                              | Check header value across multiple requests                        |
| 4.10 | `X-RateLimit-Remaining` never goes below 0                                       | Saturate limit, check header                                       |
| 4.11 | 429 response includes correct error body                                         | `{"context":"billing","message":"Throughput limit exceeded: ..."}` |

### How to Test

```bash
# 4.2 — Exceed free plan limit
# With DEFAULT_FREE_PLAN_THROUGHPUT=5, make 6 rapid requests:
for i in $(seq 1 6); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Workspace-ID: $WORKSPACE_ID" \
    http://localhost:3000/contact)
  echo "Request $i: $STATUS"
done
# Expected: Requests 1-5 return 200, request 6 returns 429

# 4.5 — Unlimited subscription
# Create unlimited plan + manual subscription first, then:
for i in $(seq 1 100); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Workspace-ID: $WORKSPACE_ID" \
    http://localhost:3000/contact)
done
echo "Last status: $STATUS"
# Expected: All return 200

# 4.9 — Remaining decreases
curl -s -D - -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/contact | grep "X-RateLimit-Remaining"
# Run twice, second value should be lower
```

---

## 5. Scope Behavior

### Scenario Map

| #   | Scenario                                                              | Expected                                                                     |
| --- | --------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| 5.1 | User-scoped plan — request to workspace A counts                      | Counter increments                                                           |
| 5.2 | User-scoped plan — request to workspace B also counts                 | Same user counter increments (cross-workspace)                               |
| 5.3 | User-scoped plan — exceed limit in workspace A, try workspace B       | 429 in workspace B too (same user counter)                                   |
| 5.4 | Workspace-scoped plan — request in that workspace counts              | Workspace counter increments                                                 |
| 5.5 | Workspace-scoped plan — request in different workspace does NOT count | Different counter, not affected                                              |
| 5.6 | Both scopes active — user limit hit first                             | 429 from user scope                                                          |
| 5.7 | Both scopes active — workspace limit hit first                        | 429 from workspace scope                                                     |
| 5.8 | User-scoped request on route without `X-Workspace-ID`                 | Only user-scoped check runs, workspace check is skipped                      |
| 5.9 | Workspace-scoped subscription but no user-scoped subscription         | User falls back to free plan for user scope; workspace uses its subscription |

### How to Test

```bash
# 5.3 — User-scoped limit is cross-workspace
# Create user-scoped subscription with limit=5
# Make 5 requests to workspace A (exhaust limit)
# Then try workspace B:
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_B_ID" \
  http://localhost:3000/contact
# Expected: 429 (user-scoped limit already exhausted)
```

---

## 6. Subscription Stacking

### Scenario Map

| #   | Scenario                                           | Expected                                                  |
| --- | -------------------------------------------------- | --------------------------------------------------------- |
| 6.1 | Two active user-scoped subscriptions (1000 + 2000) | Effective limit = 3000                                    |
| 6.2 | One limited + one unlimited subscription           | Effective = unlimited                                     |
| 6.3 | Three active workspace-scoped subscriptions        | Limits sum for that workspace                             |
| 6.4 | One subscription expires while another is active   | Limit drops to remaining subscription's value             |
| 6.5 | Stacking uses the smallest `window_seconds`        | If plans have 30s and 60s windows, effective window = 30s |

### How to Test

```bash
# 6.1 — Stacking two subscriptions
# Create two plans: planA (limit=5), planB (limit=10)
# Create manual subscriptions for both on the same user
# Check X-RateLimit-Limit header:
curl -s -D - -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/contact | grep "X-RateLimit-Limit:"
# Expected: X-RateLimit-Limit: 15

# 6.2 — Limited + unlimited = unlimited
# Add a subscription with unlimited plan (throughput_limit=0)
# All requests should succeed without rate limit headers
```

---

## 7. Expiration & Fallback

### Scenario Map

| #   | Scenario                                               | Expected                                             |
| --- | ------------------------------------------------------ | ---------------------------------------------------- |
| 7.1 | Subscription expires — user has no others              | Falls back to default free plan                      |
| 7.2 | Subscription expires — user has another active one     | Limit drops to remaining subscription's value        |
| 7.3 | All subscriptions expired + no default free plan in DB | Falls back to `DEFAULT_FREE_PLAN_THROUGHPUT` env var |
| 7.4 | Default free plan exists in DB                         | Free plan from DB is used (not env var)              |
| 7.5 | Cancelled subscription is not counted                  | Limit excludes cancelled subscriptions               |

### How to Test

To test expiration without waiting, create a subscription with a plan that has `duration_days=0` or directly set `expires_at` to a past timestamp in the database:

```sql
-- Force-expire a subscription for testing
UPDATE subscriptions SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = '<sub_id>';
```

Then make a request and verify the limit drops to the free plan.

---

## 8. Endpoint Weights

### Scenario Map

| #   | Scenario                                     | Expected                                                                           |
| --- | -------------------------------------------- | ---------------------------------------------------------------------------------- |
| 8.1 | Default weight — all endpoints cost 1        | Each request decrements remaining by 1                                             |
| 8.2 | Custom weight (5) on `/message` POST         | Each POST /message decrements remaining by 5                                       |
| 8.3 | Custom weight — endpoint not in table        | Falls back to weight 1                                                             |
| 8.4 | Delete custom weight — endpoint reverts to 1 | Weight goes back to 1 after delete                                                 |
| 8.5 | Create endpoint weight — non-admin           | 403 Forbidden                                                                      |
| 8.6 | List endpoint weights                        | Returns all custom weights                                                         |
| 8.7 | Weight applied correctly at limit boundary   | With limit=10 and weight=3: 3 requests OK (3+3+3=9), 4th returns 429 (9+3=12 > 10) |

### How to Test

```bash
# 8.2 — Create custom weight
curl -s -X POST http://localhost:3000/billing/endpoint-weight \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "POST",
    "path_pattern": "/message",
    "weight": 5
  }'

# 8.7 — Verify weighted counting
# With free plan limit=10, weight=3 on GET /contact:
for i in $(seq 1 4); do
  RESP=$(curl -s -D - -H "Authorization: Bearer $TOKEN" \
    -H "X-Workspace-ID: $WORKSPACE_ID" \
    http://localhost:3000/contact)
  STATUS=$(echo "$RESP" | head -1 | awk '{print $2}')
  REMAINING=$(echo "$RESP" | grep -i "x-ratelimit-remaining:" | awk '{print $2}')
  echo "Request $i: status=$STATUS remaining=$REMAINING"
done
# Expected: 1:remaining=7, 2:remaining=4, 3:remaining=1, 4:status=429
```

---

## 9. Usage Endpoint

### Scenario Map

| #   | Scenario                                        | Expected                                            |
| --- | ----------------------------------------------- | --------------------------------------------------- |
| 9.1 | Fresh user — no requests made                   | `current_usage=0`, `remaining=<limit>`              |
| 9.2 | After N requests                                | `current_usage=N`, `remaining=<limit>-N`            |
| 9.3 | Unlimited user                                  | `unlimited=true`, `remaining=-1`, `current_usage=0` |
| 9.4 | With `X-Workspace-ID` — two summaries returned  | Array with user scope + workspace scope entries     |
| 9.5 | Without `X-Workspace-ID` — one summary returned | Array with only user scope entry                    |
| 9.6 | After window expires                            | `current_usage` resets to 0                         |

### How to Test

```bash
# 9.2 — Usage after requests
# Make 3 requests, then check usage:
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3000/billing/usage | jq
# Expected: current_usage=3 (approximately, includes this request)

# 9.3 — Unlimited usage
# User with unlimited subscription:
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3000/billing/usage | jq
# Expected: [{"scope":"user","unlimited":true,"remaining":-1,...}]

# 9.4 — With workspace context
curl -s -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/billing/usage | jq
# Expected: 2 items in array (user + workspace)
```

---

## 10. Stripe Checkout Flow

### Scenario Map

| #    | Scenario                                            | Expected                                      |
| ---- | --------------------------------------------------- | --------------------------------------------- |
| 10.1 | Checkout — Stripe not configured                    | 503 Service Unavailable                       |
| 10.2 | Checkout — valid request                            | 200 OK with `checkout_url` and `external_id`  |
| 10.3 | Checkout — inactive plan                            | 400 (`plan not available`)                    |
| 10.4 | Checkout — nonexistent plan                         | 404                                           |
| 10.5 | Checkout — workspace scope without `workspace_id`   | 400 (`workspace_id required`)                 |
| 10.6 | Checkout — missing `success_url`                    | 400 (validation error)                        |
| 10.7 | Stripe webhook — valid `checkout.session.completed` | Subscription created in DB, cache invalidated |
| 10.8 | Stripe webhook — invalid signature                  | 400                                           |
| 10.9 | Stripe webhook — unhandled event type               | 200 OK (no-op)                                |

### How to Test

```bash
# 10.2 — Checkout (use Stripe test mode)
curl -s -X POST http://localhost:3000/billing/subscription/checkout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "'$PLAN_ID'",
    "scope": "user",
    "success_url": "http://localhost:3000/success",
    "cancel_url": "http://localhost:3000/cancel"
  }'
# Expected: 200 with checkout_url pointing to checkout.stripe.com

# 10.7 — Stripe webhook (use Stripe CLI)
stripe listen --forward-to localhost:3000/billing/webhook/stripe
# Then complete a test checkout. Verify subscription appears:
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/billing/subscription | jq
```

---

## 11. Cache & Invalidation

### Scenario Map

| #    | Scenario                                              | Expected                                                  |
| ---- | ----------------------------------------------------- | --------------------------------------------------------- |
| 11.1 | Create subscription — cache invalidated               | Next request resolves new limit (not stale cache)         |
| 11.2 | Cancel subscription — cache invalidated               | Next request resolves reduced limit                       |
| 11.3 | Cache TTL (5 min) — stale entry reused within TTL     | Same limit returned without DB query for 5 min            |
| 11.4 | Cache TTL expired — DB re-queried                     | After 5 min, fresh data fetched                           |
| 11.5 | Concurrent requests for same user on cold cache       | Only one DB query (MutexSwapper prevents thundering herd) |
| 11.6 | Concurrent requests for different users on cold cache | Both DB queries run in parallel (not blocking each other) |

### How to Test

```bash
# 11.1 — Invalidation on subscription create
# Check X-RateLimit-Limit before and after creating a subscription:
# Before:
curl -s -D - -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/contact | grep "X-RateLimit-Limit:"
# Create subscription...
# After (immediately):
curl -s -D - -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/contact | grep "X-RateLimit-Limit:"
# Expected: Limit changed immediately (no 5-min delay)
```

---

## 12. Policy & Authorization

### Scenario Map

| #     | Scenario                                                   | Expected                                 |
| ----- | ---------------------------------------------------------- | ---------------------------------------- |
| 12.1  | List plans — any authenticated user                        | 200 OK                                   |
| 12.2  | Create plan — user with `billing.admin`                    | 201 Created                              |
| 12.3  | Create plan — user with `workspace.admin`                  | 201 Created (workspace.admin grants all) |
| 12.4  | Create plan — user without billing policies                | 403 Forbidden                            |
| 12.5  | Create manual subscription — `billing.admin` required      | 403 without, 201 with                    |
| 12.6  | Endpoint weights — `billing.admin` required                | 403 without, 200/201 with                |
| 12.7  | Checkout — any authenticated user (no policy needed)       | 200 OK                                   |
| 12.8  | Cancel subscription — any authenticated user (owner check) | 204 if owner, error if not               |
| 12.9  | Usage — any authenticated user                             | 200 OK                                   |
| 12.10 | Stripe webhook — no auth required                          | 200 OK (signature-validated)             |

---

## 13. Edge Cases & Boundary Conditions

### Scenario Map

| #     | Scenario                                                | Expected                                                       |
| ----- | ------------------------------------------------------- | -------------------------------------------------------------- |
| 13.1  | Plan with `window_seconds=1` — very fast window         | Counter resets every second                                    |
| 13.2  | Plan with `window_seconds=3600` — 1-hour window         | Counter persists for full hour                                 |
| 13.3  | Plan with `duration_days=0`                             | Subscription expires instantly (ExpiresAt = StartsAt)          |
| 13.4  | Plan with `duration_days=36500`                         | ~100 year subscription (perpetual)                             |
| 13.5  | Subscription `starts_at` in the future                  | Subscription not counted until start date                      |
| 13.6  | Multiple default plans in DB                            | First one returned by DB query is used                         |
| 13.7  | Exactly at the limit (count == limit)                   | Request succeeds (exceeded only when count > limit)            |
| 13.8  | Weight = 0 endpoint                                     | Request costs nothing (counter not incremented)                |
| 13.9  | Very large weight (e.g. 9999) on low limit              | Single request may exceed limit                                |
| 13.10 | Request during window boundary (exact second of expiry) | Window resets, counter starts fresh                            |
| 13.11 | `throughput_override` set to negative value             | Treated as unlimited (effective <= 0)                          |
| 13.12 | Server restart — counters reset                         | All in-memory counters lost; usage starts from 0               |
| 13.13 | Plan update while subscription active                   | Cached limit may be stale for up to 5 min; no immediate effect |

---

## 14. Concurrency

### Scenario Map

| #    | Scenario                                     | Expected                                                                                |
| ---- | -------------------------------------------- | --------------------------------------------------------------------------------------- |
| 14.1 | 100 concurrent requests from same user       | Counter accurately tracks all 100 (no lost increments)                                  |
| 14.2 | 100 concurrent requests from different users | Each user's counter is independent (MutexSwapper per-key)                               |
| 14.3 | Concurrent subscription creation + requests  | Cache invalidated correctly; no stale data served after invalidation                    |
| 14.4 | Concurrent cancel + request                  | Request may use stale cache briefly; next request after invalidation sees updated limit |

### How to Test

```bash
# 14.1 — Concurrent accuracy (requires a tool like hey, ab, or wrk)
# With limit=200, send 200 concurrent requests:
hey -n 200 -c 50 \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  http://localhost:3000/contact

# Check usage endpoint:
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3000/billing/usage | jq
# Expected: current_usage = 201 (200 from hey + 1 from this request)

# 14.2 — Different users don't block each other
# Run two concurrent hey sessions with different tokens
# Both should complete without artificial serialization delays
```

---

## 15. Response Validation

### Headers

For every successful response when billing is enabled and scope is limited:

| Header                            | Format                  | Example      |
| --------------------------------- | ----------------------- | ------------ |
| `X-RateLimit-Limit`               | Integer                 | `1000`       |
| `X-RateLimit-Remaining`           | Integer >= 0            | `997`        |
| `X-RateLimit-Reset`               | Unix timestamp          | `1707652800` |
| `X-RateLimit-Limit-Workspace`     | Integer (optional)      | `500`        |
| `X-RateLimit-Remaining-Workspace` | Integer >= 0 (optional) | `498`        |

For 429 responses:

| Header                  | Format            | Example      |
| ----------------------- | ----------------- | ------------ |
| `Retry-After`           | Seconds (integer) | `45`         |
| `X-RateLimit-Limit`     | Integer           | `1000`       |
| `X-RateLimit-Remaining` | `0`               | `0`          |
| `X-RateLimit-Reset`     | Unix timestamp    | `1707652800` |

### Body Shapes

**429 error body**:

```json
{
    "context": "billing",
    "description": "Throughput limit exceeded: <limit> weighted requests per <window>s",
    "message": "Throughput limit exceeded: <limit> weighted requests per <window>s"
}
```

**Usage response (limited)**:

```json
[
    {
        "scope": "user",
        "user_id": "<uuid>",
        "unlimited": false,
        "throughput_limit": 1000,
        "window_seconds": 60,
        "current_usage": 42,
        "remaining": 958
    }
]
```

**Usage response (unlimited)**:

```json
[
    {
        "scope": "user",
        "user_id": "<uuid>",
        "unlimited": true,
        "throughput_limit": 0,
        "window_seconds": 0,
        "current_usage": 0,
        "remaining": -1
    }
]
```

---

## 16. Full Integration Scenarios

These are end-to-end flows that cross multiple components.

### 16.1 — Free User Lifecycle

1. New user registers
2. Makes requests → limited by default free plan (e.g. 100/min)
3. Check `GET /billing/usage` → `throughput_limit=100`
4. Exceed limit → 429
5. Wait for window reset → requests succeed again

### 16.2 — Plan Purchase via Stripe

1. Admin creates a plan via `POST /billing/plan`
2. User lists plans via `GET /billing/plan`
3. User initiates checkout via `POST /billing/subscription/checkout`
4. User completes payment on Stripe
5. Stripe sends webhook to `POST /billing/webhook/stripe`
6. Subscription is created
7. User's limit increases immediately (cache invalidated)
8. Check `GET /billing/usage` → new limit
9. Check `GET /billing/subscription` → subscription visible

### 16.3 — Admin Custom Deal

1. Admin creates custom plan with `is_custom=true`
2. Admin creates manual subscription via `POST /billing/subscription/manual` with `throughput_override=0` (unlimited)
3. User makes unlimited requests → all succeed, no rate limit headers
4. Check `GET /billing/usage` → `unlimited=true`

### 16.4 — Plan Expiration & Downgrade

1. User has active paid subscription (limit=5000)
2. Subscription expires (set `expires_at` to past in DB for testing)
3. Wait for cache TTL (5 min) or invalidate
4. User's limit drops to free plan (100/min)
5. Check `GET /billing/usage` → `throughput_limit=100`

### 16.5 — Stacking & Cancellation

1. User has two active subscriptions: plan A (500/min) + plan B (1000/min)
2. Check limit → 1500/min
3. User cancels plan B via `DELETE /billing/subscription?id=<planB_sub_id>`
4. Check limit → 500/min (immediately, cache invalidated on cancel)

### 16.6 — Dual Scope Enforcement

1. Admin creates workspace-scoped subscription (limit=50) for workspace W
2. User also has user-scoped subscription (limit=200)
3. User makes 51 requests to workspace W → 429 on request 51 (workspace limit hit)
4. User switches to workspace V (no workspace subscription) → requests succeed (user limit still has capacity)
5. User makes 200 total requests across all workspaces → 429 on request 201 (user limit hit)

### 16.7 — Weighted Endpoint Surge

1. Admin sets weight=10 on `POST /message`
2. User has plan with limit=100/min
3. User sends 10 messages → counter at 100
4. User sends 11th message → 429
5. User reads contacts (`GET /contact`, weight=1) → also 429 (user-scoped counter exhausted)

---

## Quick Reference: Test Matrix

| Component      | Happy Path                | Error Cases                      | Edge Cases          | Concurrency |
| -------------- | ------------------------- | -------------------------------- | ------------------- | ----------- |
| Billing toggle | 1.1, 1.3, 1.6             | —                                | 1.2, 1.4            | —           |
| Plan CRUD      | 2.2, 2.5, 2.6, 2.10, 2.12 | 2.1, 2.4, 2.8, 2.9, 2.13         | 2.7, 2.14           | —           |
| Subscriptions  | 3.2, 3.4, 3.5, 3.11       | 3.6, 3.9, 3.10, 3.12, 3.13, 3.14 | 3.7, 3.8, 3.15      | 14.3        |
| Middleware     | 4.1, 4.3, 4.5, 4.9        | 4.2, 4.4, 4.11                   | 4.6, 4.7, 4.8, 4.10 | 14.1, 14.2  |
| Scopes         | 5.1, 5.4, 5.8             | 5.3, 5.6, 5.7                    | 5.2, 5.5, 5.9       | —           |
| Stacking       | 6.1, 6.3                  | —                                | 6.2, 6.4, 6.5       | —           |
| Expiration     | 7.1, 7.4                  | —                                | 7.2, 7.3, 7.5       | —           |
| Weights        | 8.1, 8.2, 8.6             | 8.5                              | 8.3, 8.4, 8.7       | —           |
| Usage          | 9.1, 9.2, 9.4             | —                                | 9.3, 9.5, 9.6       | —           |
| Stripe         | 10.2, 10.7                | 10.1, 10.3-10.6, 10.8            | 10.9                | —           |
| Cache          | 11.1, 11.2                | —                                | 11.3-11.6           | 11.5, 11.6  |
| Policies       | 12.1, 12.2, 12.7          | 12.4, 12.5, 12.6                 | 12.3, 12.8, 12.10   | —           |
| Boundaries     | —                         | —                                | 13.1-13.13          | —           |
