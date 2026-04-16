# Agent Workspace

This directory holds repo-local assets for agentic coding workflows.

## Layout

- `core/` - shared repository context, conventions, and task templates
- `prompts/` - reusable prompts for common engineering tasks
- `skills/` - focused workflows for repeatable work in this codebase
- `tools/` - helper scripts and inspection utilities

Keep this tree repo-local so prompts and automation stay aligned with the code.

## Default Usage

Start with the generic assets in `core/` and `prompts/` for most work:

- `core/CONVENTIONS.md` - repository-wide implementation rules
- `core/ARCHITECTURE_MAP.md` - workspace and runtime orientation
- `core/DOCS_MAP.md` - where to find broader reference material in `./docs`
- `core/TASK_TEMPLATE.md` - default template for scoped coding tasks
- `prompts/implement-change.md` - prompt for feature or refactor work
- `prompts/review-change.md` - prompt for code review work

Use `README.md` and `./docs` as the main source of broader repository context.
The files under `.agent/` should stay concise and point to the relevant material
instead of duplicating feature documentation.

## Repo-Specific Assets

Current examples:

- `skills/go-update-endpoint/SKILL.md` - workflow for safe GORM update handlers that must preserve zero values
- `skills/add-module-slice/SKILL.md` - workflow for adding a new `handler/router/service` feature slice
- `skills/gorm-batch-operations/SKILL.md` - workflow for avoiding N+1 queries and handling errors correctly with `.Preload()`
- `tools/inspect-server-surface.sh` - quick trace for server boot, routing, config, sync, and worker entry points
