# Multi-Currency Plan Pricing - Implementation Summary

This document describes the multi-currency pricing system for billing plans. Previously, each plan had a single `price_cents` and `currency` field. Now, a plan can have multiple `PlanPrice` entries — one per currency. The frontend must use the new `plan_price` endpoints to manage prices and pass a `currency` field when initiating checkout.

---

## Data Model Changes

### `Plan` entity — removed fields

The following fields were **removed** from the `Plan` entity:

| Removed field       | Moved to    |
| ------------------- | ----------- |
| `price_cents`       | `PlanPrice` |
| `currency`          | `PlanPrice` |
| `stripe_price_id`   | `PlanPrice` |
| `stripe_product_id` | `PlanPrice` |

The `Plan` response now includes a `prices` array (see below).

### `PlanPrice` entity — new

```json
{
    "id": "uuid",
    "plan_id": "uuid",
    "currency": "usd",
    "price_cents": 2999,
    "is_default": true,
    "stripe_price_id": "price_xxx",
    "stripe_product_id": "prod_xxx",
    "created_at": "...",
    "updated_at": "..."
}
```

**Rules:**

- `(plan_id, currency)` is unique — one price per currency per plan.
- Exactly **one** `PlanPrice` per plan should have `is_default: true`. That entry is used when no `currency` is specified at checkout.
- `stripe_price_id` / `stripe_product_id` are server-managed (Stripe caching). Do not send them on create/update.

### Updated `Plan` response

`GET /billing/plan/` now includes a `prices` array on each plan:

```json
{
  "id": "uuid",
  "name": "Pro",
  "slug": "pro",
  "throughput_limit": 1000,
  "window_seconds": 60,
  "duration_days": 30,
  "is_default": false,
  "is_custom": false,
  "active": true,
  "prices": [
    { "id": "...", "currency": "usd", "price_cents": 2999, "is_default": true, ... },
    { "id": "...", "currency": "brl", "price_cents": 14900, "is_default": false, ... }
  ]
}
```

### Updated `CreatePlan` / `UpdatePlan` — removed fields

`POST /billing/plan/` and `PUT /billing/plan/` no longer accept `price_cents` or `currency`. Create the plan first, then add prices via `POST /billing/plan/:plan_id/price/`.

---

## Plan Price Endpoints

All price endpoints are nested under the plan: `/billing/plan/:plan_id/price/`.

### List prices

```
GET /billing/plan/:plan_id/price/
```

Auth: any authenticated user (email verified).

Query params: standard pagination (`page`, `page_size`, `order`, date filters).

Response: array of `PlanPrice` objects.

---

### Create a price (admin only)

```
POST /billing/plan/:plan_id/price/
X-Workspace-ID: <workspace_id>   (requires billing.admin policy)
```

Body:

```json
{
    "currency": "brl",
    "price_cents": 14900,
    "is_default": false
}
```

- `currency` — required. ISO 4217 lowercase (e.g. `"usd"`, `"brl"`, `"eur"`).
- `price_cents` — required. Price in the smallest currency unit (e.g. cents). `0` is valid for free tiers.
- `is_default` — optional. If `true`, the server automatically unsets the previous default for this plan.

Response `201`: created `PlanPrice` object.

---

### Update a price (admin only)

```
PUT /billing/plan/:plan_id/price/?id=<price_id>
X-Workspace-ID: <workspace_id>
```

Body (all fields optional):

```json
{
    "price_cents": 19900,
    "is_default": true
}
```

- Setting `is_default: true` automatically unsets the previous default for this plan.
- `currency` cannot be changed after creation — delete and recreate if needed.

Response `200`: updated `PlanPrice` object.

---

### Delete a price (admin only)

```
DELETE /billing/plan/:plan_id/price/?id=<price_id>
X-Workspace-ID: <workspace_id>
```

Response `204`.

---

## Checkout Changes

### Updated `CheckoutRequest`

A new optional `currency` field was added to the checkout request body:

```json
{
    "plan_id": "uuid",
    "currency": "brl",
    "scope": "workspace",
    "workspace_id": "uuid",
    "payment_mode": "payment",
    "success_url": "https://...",
    "cancel_url": "https://..."
}
```

**Resolution logic:**

- If `currency` is provided → uses the `PlanPrice` matching that currency. Returns `400` if none exists.
- If `currency` is omitted → uses the `PlanPrice` with `is_default: true`. Returns `400` if no default is set.

**Frontend recommendation:** show the user available currencies from the `plan.prices` array and pass the selected `currency` to checkout.

---

## Migration Notes (server-side, no frontend action needed)

The server runs a goose migration (`20260308000001`) on startup that automatically copies each plan's old `currency`, `price_cents`, `stripe_price_id`, and `stripe_product_id` values into the new `plan_prices` table with `is_default: true`. Existing plans will have exactly one price entry after migration.

---

## Typical Frontend Flow

### Displaying plans and prices

1. `GET /billing/plan/` → iterate `plan.prices` to show all available currencies and amounts.
2. Let the user pick a currency (or default to `plan.prices.find(p => p.is_default)`).

### Admin: setting up pricing for a new plan

1. `POST /billing/plan/` — create the plan (no price fields).
2. `POST /billing/plan/:plan_id/price/` with `is_default: true` and the first currency (e.g. USD).
3. `POST /billing/plan/:plan_id/price/` with additional currencies (e.g. BRL) as needed.

### Admin: changing a price

```
PUT /billing/plan/:plan_id/price/?id=<price_id>
{ "price_cents": 19900 }
```

### Admin: changing the default currency for a plan

```
PUT /billing/plan/:plan_id/price/?id=<brl_price_id>
{ "is_default": true }
```

The previous default is automatically unset.

### Admin: removing a currency

```
DELETE /billing/plan/:plan_id/price/?id=<price_id>
```

### User: checkout with a specific currency

```
POST /billing/subscription/checkout
{
  "plan_id": "...",
  "currency": "brl",
  "scope": "workspace",
  "workspace_id": "...",
  "payment_mode": "payment",
  "success_url": "...",
  "cancel_url": "..."
}
```
