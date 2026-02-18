# Billing Subscription Renewal - Implementation Summary

This document describes the subscription renewal system added on top of the existing throughput-based billing. Each checkout now supports two payment modes: **one-time payment** (the original behavior) and **recurring subscription** (auto-renews via Stripe). The payment mode is chosen per-checkout, not per-plan — the same plan can be purchased either way.

## Core Concepts

### Payment Modes

The `payment_mode` field on each subscription determines its lifecycle:

| Mode      | Value            | Stripe Checkout Mode   | Renewal                                                          | Cancellation                                   |
| --------- | ---------------- | ---------------------- | ---------------------------------------------------------------- | ---------------------------------------------- |
| One-time  | `"payment"`      | `mode: "payment"`      | None. Expires at `expires_at`.                                   | Cannot be cancelled. Expires naturally.        |
| Recurring | `"subscription"` | `mode: "subscription"` | Auto-renews every `duration_days`. Stripe charges automatically. | Stops renewal. Stays active until period ends. |

Payment mode is chosen at checkout time. If omitted, defaults to `"payment"` (backwards compatible).

### Subscription Lifecycle — One-Time (payment mode)

```
User checkouts with payment_mode="payment"
  -> Stripe one-time payment
  -> checkout.session.completed webhook
  -> Subscription created (starts_at=now, expires_at=now+duration_days)
  -> Subscription expires at expires_at, falls back to free plan
```

No renewal. No cancellation. The subscription simply expires.

### Subscription Lifecycle — Recurring (subscription mode)

```
1. User checkouts with payment_mode="subscription"
     -> Stripe subscription checkout (creates Stripe Customer + Subscription)
     -> checkout.session.completed webhook
     -> Subscription created with stripe_subscription_id

2. Every duration_days, Stripe auto-charges
     -> invoice.paid webhook
     -> expires_at extended to new period end

3. User cancels (DELETE /billing/subscription?id=<uuid>)
     -> Backend calls Stripe: cancel_at_period_end=true
     -> Sets cancel_at_period_end=true on the subscription
     -> Subscription stays active until current period ends
     -> No cancelled_at set yet

4a. User reactivates (POST /billing/subscription/reactivate?id=<uuid>)
     -> Backend calls Stripe: cancel_at_period_end=false
     -> Sets cancel_at_period_end=false on the subscription
     -> Subscription resumes auto-renewal

4b. Period ends after cancellation (if not reactivated)
     -> customer.subscription.deleted webhook
     -> cancelled_at = now
     -> Subscription becomes inactive, falls back to free plan
```

## Schema Changes

### Subscription Table — New Fields

| Field                    | Type        | Default     | Description                                                                      |
| ------------------------ | ----------- | ----------- | -------------------------------------------------------------------------------- |
| `payment_mode`           | varchar(20) | `"payment"` | `"payment"` (one-time) or `"subscription"` (recurring)                           |
| `stripe_subscription_id` | string?     | null        | Stripe Subscription ID. Set only for recurring subscriptions.                    |
| `cancel_at_period_end`   | bool        | false       | True when cancellation is pending. Subscription stays active until `expires_at`. |

### Plan Table — New Fields

| Field               | Type    | Default | Description                                                              |
| ------------------- | ------- | ------- | ------------------------------------------------------------------------ |
| `stripe_price_id`   | string? | null    | Cached Stripe Price ID. Created lazily on first subscription checkout.   |
| `stripe_product_id` | string? | null    | Cached Stripe Product ID. Created lazily on first subscription checkout. |

These are internal caching fields. Stripe requires a Price object for subscription checkouts. The system creates Stripe Product + Price on the first subscription-mode checkout for a plan and caches the IDs to avoid recreating them.

### User Table — New Fields

| Field                | Type    | Default | Description                                                            |
| -------------------- | ------- | ------- | ---------------------------------------------------------------------- |
| `stripe_customer_id` | string? | null    | Stripe Customer ID. Created on first subscription checkout and reused. |

All schema changes are handled automatically by GORM AutoMigrate. No manual migration needed.

## API Changes

### Checkout — `POST /billing/subscription/checkout`

**New field in request body**:

| Field          | Type   | Required | Default     | Description                                              |
| -------------- | ------ | -------- | ----------- | -------------------------------------------------------- |
| `payment_mode` | string | No       | `"payment"` | `"payment"` for one-time, `"subscription"` for recurring |

All other fields remain the same (`plan_id`, `scope`, `workspace_id`, `success_url`, `cancel_url`).

**Example — One-time checkout (default, backwards compatible)**:

