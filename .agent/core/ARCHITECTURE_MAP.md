# Architecture Map

This map is the default orientation for agent work in this repository.

## Workspace Entry Points

- `README.md`
  - local development and Docker workflow
  - migration commands
  - Swagger generation
- `go.mod`
  - module identity
  - major runtime dependencies
- `main.go`
  - CLI migration commands
  - import-driven boot sequence
- `docs/`
  - feature implementation notes and testing guides

## Runtime Spine

- `main.go`
  - CLI command dispatch for `migrate:*`
  - default server startup through package init side effects
- `src/config/main.go`
  - loads env configuration and sync backend initialization
- `src/server/main.go`
  - guards startup during migration commands
- `src/server/serve.go`
  - Fiber app construction
  - middleware, routers, websocket wiring, and background workers
- `src/database/connect.go`
  - Postgres connection and pool configuration
- `src/database/migrate/`
  - auto-migration boot path
- `src/synch/`
  - memory vs Redis synchronization backend wiring

## Major Subsystems

- `src/config/env/`
  - environment variable parsing and runtime defaults
- `src/auth/`
  - registration, login, token middleware, email verification, and access checks
- `src/workspace/`
  - workspace-scoped routing and multitenant boundaries
- `src/user/`, `src/contact/`, `src/messaging-product/`
  - core CRUD-style API surfaces
- `src/message/`, `src/status/`
  - messaging, status tracking, websocket updates, and sync integration
- `src/webhook/`, `src/webhook-in/`
  - outbound delivery, inbound webhook ingestion, verification, and workers
- `src/campaign/`
  - campaign sending, scheduling, websocket progress, and scheduler worker
- `src/billing/`
  - plans, subscriptions, usage, throughput limits, and Stripe integration
- `src/phone-config/`
  - per-workspace WhatsApp phone registration and verification flows
- `src/media/`
  - WhatsApp media endpoints
- `src/database/migrations-before/`, `src/database/migrations/`
  - pre- and post-auto-migration schema changes

## Common Task Routing

- Env or startup behavior change:
  start in `src/config/env/`, `src/config/main.go`, and `src/server/serve.go`.
- HTTP endpoint change:
  start in the owning slice's `router/`, then trace into `handler/` and `service/`.
- Workspace or auth boundary change:
  start in `src/auth/middleware/`, `src/auth/service/`, and `src/workspace/`.
- Message or status sync change:
  start in `src/message/service/`, `src/status/service/`, and `src/synch/`.
- Webhook ingestion or delivery change:
  start in `src/webhook-in/`, `src/webhook/`, and `src/webhook/worker/`.
- Campaign scheduling change:
  start in `src/campaign/service/` and `src/campaign/worker/`.
- Billing or throughput change:
  start in `src/billing/` and then inspect middleware integration.
- Schema or persistence change:
  inspect entity models, repository usage, and both migration directories.

## Validation Map

- Unit and package tests:
  `go test ./... -v`
- Race coverage without Redis:
  `make test-memory`
- Distributed sync coverage with Redis and Postgres:
  `make test-distributed`
- Build and Swagger generation:
  `make build`

Use `core/DOCS_MAP.md` before adding more architecture summary under `.agent/`.
