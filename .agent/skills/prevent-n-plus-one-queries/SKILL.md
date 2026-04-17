# Skill: Prevent N+1 Queries and use GORM Batch Operations

Use this skill when processing multiple records, payloads, or webhooks that require database lookups or insertions.

## Purpose

Prevent N+1 query bottlenecks and excessive transaction overhead by replacing loop-based database operations with batch processing and optimized GORM batch insertions.

## Core Problem

In high-throughput areas like webhook handlers or messaging components, iterating over a list of items and executing a single database lookup or insertion for each item (N+1 queries) severely degrades performance. This causes excessive database round-trips and transaction setup/teardown overhead.

## Preferred Pattern

- **Batch Lookups**: Execute the required database lookup exactly once before iterating through payloads to enqueue operations. Fetch all necessary related data in a single query (e.g., using `WHERE id IN (...)`).
- **Batch Insertions**: Use GORM's `db.Create(&slice)` to insert multiple records at once, reducing round-trips and transaction overhead compared to iterative single-record creation in a loop.

## Workflow

1. Identify areas where database queries (lookups or inserts) are happening inside a `for` loop over a collection of payloads or models.
2. **For lookups:**
   - Extract the keys (e.g., IDs) from the payloads into a slice.
   - Perform a single query using `Where("id IN ?", keys)` to fetch all required records.
   - Convert the fetched records into a map (e.g., `map[ID]Record`) for fast O(1) lookups inside the processing loop.
3. **For insertions:**
   - Construct a slice of models representing the new records inside the loop (but do not call `db.Create` yet).
   - After the loop, call `db.Create(&slice)` to insert them in a single batch.

## Inspect First

- `src/webhook/worker/` or messaging handlers where batch processing functions (e.g., `SendBatchByQuery`) are used or should be used.
- Repository layers where loop-based `db.First` or `db.Create` might be hidden inside a `.Save()` or `.Add()` method that is called repeatedly.

## Anti-Patterns

- Calling `db.First` or `db.Find` inside a `for` loop.
- Calling `db.Create` or `db.Save` on individual records inside a `for` loop when bulk creation is possible.
- Implementing batch processing without considering memory limits (if batch sizes are extremely large, consider using `CreateInBatches(slice, batchSize)`).

## Done Criteria

- Loop-based queries are replaced with pre-fetching into maps.
- Loop-based insertions are replaced with slice construction and a single `db.Create(&slice)` call.
- The system handles multiple payloads efficiently without excessive database queries.