```json
{
    "plan_id": "660e8400-e29b-41d4-a716-446655440001",
    "scope": "user",
    "success_url": "https://app.example.com/billing/success",
    "cancel_url": "https://app.example.com/billing/cancel"
}
```

**Example — Recurring subscription checkout**:

```json
{
    "plan_id": "660e8400-e29b-41d4-a716-446655440001",
    "scope": "user",
    "payment_mode": "subscription",
    "success_url": "https://app.example.com/billing/success",
    "cancel_url": "https://app.example.com/billing/cancel"
}
```

**Response** is the same for both modes:

```json
{
    "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_abc123...",
    "external_id": "cs_test_abc123"
}
```

The Stripe Checkout page UI differs based on mode — subscription mode shows "Subscribe to..." instead of "Pay...".

### Cancel Subscription — `DELETE /billing/subscription?id=<uuid>`

Behavior now depends on the subscription's `payment_mode`:

| Subscription Mode | Cancel Behavior                                                                                                                  | HTTP Response    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `"payment"`       | **Rejected.** Returns error: "one-time payment subscriptions cannot be cancelled — they expire naturally"                        | `500` with error |
| `"subscription"`  | Calls Stripe `cancel_at_period_end=true`. Sets `cancel_at_period_end=true` on the subscription. Stays active until `expires_at`. | `204 No Content` |

After cancelling a recurring subscription:

- The subscription remains active (no `cancelled_at` set yet)
- Stripe stops charging at the end of the current period
- When the period ends, Stripe fires `customer.subscription.deleted`
- The backend sets `cancelled_at` and the subscription becomes inactive

### Reactivate Subscription — `POST /billing/subscription/reactivate?id=<uuid>`

Reverses a pending cancellation, re-enabling auto-renewal. Only works when the subscription is in the "cancellation pending" state (`cancel_at_period_end=true`, `cancelled_at=null`).

| Condition                                        | Result                                                                        | HTTP Response    |
| ------------------------------------------------ | ----------------------------------------------------------------------------- | ---------------- |
| `cancel_at_period_end=true`, `cancelled_at=null` | Calls Stripe to set `cancel_at_period_end=false`. Clears the flag locally.    | `204 No Content` |
| `cancel_at_period_end=false`                     | Rejected: "subscription is not pending cancellation"                          | `500` with error |
| `cancelled_at` is set                            | Rejected: "subscription is already fully cancelled and cannot be reactivated" | `500` with error |
| `payment_mode="payment"`                         | Rejected: "only recurring subscriptions can be reactivated"                   | `500` with error |

### List Subscriptions — `GET /billing/subscription`

Response now includes the new fields:

```json
[
    {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "plan_id": "660e8400-e29b-41d4-a716-446655440001",
        "scope": "user",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "starts_at": "2026-02-17T12:00:00Z",
        "expires_at": "2026-03-19T12:00:00Z",
        "payment_provider": "stripe",
        "payment_external_id": "cs_test_abc123",
        "payment_mode": "payment",
        "plan": {
            "id": "660e8400-e29b-41d4-a716-446655440001",
            "name": "Pro",
            "slug": "pro",
            "throughput_limit": 5000,
            "window_seconds": 60
        },
        "created_at": "2026-02-17T12:00:00Z",
        "updated_at": "2026-02-17T12:00:00Z"
    },
    {
        "id": "990e8400-e29b-41d4-a716-446655440005",
        "plan_id": "660e8400-e29b-41d4-a716-446655440001",
        "scope": "workspace",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "workspace_id": "aa0e8400-e29b-41d4-a716-446655440006",
        "starts_at": "2026-02-17T12:00:00Z",
        "expires_at": "2026-03-19T12:00:00Z",
        "payment_provider": "stripe",
        "payment_external_id": "cs_test_def456",
        "payment_mode": "subscription",
        "stripe_subscription_id": "sub_1234567890",
        "plan": {
            "id": "660e8400-e29b-41d4-a716-446655440001",
            "name": "Pro",
            "slug": "pro",
            "throughput_limit": 5000,
            "window_seconds": 60
        },
        "created_at": "2026-02-17T12:00:00Z",
        "updated_at": "2026-02-17T12:00:00Z"
    }
]
```

Key fields for frontend logic:

| Field                    | When present                           | Frontend use                               |
| ------------------------ | -------------------------------------- | ------------------------------------------ |
| `payment_mode`           | Always                                 | Determines whether to show "Cancel" button |
| `stripe_subscription_id` | Only for `payment_mode="subscription"` | Indicates Stripe-managed recurring billing |
| `cancel_at_period_end`   | Always (bool, default false)           | Show "Cancellation pending" state          |
| `cancelled_at`           | After cancellation completes           | Show "Cancelled" status                    |

