# Skill: GORM Manual Preload Optimization

Use this skill when implementing "get-or-save" (or "create-if-not-exists") database operations where related entities are already available in-memory, but the current implementation redundantly queries the database using GORM's `.Preload()` after record insertion.

## Purpose

Optimize database interactions by eliminating redundant round-trips for fetching related entities immediately after creating a new record.

## Core Problem

A common pattern when saving new records with relationships is to:
1. Create the base record via `db.Create()`.
2. Immediately execute another query to fetch the newly created record along with its relationships using `db.Preload("Relationship").First()`.

If the related entity's data was already queried or is available in memory (e.g., from a prior lookup required to validate the insertion), this second query is inefficient and unnecessary.

## Preferred Pattern

Manually populate the relationship fields on the newly created entity struct via pointer assignments using the already available in-memory data.

## Workflow

1. Identify areas using `.Preload()` immediately following a `db.Create()` or `db.Save()` call.
2. Check if the data being preloaded was already queried earlier in the function or request scope.
3. If the data is available in memory, remove the redundant `.Preload()` query.
4. Manually assign the in-memory related entity (or its relevant fields) to the relationship pointer/struct on the newly created record.
5. Return the manually populated record.

## Inspect First

- `src/<domain>/service/` or `src/<domain>/repository/` to find "get-or-create" logic.
- Look for `db.Create(&record)` followed shortly by `db.Preload(...).First(&record, record.ID)`.

## Anti-Patterns

- Firing a `.Preload()` query for related data that was just inserted or fetched moments before in the same transaction or request scope.
- Overusing GORM's automatic `.Preload()` without considering the current in-memory state.

## Done Criteria

- The redundant `.Preload()` query is removed.
- The newly created entity's relationship fields are properly populated via manual in-memory assignment.
- The repository returns the fully populated entity, matching the expected contract of the service layer.
- Tests confirm that the returned entity contains the expected related data and that database query counts are reduced.
