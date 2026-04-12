# Agent Conventions

These conventions are specific to this repository and should guide agent work.

## General Rules

1. Preserve the existing feature-slice structure unless the task clearly justifies a broader refactor.
2. Prefer the narrowest viable change over cross-module rewrites.
3. Keep route registration in `router/`, request orchestration in `handler/`, and business logic in `service/`.
4. Keep environment parsing in `src/config/env/` and avoid scattering `os.Getenv` calls into feature code.
5. Treat workspace scoping and auth middleware as security boundaries, not convenience layers.
6. Update migrations intentionally when persistence shape changes; do not rely on implicit auto-migration behavior alone.
7. Keep memory and Redis sync modes behaviorally aligned unless a task explicitly targets one backend.
8. Do not leave temporary `.go` files containing `main` packages or functions in the repository, as they cause GitHub CI (CodeQL SAST scans and builds) to fail.

## Repo Design Bias

The repository is organized around:

- Go feature slices under `src/<domain>/`
- Fiber routers and handlers at the API edge
- service packages for domain behavior
- GORM-backed persistence with explicit migration directories
- background workers for webhook delivery and campaign scheduling
- optional distributed coordination through Redis-backed sync

When changing code, look for the existing layer that should own the behavior:

- `src/config/env/` for config parsing and defaults
- `src/auth/` and `src/workspace/` for access control and tenant boundaries
- `src/<domain>/router/` for URL and middleware wiring
- `src/<domain>/handler/` for request validation and response mapping
- `src/<domain>/service/` for domain logic and integrations
- `src/websocket/` and `src/*/websocket-router/` for websocket entry points
- `src/webhook/worker/` and `src/campaign/worker/` for long-running background processes
- `src/database/` for connection setup and migrations
- `docs/features/` and `docs/guides/` for durable feature context

## Change Strategy

For most tasks:

1. Trace the current request or worker path end to end before editing.
2. Identify the owning feature slice and the narrowest edit surface.
3. Preserve existing API contracts unless the task explicitly requires a break.
4. Keep request validation, auth, and workspace checks explicit.
5. Use `README.md` and `./docs` for broader project context instead of recreating it in `.agent/`.
6. Update tests and docs when behavior, config, schema, or operator workflow changes.

## Testing Expectations

At minimum, changes should include:

- focused package tests for the changed behavior when practical
- `go test ./... -v -p 1` for normal regression coverage (use `-p 1` to execute packages sequentially, preventing race conditions and unique constraint violations caused by concurrent GORM `AutoMigrate` calls)
- `make test-memory` or `make test-distributed` when sync, database, websocket, or worker behavior changes
- an explicit note of any verification gap when external dependencies make full validation impractical

## Documentation Expectations

Update the closest user-facing or architecture-facing document when behavior changes:

- `README.md` for setup, build, runtime, or migration workflow changes
- `docs/features/auth/` for auth-related behavior
- `docs/features/billing/` for billing, throughput, or Stripe behavior
- `docs/features/horizontal_scaling/` for distributed sync or multi-instance behavior
- `docs/features/multi_tenant/` for tenant and workspace semantics
- `docs/features/webhook/` for webhook system behavior
- `docs/guides/` for operator-facing setup or testing procedures

When looking for context before editing, start with `.agent/core/DOCS_MAP.md`.