### Webhook — `POST /billing/webhook/stripe`

Now handles four event types (previously only one):

| Stripe Event                    | Action                                                                                           |
| ------------------------------- | ------------------------------------------------------------------------------------------------ |
| `checkout.session.completed`    | Creates subscription. Stores `payment_mode`, `stripe_subscription_id`, and `stripe_customer_id`. |
| `invoice.paid`                  | Extends `expires_at` to the new period end (renewal). Skips gracefully for the initial invoice.  |
| `customer.subscription.deleted` | Sets `cancelled_at` on the subscription (period ended after cancellation).                       |
| `customer.subscription.updated` | Syncs `cancel_at_period_end` from Stripe to the local subscription.                              |

### Unchanged APIs

These APIs are **not affected** by this change:

- `GET /billing/plan` — Plans don't have payment_mode; it's per-subscription
- `POST /billing/plan` — No new fields
- `PUT /billing/plan?id=<uuid>` — No new fields
- `DELETE /billing/plan?id=<uuid>` — No change
- `GET /billing/usage` — Reads active subscriptions regardless of payment mode
- `POST /billing/subscription/manual` — Manual subscriptions default to payment mode
- All endpoint weight APIs — No change
- Throughput middleware — No change

## Stripe Dashboard Configuration

After deploying, update the Stripe webhook endpoint to send these additional events (in addition to the existing `checkout.session.completed`):

- **`invoice.paid`** — Required for renewals
- **`customer.subscription.deleted`** — Required for cancellation completion
- **`customer.subscription.updated`** — Optional, for logging

## Stripe Objects Created Automatically

When the first subscription-mode checkout is initiated for a plan, the backend lazily creates:

1. **Stripe Product** — Named after the plan. `plan_id` stored in metadata.
2. **Stripe Price** — Recurring price with `interval=day`, `interval_count=duration_days`. Linked to the Product.

Both IDs are cached on the Plan entity (`stripe_price_id`, `stripe_product_id`) and reused for subsequent checkouts. One-time payment checkouts do not create these objects — they use ad-hoc pricing as before.

When a user's first subscription-mode checkout completes:

3. **Stripe Customer** — Created by Stripe during checkout (via `customer_email`). The Customer ID is persisted on the User entity (`stripe_customer_id`) and reused for subsequent subscription checkouts to avoid duplicate customers.

## Frontend Integration Guide

### Checkout Flow — Payment Mode Selection

When the user clicks "Subscribe" on a plan, the frontend should offer two options:

1. **One-time purchase** — Pay once, plan active for `duration_days`, then expires
2. **Recurring subscription** — Auto-renews every `duration_days`, cancel anytime

```
┌─────────────────────────────────────────┐
│  Pro Plan — $49/month                   │
│  5,000 requests/min for 30 days         │
│                                         │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │  Buy Once   │  │  Subscribe       │  │
│  │  $49        │  │  $49/month       │  │
│  └─────────────┘  └──────────────────┘  │
└─────────────────────────────────────────┘
```

- "Buy Once" sends `payment_mode: "payment"` (or omits it)
- "Subscribe" sends `payment_mode: "subscription"`

### Subscription Status Display

Use these fields from `GET /billing/subscription` to determine the display state:

```
if cancelled_at is set:
    Status = "Cancelled"
    Color  = gray

else if cancel_at_period_end is true:
    Status = "Cancellation pending — Active until {expires_at}"
    Color  = orange
    Show "Reactivate" button (POST /billing/subscription/reactivate?id=<id>)

else if payment_mode == "subscription" and expires_at is in the future:
    Status = "Active — Renews on {expires_at}"
    Color  = green
    Show "Cancel" button

else if payment_mode == "payment" and expires_at is in the future:
    Status = "Active — Expires on {expires_at}"
    Color  = green
    No "Cancel" button (one-time subscriptions expire naturally)

else if expires_at is in the past:
    Status = "Expired"
    Color  = gray
```

### Cancel Button Logic

Only show the "Cancel" button when **all** of these conditions are met:

1. `payment_mode == "subscription"`
2. `cancelled_at` is `null`
3. `cancel_at_period_end` is `false`
4. `expires_at` is in the future (subscription is still active)

After the user clicks "Cancel" and the `DELETE` request succeeds, show:

```
"Your subscription will remain active until {expires_at}.
 You will not be charged again."
```

The subscription stays in the list with status "Active" until the period ends. After `expires_at` passes (and the `customer.subscription.deleted` webhook fires), `cancelled_at` is set and the status changes to "Cancelled".

