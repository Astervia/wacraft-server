# Stripe Setup Guide

This guide walks you through configuring Stripe for the wacraft-server billing system.

---

## Prerequisites

- A Stripe account ([stripe.com](https://stripe.com))
- The wacraft-server deployed or running locally
- Access to your server's environment variables

---

## 1. Get Your Stripe API Keys

1. Log in to the [Stripe Dashboard](https://dashboard.stripe.com)
2. Make sure you're in **Test mode** (toggle in the top-right) while setting up
3. Go to **Developers > API keys**
4. Copy your **Secret key** (starts with `sk_test_` in test mode, `sk_live_` in production)

> **Important**: Never expose your secret key in client-side code or commit it to version control. Only use it server-side.

Set the environment variable:

```bash
STRIPE_SECRET_KEY=sk_test_your_secret_key_here
```

### Test vs Live Keys

| Environment | Secret Key Prefix | Webhook Secret Prefix | Real Charges |
| ----------- | ----------------- | --------------------- | ------------ |
| Test        | `sk_test_`        | `whsec_`              | No           |
| Live        | `sk_live_`        | `whsec_`              | Yes          |

Always develop and test with **test mode** keys first. Switch to live keys only when deploying to production.

---

## 2. Create a Webhook Endpoint

The server listens for Stripe events at `POST /billing/webhook/stripe`. You need to register this URL in Stripe so it knows where to send payment notifications.

### Via Stripe Dashboard

1. Go to **Developers > Webhooks**
2. Click **Add endpoint**
3. Enter your endpoint URL:
    - Production: `https://your-api-domain.com/billing/webhook/stripe`
    - Local development: use a tunnel URL (see [Local Development](#5-local-development) below)
4. Under **Select events to listen to**, click **Select events**
5. Search for and select: **`checkout.session.completed`**
6. Click **Add endpoint**

### Copy the Webhook Signing Secret

After creating the endpoint:

1. Click on the newly created endpoint
2. Under **Signing secret**, click **Reveal**
3. Copy the secret (starts with `whsec_`)

Set the environment variable:

```bash
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_signing_secret_here
```

---

## 3. Configure Environment Variables

Add all billing-related variables to your `.env` file or deployment configuration:

```bash
# Enable billing enforcement
BILLING_ENABLED=true

# Stripe credentials
STRIPE_SECRET_KEY=sk_test_your_secret_key_here
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_signing_secret_here

# Default free plan settings (optional - these are the defaults)
DEFAULT_FREE_PLAN_THROUGHPUT=100
DEFAULT_FREE_PLAN_WINDOW=60
```

### Variable Reference

| Variable                       | Required | Default | Description                                                    |
| ------------------------------ | -------- | ------- | -------------------------------------------------------------- |
| `BILLING_ENABLED`              | No       | `false` | Set to `true` to enforce throughput limits                     |
| `STRIPE_SECRET_KEY`            | No       | -       | Stripe API secret key. Required for checkout.                  |
| `STRIPE_WEBHOOK_SECRET`        | No       | -       | Stripe webhook signing secret. Required for payment callbacks. |
| `DEFAULT_FREE_PLAN_THROUGHPUT` | No       | `100`   | Fallback free plan throughput limit (requests per window)      |
| `DEFAULT_FREE_PLAN_WINDOW`     | No       | `60`    | Fallback free plan window duration in seconds                  |

### Startup Behavior

- If `STRIPE_SECRET_KEY` is set, the Stripe payment provider is initialized automatically on server start.
- If it is not set, checkout endpoints will return `503 Service Unavailable`.
- Billing API routes (plans, subscriptions, usage) work regardless of whether Stripe is configured -- admins can manage plans before enabling payments.

---

## 4. Verify the Setup

### 4.1 Check Server Logs

Start the server and look for billing initialization messages. The server logs whether billing is enabled and whether the Stripe provider was initialized.

### 4.2 Create a Test Plan

Use the admin API to create a plan:

```bash
curl -X POST https://your-api-domain.com/billing/plan \
  -H "Authorization: Bearer <admin_token>" \
  -H "X-Workspace-ID: <workspace_id>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro",
    "slug": "pro",
    "throughput_limit": 5000,
    "window_seconds": 60,
    "duration_days": 30,
    "price_cents": 4900,
    "currency": "usd",
    "active": true
  }'
```

### 4.3 Test the Checkout Flow

Initiate a checkout:

```bash
curl -X POST https://your-api-domain.com/billing/subscription/checkout \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "<plan_uuid>",
    "scope": "user",
    "success_url": "https://your-app.com/billing/success",
    "cancel_url": "https://your-app.com/billing/cancel"
  }'
```

You should receive a response with a `checkout_url`. Open it in a browser to see the Stripe checkout page.

### 4.4 Use Stripe Test Cards

On the checkout page, use these test card numbers:

| Scenario            | Card Number           | Expiry          | CVC          |
| ------------------- | --------------------- | --------------- | ------------ |
| Successful payment  | `4242 4242 4242 4242` | Any future date | Any 3 digits |
| Declined            | `4000 0000 0000 0002` | Any future date | Any 3 digits |
| Requires auth (3DS) | `4000 0025 0000 3155` | Any future date | Any 3 digits |

Use any name, email, and billing address.

### 4.5 Verify Webhook Delivery

After a successful test payment:

1. Go to **Developers > Webhooks** in the Stripe Dashboard
2. Click on your endpoint
3. Check the **Webhook attempts** section
4. You should see a `checkout.session.completed` event with a `200` response status

If the webhook succeeded, the subscription should now be active. Verify:

```bash
curl https://your-api-domain.com/billing/subscription \
  -H "Authorization: Bearer <user_token>"
```

---

## 5. Local Development

Stripe can't send webhooks to `localhost`. Use the Stripe CLI to forward events to your local server.

### Install the Stripe CLI

```bash
# macOS
brew install stripe/stripe-cli/stripe

# Linux (Debian/Ubuntu)
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list
sudo apt update && sudo apt install stripe
```

### Log In

```bash
stripe login
```

This opens a browser for authentication. Follow the prompts.

### Forward Webhooks

```bash
stripe listen --forward-to localhost:3000/billing/webhook/stripe
```

The CLI will print a webhook signing secret:

```
> Ready! Your webhook signing secret is whsec_abc123... (^C to quit)
```

Use this secret as your `STRIPE_WEBHOOK_SECRET` for local development.

### Trigger Test Events

In a separate terminal, trigger a checkout completion event:

```bash
stripe trigger checkout.session.completed
```

Or complete a real test checkout flow in the browser -- the CLI will forward the webhook to your local server.

---

## 6. Going to Production

### Checklist

- [ ] Switch from test keys to live keys (`sk_live_`, `whsec_` from production endpoint)
- [ ] Create a new webhook endpoint in Stripe Dashboard pointing to your production URL
- [ ] Select the `checkout.session.completed` event
- [ ] Update environment variables with production values
- [ ] Set `BILLING_ENABLED=true`
- [ ] Create your production plans via the admin API
- [ ] Test a real checkout flow with a small amount

### Security Considerations

- Store `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` in a secrets manager (e.g., AWS Secrets Manager, Vault, Doppler) rather than plain `.env` files in production
- The webhook endpoint (`/billing/webhook/stripe`) does not use application authentication -- Stripe validates requests via the `Stripe-Signature` header and the webhook signing secret
- Ensure your production server uses HTTPS -- Stripe requires it for webhook endpoints
- Restrict access to billing admin endpoints (`billing.admin` policy) to trusted operators

---

## Troubleshooting

### Checkout returns 503

The Stripe provider is not initialized. Verify that `STRIPE_SECRET_KEY` is set and the server has been restarted.

### Webhook returns 400

The webhook signature validation failed. Common causes:

- `STRIPE_WEBHOOK_SECRET` is wrong or missing
- You're using the dashboard's signing secret with the CLI (or vice versa) -- they are different
- The request body was modified by a proxy or middleware before reaching the handler

### Subscription not activated after payment

1. Check the Stripe Dashboard under **Developers > Webhooks** for failed delivery attempts
2. Check your server logs for errors in the webhook handler
3. Verify the plan ID in the checkout metadata matches an existing plan in your database

### Webhook events arriving but not processed

Only `checkout.session.completed` is handled. Other event types are acknowledged with `200 OK` but no action is taken.
