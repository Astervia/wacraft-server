# Firewall Implementation Summary

This document describes the auth firewall: IP-based access control and rate limiting across auth and login routes, fully configurable via environment variables.

---

## Overview

The firewall has two independent layers:

1. **IP filter** — allowlist and denylist by CIDR, applied globally to all routes.
2. **Rate limiting** — per-route limiters keyed by IP (or IP + email for login), applied to all auth and login endpoints.

Both layers are controlled by environment variables and are zero-cost passthroughs when not configured.

---

## Environment Variables

### IP Filter

| Variable | Default | Description |
|---|---|---|
| `IP_ALLOWLIST` | `""` | Comma-separated CIDRs. If non-empty, any IP not matched is rejected with `403`. |
| `IP_DENYLIST` | `""` | Comma-separated CIDRs. Any IP matched is rejected with `403`. |

Both accept standard CIDR notation (e.g., `10.0.0.0/8`, `192.168.1.5/32`). Multiple entries are comma-separated: `10.0.0.0/8,172.16.0.0/12`.

When a variable is empty, its middleware is a no-op passthrough with no overhead.

### Rate Limiting

| Variable | Default | Description |
|---|---|---|
| `RATE_LIMIT_ENABLED` | `true` | Set to `false` to disable all rate limiting. |
| `RATE_LIMIT_REGISTRATION` | `5` | Max registration attempts per window per IP. |
| `RATE_LIMIT_REGISTRATION_WINDOW` | `1h` | Window duration for registration. |
| `RATE_LIMIT_LOGIN` | `10` | Max login attempts per window per IP+email. |
| `RATE_LIMIT_LOGIN_WINDOW` | `15m` | Window duration for login. |
| `RATE_LIMIT_PASSWORD_RESET` | `5` | Max forgot-password requests per window per IP. |
| `RATE_LIMIT_PASSWORD_RESET_WINDOW` | `1h` | Window duration for forgot-password. |
| `RATE_LIMIT_EMAIL_VERIFICATION` | `5` | Max resend-verification requests per window per IP. |
| `RATE_LIMIT_EMAIL_VERIFICATION_WINDOW` | `1h` | Window duration for email verification. |
| `RATE_LIMIT_RESET_PASSWORD` | `10` | Max reset-password submissions per window per IP. |
| `RATE_LIMIT_RESET_PASSWORD_WINDOW` | `1h` | Window duration for reset-password. |

Window values use Go duration format: `30s`, `15m`, `1h`, `24h`, etc. Invalid or empty values fall back to the default.

### Example `.env`

```env
# IP filter
IP_DENYLIST=198.51.100.0/24,203.0.113.5/32

# Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_LOGIN=5
RATE_LIMIT_LOGIN_WINDOW=10m
RATE_LIMIT_REGISTRATION=3
RATE_LIMIT_REGISTRATION_WINDOW=1h
```

---

## Protected Routes

| Route | Limiter | Key | Default limit |
|---|---|---|---|
| `POST /auth/register` | `RegistrationRateLimiter` | IP | 5 / 1h |
| `POST /user/oauth/token` | `LoginRateLimiter` | IP + email | 10 / 15m |
| `POST /auth/forgot-password` | `PasswordResetRateLimiter` | IP | 5 / 1h |
| `POST /auth/reset-password` | `ResetPasswordRateLimiter` | IP | 10 / 1h |
| `POST /auth/resend-verification` | `EmailVerificationRateLimiter` | IP | 5 / 1h |

---

## Responses

### Rate limit exceeded — `429 Too Many Requests`

```json
{
    "error": "Too many login attempts",
    "message": "Please try again later"
}
```

Response headers included:

| Header | Description |
|---|---|
| `Retry-After` | Seconds until the rate limit window resets. Derived from `X-RateLimit-Reset` when available; falls back to the full window duration. |
| `X-RateLimit-Limit` | Maximum requests allowed in the window. |
| `X-RateLimit-Remaining` | Requests remaining in the current window. |
| `X-RateLimit-Reset` | Unix timestamp when the window resets. |

`Retry-After` is also exposed in the CORS `ExposeHeaders` policy so browser clients can read it.

### IP denied — `403 Forbidden`

Allowlist rejection:
```json
{ "error": "Access denied: unauthorized IP" }
```

Denylist rejection:
```json
{ "error": "Access denied" }
```

---

## Architecture

### File layout

```
src/
  config/env/
    firewall.go              # All firewall env vars and parsers
    registration.go          # AllowRegistration, RequireEmailVerification
  auth/middleware/
    rate-limit.go            # Rate limiter factory + all limiter instances
    ip-filter.go             # IPAllowlistMiddleware, IPDenylistMiddleware,
                             # NewAllowlistMiddleware, NewDenylistMiddleware
  server/
    serve.go                 # Global IP filter wiring (before all routes)
  auth/router/
    main.go                  # Auth route wiring
  user/router/
    auth.go                  # Login route wiring (POST /user/oauth/token)
```

### IP filter — global application

Both IP middlewares are registered on the Fiber app root, before any route group, so they apply to every request:

```go
app.Use(auth_middleware.NewDenylistMiddleware())
app.Use(auth_middleware.NewAllowlistMiddleware())
```

Evaluation order: denylist is checked first, then allowlist.

### Rate limiter — factory pattern

All rate limiters are package-level vars initialized at startup via `newRateLimiter()`. When `RATE_LIMIT_ENABLED=false`, the factory returns a `c.Next()` passthrough instead of constructing a limiter, so no in-memory state is allocated.

```go
func newRateLimiter(keyPrefix string, max int, window time.Duration, errorMsg string) fiber.Handler {
    if !env.RateLimitEnabled {
        return func(c *fiber.Ctx) error { return c.Next() }
    }
    // ... Fiber limiter setup
}
```

The login limiter uses a composite key (`IP + email`) to prevent credential stuffing even when the attacker rotates emails but keeps the same IP — and vice versa.

### `Retry-After` precision

The limiter sets `X-RateLimit-Reset` (Unix timestamp) before invoking `LimitReached`. The `retryAfterSeconds` helper reads that header to compute the exact remaining seconds rather than returning the full window blindly:

```go
func retryAfterSeconds(c *fiber.Ctx, window time.Duration) string {
    if reset := c.GetRespHeader("X-RateLimit-Reset"); reset != "" {
        if resetTime, err := strconv.ParseInt(reset, 10, 64); err == nil {
            if diff := resetTime - time.Now().Unix(); diff > 0 {
                return strconv.FormatInt(diff, 10)
            }
        }
    }
    return strconv.Itoa(int(window.Seconds()))
}
```

---

## Storage

Rate limiter state is held **in-memory** using Fiber's default limiter storage. This is sufficient for single-instance deployments. For multi-instance deployments, a shared storage backend (e.g., Redis via `github.com/gofiber/storage/redis`) should be plugged into the limiter `Config.Storage` field — no logic changes required.

---

## Frontend Recommendations

- Read `Retry-After` from `429` responses and surface a countdown ("Try again in 42 seconds") instead of a generic error.
- Disable the submit button for the duration indicated by `Retry-After` to prevent repeated user attempts.
- For login, avoid pre-fetching or probing the email field in ways that would increment the login rate limit.
