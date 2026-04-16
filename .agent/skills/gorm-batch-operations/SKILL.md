# Skill: GORM Batch Operations and Error Handling

Use this skill when implementing handlers or workers that process multiple items, or when optimizing GORM queries that involve related entities and distinct error handling requirements.

## Purpose

Prevent N+1 query bottlenecks and reduce transaction overhead in high-throughput areas (like webhooks or messaging handlers). Clarify when to use `.Preload()` versus explicit relation lookups to maintain precise HTTP error handling.

## Core Problem

- Iterating over an array to perform single-record database lookups or `db.Create(item)` calls leads to N+1 performance issues, drastically increasing database round-trips.
- Attempting to "optimize" by using GORM's `.Preload()` can conflate error handling. If a base entity is found but its related entity is not, `.Preload()` might fail silently or mask the distinction between "base entity missing" (e.g., 404) and "unauthorized for related entity" (e.g., 403).

## Preferred Pattern

### 1. Batch Lookups and Insertions

- Before iterating through a slice of payloads, execute a single database lookup to fetch all required prerequisite data (e.g., fetching all relevant Workspaces or Users using `IN (...)`).
- For creating multiple records, always use GORM's batch insertion capability by passing a slice to `db.Create(&slice)` instead of calling `db.Create(&item)` inside a loop.
- Implement explicit batch processing functions (e.g., `SendBatchByQuery(payloads []any)`) that manage the lookup-then-iterate-then-insert workflow.

### 2. Explicit Lookups vs. `.Preload()`

- Only use `.Preload()` when the success of the relation lookup does not dictate a distinct HTTP error response compared to the base lookup.
- If the base query (e.g., finding a message) and the relation lookup (e.g., verifying workspace ownership) require distinct HTTP error handling logic (like returning a 500 vs. a 403), do **not** combine them using `.Preload()`.
- Instead, execute explicit `.First()` queries sequentially. The performance gain of `.Preload()` (which executes as a separate query anyway) does not justify obfuscating the error handling.

## Workflow

1. **Review Loops:** Identify loops in handlers, services, or workers that contain database queries (`db.First`, `db.Find`, `db.Create`, `db.Save`).
2. **Extract Lookups:** Move database lookups outside the loop using `IN` clauses to fetch all necessary context at once.
3. **Batch Inserts:** Collect new entities into a slice and call `db.Create(&entities)` after the loop.
4. **Audit `.Preload()`:** Review instances of `.Preload()`. Ask: "If the base query succeeds but the preload fails or returns empty, do I need to return a specific error code like 403 instead of 404?" If yes, separate the queries.

## Anti-Patterns

- Executing `db.Create(&struct)` inside a `for` loop.
- Executing `db.First(&struct)` inside a `for` loop.
- Using `.Preload()` to fetch a security/ownership boundary entity and subsequently failing to return a precise 403/Unauthorized response because the error was swallowed or conflated with a base entity lookup failure.

## Done Criteria

- Loops contain zero database queries.
- Database insertions use slices.
- Handlers return correct, distinct HTTP error codes for missing base entities versus unauthorized relation lookups.