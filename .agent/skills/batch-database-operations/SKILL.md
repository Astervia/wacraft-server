# Skill: Batch Database Operations

Use this skill when processing multiple items, performing bulk insertions, or handling arrays of webhooks/messages to prevent N+1 query bottlenecks and excessive database load.

## Purpose

Optimize database interactions by grouping lookups and insertions into single queries rather than executing individual queries sequentially in a loop.

## Core Problem

Iterating through an array of items and calling `.First()`, `.Find()`, or `.Create()` inside the loop forces the application to execute a separate database query for every item (N+1 queries). This severely impacts performance, consumes database connections, and creates unnecessary latency, especially in webhook handlers or message processing queues.

## Preferred Pattern

- **For Insertions:** Append all instances to a slice, then pass the slice pointer to GORM's `.Create(&slice)` outside the loop.
- **For Lookups:** Extract the required keys or IDs into a slice, then perform a single `.Where("id IN ?", keys).Find(&results)` before processing.

## Workflow

1. Identify loops that contain database queries (e.g., in `src/<domain>/handler/` or `src/<domain>/worker/`).
2. **Batch Read:** If the loop requires existing data, extract the identifiers into an array before the loop. Perform one query using `IN (?)`, map the results by ID (e.g., `map[string]Entity`), and then use the map for constant-time lookups inside the loop.
3. **Batch Write:** If the loop creates new records, instantiate the objects inside the loop and append them to a slice. After the loop completes, use `db.Create(&slice)` to perform a single multi-insert SQL statement.
4. If batch sizes are excessively large (e.g., > 1000 records), consider chunking the slice and performing batch creates in chunks using GORM's `CreateInBatches`.

## Inspect First

- Handlers processing array inputs (e.g., `[]Payload`).
- Webhook processors in `src/webhook/worker/` or message handlers.
- Existing bulk sync functions.

## Anti-Patterns

- Calling `db.Create(&singleItem)` inside a `for` loop.
- Calling `db.Where("id = ?", item.ID).First(&entity)` inside a `for` loop.

## Done Criteria

- Database queries are hoisted outside of loops.
- Bulk insertions are handled via slice-based `db.Create`.
- Related lookup data is fetched in one query and indexed locally via maps.
- Tests continue to pass and correctly handle multiple items at once.
