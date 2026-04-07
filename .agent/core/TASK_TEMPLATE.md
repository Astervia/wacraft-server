# Task Template

Use this template for scoped agent work in this repository.

## Goal

State the task in one sentence.

## Constraints

- Preserve existing feature-slice boundaries unless the task justifies a change.
- Avoid unrelated refactors.
- Keep auth, workspace scoping, and API behavior backward compatible where practical.
- Keep memory and Redis sync behavior aligned unless the task targets one backend specifically.

## Steps

1. Trace the current request, worker, or startup path.
2. Identify the owning module and the narrowest edit surface.
3. Implement the change and any required routing or wiring.
4. Add or update tests and verification steps.
5. Update docs when behavior, config, schema, or workflows changed.

## Deliverables

- code changes
- tests or explicit testing gaps
- docs updates
- residual risks
