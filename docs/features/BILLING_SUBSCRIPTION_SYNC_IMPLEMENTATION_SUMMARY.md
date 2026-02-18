# Billing Subscription Sync - Implementation Summary

This document describes the subscription sync endpoint that reconciles local subscription state with the payment provider (Stripe). It exists as a recovery mechanism for when webhooks (`invoice.paid`, `customer.subscription.updated`, `customer.subscription.deleted`) fail or arrive out of order, leaving local state stale.

## Problem

The subscription lifecycle depends on Stripe webhooks to update local state:

| Webhook                         | Local Update                           |
| ------------------------------- | -------------------------------------- |
| `invoice.paid`                  | Extends `expires_at` to new period end |
| `customer.subscription.updated` | Syncs `cancel_at_period_end`           |
| `customer.subscription.deleted` | Sets `cancelled_at`                    |

If any of these webhooks fail (network issues, server downtime, Stripe retry exhaustion), the local subscription record drifts from reality. For example, a user who paid successfully might see their subscription expire because the `invoice.paid` webhook never arrived.

## Solution

A new endpoint lets authenticated users manually trigger a reconciliation between local state and Stripe's source of truth:

```
POST /billing/subscription/sync?id=<uuid>
```

## Scope

Only **subscription-mode** subscriptions (those with a `stripe_subscription_id`) can be synced. Payment-mode (one-time) subscriptions have no provider-side state to fetch — their lifecycle is fully determined by `starts_at`/`expires_at`.

## Sync Logic

The endpoint fetches the current subscription state from Stripe and updates local fields:

```
1. Fetch subscription from DB (with Plan preload)
2. Verify ownership (sub.UserID == caller's userID)
3. Reject if payment_mode != "subscription" or stripe_subscription_id is nil
4. Call Stripe: GET /v1/subscriptions/{stripe_subscription_id}
5. Update local fields:
   - expires_at = Stripe current_period_end
   - cancel_at_period_end = Stripe cancel_at_period_end
   - If Stripe status == "canceled" and cancelled_at is nil -> cancelled_at = now
6. Save to DB
7. Invalidate throughput cache
8. Return updated subscription (with preloaded Plan)
```

### Field Reconciliation

| Local Field            | Stripe Source                 | Fixes Missed Webhook            |
| ---------------------- | ----------------------------- | ------------------------------- |
| `expires_at`           | `items[0].current_period_end` | `invoice.paid`                  |
| `cancel_at_period_end` | `cancel_at_period_end`        | `customer.subscription.updated` |
| `cancelled_at`         | `status == "canceled"`        | `customer.subscription.deleted` |

## API

### Sync Subscription — `POST /billing/subscription/sync?id=<uuid>`

**Authentication**: Required (user token + verified email).

**Query Parameters**:

| Parameter | Type | Required | Description     |
| --------- | ---- | -------- | --------------- |
| `id`      | uuid | Yes      | Subscription ID |

**Success Response** (`200`):

Returns the updated subscription entity with preloaded Plan, identical to the format returned by `GET /billing/subscription`:

```json
{
    "id": "880e8400-e29b-41d4-a716-446655440003",
    "plan_id": "660e8400-e29b-41d4-a716-446655440001",
    "scope": "user",
    "user_id": "990e8400-e29b-41d4-a716-446655440004",
    "starts_at": "2026-02-17T12:00:00Z",
    "expires_at": "2026-03-19T12:00:00Z",
    "payment_provider": "stripe",
    "payment_mode": "subscription",
    "stripe_subscription_id": "sub_1234567890",
    "cancel_at_period_end": false,
    "plan": {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "Pro",
        "slug": "pro",
        "throughput_limit": 5000,
        "window_seconds": 60
    }
}
```

**Error Responses**:

| Status | Condition                                                                               |
| ------ | --------------------------------------------------------------------------------------- |
| `400`  | Missing or invalid `id` query parameter                                                 |
| `500`  | Subscription not found, not owned by caller, not subscription-mode, or Stripe API error |
| `503`  | Payment provider not configured                                                         |

## Frontend Integration

The sync button should be available as a fallback action on subscription-mode subscriptions. Use it when the displayed state looks stale (e.g., user paid but `expires_at` hasn't updated).

```
if payment_mode == "subscription" and stripe_subscription_id is not null:
    Show "Sync" button
    On click: POST /billing/subscription/sync?id=<id>
    On success: Replace subscription in local state with response
```

The sync endpoint is idempotent. Calling it when local state is already correct is a no-op (the same values are written back).

## Files Changed

| File                                      | Change                                                                                                                                    |
| ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `src/billing/service/payment/provider.go` | Added `SubscriptionDetails` struct and `GetSubscriptionDetails` method to `Provider` interface                                            |
| `src/billing/service/payment/stripe.go`   | Implemented `GetSubscriptionDetails` using `subscription.Get()`. Reads `CurrentPeriodEnd` from `sub.Items.Data[0]` (Stripe v84 SDK)       |
| `src/billing/service/subscription.go`     | Added `SyncSubscription(subscriptionID, userID)` with ownership check, provider fetch, field reconciliation, save, and cache invalidation |
| `src/billing/handler/subscription.go`     | Added `SyncSubscription` handler (parses `id` from query, checks provider, calls service, returns updated entity)                         |
| `src/billing/router/main.go`              | Registered `POST /billing/subscription/sync` with user auth, email verification, and throughput middleware                                |
