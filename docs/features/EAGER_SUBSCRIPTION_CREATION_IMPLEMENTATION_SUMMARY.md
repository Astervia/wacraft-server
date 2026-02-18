# Eager Subscription Creation at Checkout - Implementation Summary

This document describes the eager subscription creation feature, which pre-creates subscriptions in a "pending" state at checkout time. This ensures a local record exists before the `checkout.session.completed` webhook fires, enabling sync-based recovery if the webhook fails.

## Problem

Previously, subscriptions were only created when the `checkout.session.completed` webhook arrived. If that webhook failed (network issues, server downtime, Stripe retry exhaustion), the subscription was never created locally. The sync endpoint couldn't help because there was no local record to reconcile against.

This created a gap: the user paid successfully on Stripe, but the application had no knowledge of the purchase.

## Solution

Create subscriptions in the database at checkout time with `status = "pending"`. The webhook transitions them to `"active"`. If the webhook fails, the user can call the sync endpoint to reconcile — because the local record already exists.

### Subscription Status Lifecycle

```
Checkout initiated  ──>  pending
Webhook fires       ──>  active
Sync (if paid)      ──>  active
Sync (if expired)   ──>  cancelled
Cancellation        ──>  cancelled
```

### Why a `Status` Field?

- **Why not manipulate `StartsAt`?** Semantically wrong — `StartsAt` means "when the period begins", not "whether the user paid".
- **Why not a separate `ActivatedAt` field?** Creates an implicit status every query must remember to check. A status field is explicit.
- **Migration safety:** The field defaults to `'active'`, so all existing rows remain correct without a data migration.

## Status Values

| Status      | Meaning                                   | Affects Throughput? | Visible in Active Queries? |
| ----------- | ----------------------------------------- | ------------------- | -------------------------- |
| `pending`   | Checkout initiated, payment not confirmed | No                  | No                         |
| `active`    | Payment confirmed, subscription is live   | Yes                 | Yes                        |
| `cancelled` | Subscription ended (cancelled or expired) | No                  | No                         |

## Checkout Flow (Updated)

```
1. User calls POST /billing/subscription/checkout
2. Handler creates a Stripe Checkout Session (returns checkout URL + session ID)
3. Handler calls CreatePendingSubscription (status = "pending")
   - If this fails, log a warning but still return the checkout URL (graceful degradation)
4. User completes payment on Stripe
5. Stripe fires checkout.session.completed webhook
6. ActivateSubscription looks up pending record by payment_external_id
   - If found: transitions to active, sets StartsAt/ExpiresAt/StripeSubscriptionID
   - If not found: falls back to creating a new active subscription (backward compat)
```

## ActivateSubscription Logic

The function was transformed from a simple "create" to a "find-and-update" with three paths:

```
1. Idempotency check: if a subscription with this payment_external_id
   already exists and is active, return it without modification.

2. Find pending: look up by payment_external_id + status = "pending".
   If found: set status = "active", reset StartsAt = now,
   ExpiresAt = now + plan duration, store StripeSubscriptionID, save,
   persist Stripe Customer ID on user, invalidate cache.

3. Fallback create: no pending record found — create a new active
   subscription (backward compat for pre-migration webhooks or
   edge cases where the pending record wasn't created).
```

## Sync Route — How It Works

The sync endpoint (`POST /billing/subscription/sync?id=<uuid>`) now supports three sync paths based on the subscription's current state:

### Path 1: Pending Subscription (Any Payment Mode)

When a subscription has `status = "pending"`, sync uses the checkout session ID (`payment_external_id`) to query Stripe's Checkout Session API:

```
1. Call GetCheckoutSessionStatus(payment_external_id)
2. Stripe returns: session_status, payment_status, subscription_id, customer_id
3. If payment_status == "paid":
   - Set status = "active"
   - Set StartsAt = now, ExpiresAt = now + plan duration
   - Store StripeSubscriptionID (if present, i.e. subscription mode)
   - Persist Stripe Customer ID on user
   - Invalidate throughput cache
4. If session_status == "expired":
   - Set status = "cancelled", CancelledAt = now
5. Otherwise (session still open):
   - Return subscription as-is (no changes)
```

