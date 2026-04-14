# Skill: GORM Batch Insertions

Use this skill when persisting multiple records to the database to optimize performance.

## Purpose

Optimize database interactions by reducing round-trips and transaction overhead during multi-record creation.

## Core Problem

Iterating through a slice of items and calling `db.Create(&entity)` in a loop executes a separate `INSERT` statement for each record. This results in excessive database round-trips, increased transaction setup/teardown overhead, and significant performance bottlenecks, particularly in high-throughput endpoints, webhook handlers, or data import tasks.

## Preferred Pattern

Use GORM's built-in batch insertion support by passing a slice of entities directly to `db.Create(&slice)`.

## Workflow

1. Identify areas where `db.Create` or `db.Save` is called iteratively inside a `for` loop.
2. Refactor the code to accumulate the entities into a slice (e.g., `entities := make([]Model, 0, len(items))`).
3. Apply any required pre-creation logic (e.g., setting workspace IDs, generating UUIDs) to each entity before appending it to the slice.
4. Execute the batch insertion after the loop by calling `db.Create(&entities)`.
5. If the slice might contain thousands of records, consider using `db.CreateInBatches(&entities, batchSize)` to prevent exceeding the database's maximum parameter limit per query.

## Inspect First

- Background workers (e.g., `src/webhook/worker/` or `src/campaign/worker/`) that generate multiple records at once.
- Endpoints designed to accept arrays of items.
- Migration or seeding scripts under `src/database/`.

## Anti-Patterns

- Calling `db.Create` or `db.Save` inside a `for` loop.
- Manually constructing complex `INSERT INTO ... VALUES (...)` SQL strings, as GORM's `Create` natively handles this securely and safely.

## Done Criteria

- Iterative single-record insertions are replaced with slice-based batch insertions.
- The number of generated SQL `INSERT` queries is significantly reduced.
- Pre-creation entity logic correctly applies to all elements in the batch.
