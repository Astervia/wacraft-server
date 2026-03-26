# Authentication Guide

This document covers the complete authentication system: token issuance, refresh, email verification, password reset, and invitation claiming.

---

## Table of Contents

1. [Registration](#1-registration)
2. [Email Verification](#2-email-verification)
3. [Login — Get Token](#3-login--get-token)
4. [Refresh Token](#4-refresh-token)
5. [Forgot Password](#5-forgot-password)
6. [Reset Password](#6-reset-password)
7. [Claim Invitation](#7-claim-invitation)
8. [Middleware Reference](#8-middleware-reference)
9. [Rate Limits](#9-rate-limits)
10. [Environment Variables](#10-environment-variables)

---

## 1. Registration

**`POST /auth/register`**

Creates a user, a personal workspace, and assigns all admin policies to that workspace.

### Request
```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "s3cr3t"
}
```

### Response `201`
```json
{
  "message": "User registered successfully",
  "userID": "uuid"
}
```

### Notes
- Passwords are hashed with bcrypt.
- If `REQUIRE_EMAIL_VERIFICATION=true` (default), a verification email is sent asynchronously and the user's `email_verified` flag starts as `false`.
- Emails are normalised (lower-cased + trimmed) before storage.

---

## 2. Email Verification

### Verify

**`GET /auth/verify-email?token=<token>`**

Tokens are delivered by email and expire after **24 hours**.

#### Response `200`
```json
{ "message": "Email verified successfully" }
```

- Returns `400` if the token has expired.
- Returns `200` (no-op) if the address is already verified.
- Verification and the `email_verified` flag update happen inside a transaction.

---

### Resend Verification Email

**`POST /auth/resend-verification`**

```json
{ "email": "alice@example.com" }
```

#### Response `200`
```json
{ "message": "If your email is registered, you will receive a verification email shortly" }
```

Always returns the same message regardless of whether the email exists, to prevent enumeration. Invalidates previous tokens before issuing a new one.

---

## 3. Login — Get Token

**`POST /user/oauth/token`**

Uses the OAuth 2.0 Resource Owner Password Credentials grant.

### Request
```json
{
  "grant_type": "password",
  "username": "alice@example.com",
  "password": "s3cr3t"
}
```

### Response `200`
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "token_type": "bearer",
  "expires_in": 3600
}
```

### Token Lifetimes
| Token | Lifetime |
|---|---|
| `access_token` | 1 hour |
| `refresh_token` | 7 days |

Both are signed JWTs using **HS256** with `JWT_SECRET`. Claims included: `sub` (user ID), `exp`, `iss` (`wacraft-server`).

---

## 4. Refresh Token

**`POST /user/oauth/token`** — same endpoint, different grant type.

### Request
```json
{
  "grant_type": "refresh_token",
  "refresh_token": "<7-day-jwt>"
}
```

### Response `200`
```json
{
  "access_token": "<new-1-hour-jwt>",
  "refresh_token": "<same-7-day-jwt>",
  "token_type": "bearer",
  "expires_in": 3600
}
```

- A new access token is issued; the refresh token **is not rotated** — the original 7-day token is returned unchanged.
- Returns `401` if the refresh token is expired or invalid.

---

## 5. Forgot Password

**`POST /auth/forgot-password`**

Initiates the password-reset flow. Always returns a generic success message to prevent email enumeration.

### Request
```json
{ "email": "alice@example.com" }
```

### Response `200`
```json
{ "message": "If your email is registered, you will receive a password reset link shortly" }
```

### What happens internally
1. Any existing unused reset tokens for the user are deleted.
2. A new `PasswordResetToken` record is created with a **1-hour expiry**.
3. An email is sent asynchronously containing the token.

---

## 6. Reset Password

**`POST /auth/reset-password`**

Completes the password-reset flow using the token from the email.

### Request
```json
{
  "token": "<reset-token>",
  "password": "newS3cr3t"
}
```

### Response `200`
```json
{ "message": "Password reset successfully" }
```

### Validation
- `400` — token not found, already used, or expired.
- On success: the new password is hashed and saved, and `usedAt` is stamped on the token (one-time use).

---

## 7. Claim Invitation

**`POST /auth/invitation/claim`**

Adds the authenticated user to a workspace using an invitation token.

**Requires:** `Authorization: Bearer <access_token>` + verified email.

### Request
```json
{ "token": "<invitation-token>" }
```

### Response `200`
```json
{
  "message": "Invitation claimed successfully",
  "workspaceID": "uuid"
}
```

### Validation
- `401` — not authenticated.
- `403` — email not verified.
- `400` — token not found, expired, already accepted, or the authenticated user's email does not match the invitation email (case-insensitive).
- `409` — user is already a member of the workspace.

On success the user is added as a workspace member and all associated policies are assigned.

---

## 8. Middleware Reference

### `UserMiddleware`

Validates the `Authorization: Bearer <jwt>` header, parses the access token, fetches the user from the database, and stores it in `c.Locals("user")`.

Errors:
- `400` — missing or malformed header.
- `401` — invalid/expired token or user not found.

### `EmailVerifiedMiddleware`

Must come after `UserMiddleware`. Returns `403` if `user.email_verified == false` and `REQUIRE_EMAIL_VERIFICATION=true`.

### `SuMiddleware`

Returns `403` unless the user's role is `admin`.

### `RoleMiddleware`

Parameterised version — accepts a list of allowed roles. Returns `403` if the user's role is not in the list.

### `TokenMiddleware`

Static bearer-token check against `AUTH_TOKEN` env var. Used for service-to-service auth. Skipped entirely when `AUTH_TOKEN` is empty.

### IP Filtering

Two optional middlewares controlled by env vars:

| Middleware | Env var | Behaviour |
|---|---|---|
| `IPAllowlistMiddleware` | `IP_ALLOWLIST` | Only listed CIDRs pass |
| `IPDenylistMiddleware` | `IP_DENYLIST` | Listed CIDRs are blocked |

Both accept comma-separated CIDR blocks. When the var is empty the middleware is skipped.

### WebSocket Auth

`src/auth/middleware/websocket/user.go` handles WebSocket upgrade requests. Accepts the token in the `Authorization` header **or** as a query parameter (for browser clients).

---

## 9. Rate Limits

All limits can be disabled globally with `RATE_LIMIT_ENABLED=false`.

| Endpoint | Default limit | Default window |
|---|---|---|
| `POST /auth/register` | 5 | 1 hour |
| `POST /user/oauth/token` | 10 | 15 minutes |
| `POST /auth/forgot-password` | 5 | 1 hour |
| `POST /auth/reset-password` | 10 | 1 hour |
| `POST /auth/resend-verification` | 5 | 1 hour |

Login is keyed by **IP + email**; all others by IP only.

On limit exceeded: `429` with `Retry-After` header. Limit state is exposed via `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` response headers.

---

## 10. Environment Variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | — | **Required.** HMAC secret for signing JWTs. |
| `REQUIRE_EMAIL_VERIFICATION` | `true` | If `false`, users are auto-verified on registration. |
| `AUTH_TOKEN` | — | Static bearer token for service-to-service auth. |
| `SMTP_HOST` | — | If empty, emails are logged instead of sent. |
| `SMTP_PORT` | — | |
| `SMTP_USER` | — | |
| `SMTP_PASSWORD` | — | |
| `SMTP_FROM` | — | |
| `RATE_LIMIT_ENABLED` | `true` | Master switch for all rate limiters. |
| `RATE_LIMIT_REGISTRATION` | `5` | |
| `RATE_LIMIT_REGISTRATION_WINDOW` | `1h` | |
| `RATE_LIMIT_LOGIN` | `10` | |
| `RATE_LIMIT_LOGIN_WINDOW` | `15m` | |
| `RATE_LIMIT_PASSWORD_RESET` | `5` | |
| `RATE_LIMIT_PASSWORD_RESET_WINDOW` | `1h` | |
| `RATE_LIMIT_RESET_PASSWORD` | `10` | |
| `RATE_LIMIT_RESET_PASSWORD_WINDOW` | `1h` | |
| `RATE_LIMIT_EMAIL_VERIFICATION` | `5` | |
| `RATE_LIMIT_EMAIL_VERIFICATION_WINDOW` | `1h` | |
| `IP_ALLOWLIST` | — | Comma-separated CIDR blocks. |
| `IP_DENYLIST` | — | Comma-separated CIDR blocks. |

---

## Key Source Files

| Concern | Path |
|---|---|
| Router | `src/auth/router/main.go` |
| Registration handler | `src/auth/handler/register.go` |
| Email verification handler | `src/auth/handler/verify-email.go` |
| Password reset handler | `src/auth/handler/password-reset.go` |
| Login / refresh handler | `src/user/handler/auth.go` |
| JWT service | `src/auth/service/auth.go` |
| Middleware | `src/auth/middleware/` |
| Invitation handler | `src/workspace/handler/invitation.go` |