This path works for **both payment modes**:

- **One-time payments (`payment` mode):** The checkout session's `payment_status` tells us whether the user paid. If `"paid"`, the subscription is activated. The `StripeSubscriptionID` will be empty (one-time payments don't create Stripe subscriptions), which is expected. The subscription's `ExpiresAt` is set based on the plan's `duration_days` and will expire naturally.
- **Recurring subscriptions (`subscription` mode):** Same flow, but the checkout session also carries a `subscription_id` which gets stored as `StripeSubscriptionID`. Future syncs on this subscription (once active) will use Path 2.

### Path 2: Active Subscription-Mode with StripeSubscriptionID

This is the existing sync behavior for active recurring subscriptions. It fetches the subscription state from Stripe's Subscription API:

```
1. Call GetSubscriptionDetails(stripe_subscription_id)
2. Update: expires_at, cancel_at_period_end
3. If Stripe status == "canceled": set cancelled_at, status = "cancelled"
4. Invalidate throughput cache
```

### Path 3: Everything Else

Returns an error. This covers:

- Active one-time payment subscriptions (no provider-side state to sync — they expire naturally)
- Cancelled subscriptions (terminal state)

### Sync Decision Matrix

| Status    | Payment Mode   | Has StripeSubscriptionID? | Sync Path                |
| --------- | -------------- | ------------------------- | ------------------------ |
| `pending` | `payment`      | No                        | Checkout Session API     |
| `pending` | `subscription` | No                        | Checkout Session API     |
| `active`  | `subscription` | Yes                       | Stripe Subscription API  |
| `active`  | `payment`      | No                        | Error (no state to sync) |

### Frontend Integration

The sync button should now be shown for:

- **Pending subscriptions** (any mode) — to recover from missed webhooks
- **Active subscription-mode subscriptions** — to fix stale renewal/cancellation state

```
if status == "pending":
    Show "Sync" button (label: "Check payment status")
else if payment_mode == "subscription" and stripe_subscription_id is not null:
    Show "Sync" button (label: "Sync with Stripe")
```

## Abandoned Checkout Cleanup

Pending subscriptions from abandoned checkouts will accumulate but are harmless — they are invisible to throughput resolution and active subscription queries (both filter by `status = 'active'`). A background cleanup job can be added in a follow-up to periodically check stale pending subscriptions (>48h old) via `GetCheckoutSessionStatus` and cancel expired ones.

## Files Changed

| File                                                    | Change                                                                                                                                                                                                                                                                                                           |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `wacraft-core/src/billing/model/subscription-status.go` | **NEW**: `SubscriptionStatus` type with `pending`, `active`, `cancelled` constants                                                                                                                                                                                                                               |
| `wacraft-core/src/billing/entity/subscription.go`       | Added `Status` field (default `'active'`), updated `IsActive()` to require `Status == active`                                                                                                                                                                                                                    |
| `src/billing/service/plan.go`                           | Added `AND status = 'active'` filter to `queryThroughput`                                                                                                                                                                                                                                                        |
| `src/billing/service/subscription.go`                   | Added `CreatePendingSubscription`, transformed `ActivateSubscription` to find-and-update with idempotency + fallback, extended `SyncSubscription` with pending/active/error paths, updated `MarkSubscriptionCancelled` to set `Status = cancelled`, added `status = 'active'` filter to `GetActiveSubscriptions` |
| `src/billing/handler/subscription.go`                   | `Checkout` now calls `CreatePendingSubscription` after creating the checkout session                                                                                                                                                                                                                             |
| `src/billing/service/payment/provider.go`               | Added `CheckoutSessionStatus` struct and `GetCheckoutSessionStatus` method to `Provider` interface                                                                                                                                                                                                               |
| `src/billing/service/payment/stripe.go`                 | Implemented `GetCheckoutSessionStatus` using `session.Get()`                                                                                                                                                                                                                                                     |