### Cancellation State Detection

The `cancel_at_period_end` field on the subscription entity allows the frontend to distinguish between three states without relying on optimistic UI:

| `cancelled_at` | `cancel_at_period_end` | State                                                                |
| -------------- | ---------------------- | -------------------------------------------------------------------- |
| `null`         | `false`                | Active and renewing                                                  |
| `null`         | `true`                 | Cancellation pending — active until `expires_at`, no further charges |
| set            | `true`/`false`         | Fully cancelled                                                      |

This field is set to `true` immediately when the user calls `DELETE /billing/subscription`, and is also synced from Stripe's `customer.subscription.updated` webhook for consistency.

### Checkout Flow Diagram

```
┌────────────────────────────────────────────────────────┐
│ Frontend                                                │
│                                                         │
│ 1. POST /billing/subscription/checkout                  │
│    { plan_id, scope, payment_mode, success_url, ... }   │
│                                                         │
│ 2. Receive { checkout_url, external_id }                │
│                                                         │
│ 3. window.location.href = checkout_url                  │
│    (Stripe Checkout page)                               │
│                                                         │
│ 4. User completes payment -> redirected to success_url  │
│                                                         │
│ 5. Poll GET /billing/subscription until new sub appears │
│    (webhook activates it asynchronously)                │
└────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────┐
│ Backend (async via webhook)                             │
│                                                         │
│ checkout.session.completed                              │
│   -> Create subscription with payment_mode              │
│   -> Store stripe_subscription_id (if subscription)     │
│   -> Store stripe_customer_id on user                   │
│                                                         │
│ invoice.paid (subscription mode only)                   │
│   -> Extend expires_at to new period end                │
│                                                         │
│ customer.subscription.deleted (after cancel)            │
│   -> Set cancelled_at on subscription                   │
└────────────────────────────────────────────────────────┘
```

## Backwards Compatibility

- All existing subscriptions have `payment_mode = "payment"` by default (GORM column default)
- Existing checkout calls without `payment_mode` in the request body default to `"payment"` — behavior is identical to before
- The `DELETE /billing/subscription` endpoint now rejects cancellation of one-time subscriptions. Previously it would set `cancelled_at` immediately. If the frontend currently shows a cancel button for all subscriptions, it should be updated to check `payment_mode` first.
- No changes to any non-billing APIs, middleware, or rate limiting behavior
- Stripe webhook endpoint must be updated to receive the new event types (see Stripe Dashboard Configuration)

## Files Changed

### wacraft-core

| File                                 | Change                                                                         |
| ------------------------------------ | ------------------------------------------------------------------------------ |
| `src/billing/model/payment-mode.go`  | **New.** `PaymentMode` type with `"payment"` and `"subscription"` constants.   |
| `src/billing/entity/plan.go`         | Added `stripe_price_id`, `stripe_product_id` fields.                           |
| `src/billing/entity/subscription.go` | Added `payment_mode`, `stripe_subscription_id`, `cancel_at_period_end` fields. |
| `src/billing/model/checkout.go`      | Added `payment_mode` to `CheckoutRequest`.                                     |
| `src/user/entity/user.go`            | Added `stripe_customer_id` field.                                              |

### wacraft-server

| File                                      | Change                                                                                                                                                                                                                                                                                          |
| ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/billing/service/payment/provider.go` | Expanded `WebhookEvent` with subscription fields. Updated `CreateCheckoutSession` signature. Added `ReactivateSubscription` to `Provider` interface.                                                                                                                                            |
| `src/billing/service/payment/stripe.go`   | Two checkout paths (payment/subscription). Lazy Stripe Product+Price creation. `CancelSubscription` and `ReactivateSubscription`. Webhook parsing for 4 event types.                                                                                                                            |
| `src/billing/service/subscription.go`     | `ActivateSubscription` stores payment mode + Stripe IDs. `CancelSubscription` only works for subscription mode and sets `cancel_at_period_end=true`. `ReactivateSubscription` reverses pending cancellation. New `RenewSubscription`, `MarkSubscriptionCancelled`, and `SyncCancelAtPeriodEnd`. |
| `src/billing/handler/webhook.go`          | Handles `checkout.session.completed`, `invoice.paid`, `customer.subscription.deleted`, `customer.subscription.updated` (syncs `cancel_at_period_end`).                                                                                                                                          |
| `src/billing/handler/subscription.go`     | Checkout passes `payment_mode`, user email, and Stripe customer ID. New `ReactivateSubscription` handler.                                                                                                                                                                                       |
