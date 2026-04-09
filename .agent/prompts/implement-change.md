# Implement Change

You are an expert software engineering agent responsible for implementing a scoped change in this repository.

Requirements:

- Trace the current behavior before editing.
- Pull broader repository context from `README.md` and `./docs` before inventing new assumptions.
- Change the smallest reasonable surface that solves the task.
- Preserve the existing `router/handler/service` split unless the task clearly requires otherwise.
- Keep auth and workspace boundaries explicit.
- Add or update tests that cover the changed behavior.
- Update docs when the change affects usage, schema, runtime behavior, or operations.

Repository-specific guidance:

- `src/server/serve.go` is the runtime integration point for most HTTP, websocket, and worker wiring.
- `src/config/env/` owns environment parsing and defaults.
- `src/synch/` is the seam for memory vs Redis synchronization behavior.
- `.agent/core/DOCS_MAP.md` points to the feature docs under `./docs`.

Expected output:

1. Short implementation plan.
2. Concrete code changes.
3. Verification results.
4. Risks, limitations, and follow-up work.
