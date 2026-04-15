# Skill: GORM Batch Operations

Use this skill when processing multiple operations or creating multiple records in a single request, webhook, or background worker task.

## Purpose

Prevent N+1 query bottlenecks and reduce database round-trips by executing batch lookups and bulk insertions instead of querying or saving row-by-row in a loop.

## Core Problem

Iterating through a list of items and calling `db.First()` or `db.Create()` on each item results in an N+1 query pattern.
As the number of items grows, the latency increases linearly, consuming database connections and risking timeouts.
This is particularly dangerous in webhook handlers, message processors, and data import tools.

## Preferred Pattern

Always structure operations to perform bulk queries first, and then perform bulk insertions or updates using `db.Create(&slice)`.

## Workflow

1. Extract all required identifiers (e.g., IDs, external keys) from the incoming payloads into a slice.
2. Execute exactly one database query using the gathered identifiers (`db.Where("id IN ?", ids).Find(&results)`) to retrieve all necessary existing state.
3. Map the results into memory (e.g., using a map grouped by ID) for fast, O(1) lookups during processing.
4. Iterate through the payloads, applying business logic and constructing an array of new or updated entity structs in memory.
5. Persist all constructed entities in a single database operation using `db.Create(&slice)` for insertions. For updates, consider building a bulk update or executing batch processing within a single transaction depending on project conventions.

## Inspect First

- The incoming payload or slice of data that needs processing.
- The `src/<domain>/service/` or `src/webhook/worker/` handlers.
- Existing repository batch functions if any.

## Anti-Patterns

- Calling `.First()`, `.Find()`, or `.Create()` inside a `for` loop.
- Fetching relations one-by-one instead of using `.Preload()` or bulk `.Find()`.
- Wrapping individual `.Create()` calls in a transaction while still looping, which mitigates connection overhead but does not reduce query parsing and round-trips.

## Done Criteria

- The endpoint or worker completes processing with a fixed (O(1)) number of database queries regardless of payload size.
- A single `db.Create(&slice)` is used to insert all necessary records.
- Tests verify correct batch insertion of multiple items in a single operation.
