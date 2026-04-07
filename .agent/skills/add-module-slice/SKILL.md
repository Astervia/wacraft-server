# Skill: Add Module Slice

Use this skill when adding a new feature area under `src/<domain>/`.

## Purpose

Extend the API surface without drifting away from the repository's existing
feature-slice structure.

## Default Assumption

A new domain should usually follow the existing pattern:

- `src/<domain>/router/`
- `src/<domain>/handler/`
- `src/<domain>/service/`

Add websocket or worker packages only when the feature actually needs them.

## Inspect First

- `src/server/serve.go`
- the closest existing domain package with similar behavior
- `src/auth/middleware/` and `src/workspace/` if the feature is workspace-scoped
- the nearest related docs under `docs/features/` or `docs/guides/`

## Workflow

1. Choose the owning domain rather than mixing the behavior into an unrelated slice.
2. Start from router ownership.
   - Define routes and shared middleware in `router/`.
3. Keep handlers thin.
   - Parse params and body.
   - validate inputs
   - call services
   - map outputs to HTTP responses
4. Keep domain logic in services.
   - business rules
   - external API calls
   - repository coordination
5. Wire the router in `src/server/serve.go`.
6. Add tests closest to the changed behavior.
7. Update docs if the feature changes public behavior, setup, or workflows.

## Design Rules

- Do not read env vars directly in handlers or services.
- Reuse auth and workspace middleware instead of re-implementing access checks.
- Keep request DTOs and persistence entities separate when the shapes diverge.
- Add migrations intentionally for schema changes.

## Done Criteria

- the new slice is wired through `src/server/serve.go`
- handler and service responsibilities stay separated
- auth and workspace boundaries are explicit
- tests cover the main path and at least one failure case
- docs are updated when the change is user-visible
