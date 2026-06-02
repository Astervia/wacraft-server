# Skill: GORM Manual Preload Optimization

Use this skill when performing "get-or-save" (or "create") operations where you have the related entity data already available in memory, but GORM might otherwise trigger a redundant `.Preload()` query.

## Purpose

Optimize "get-or-save" database operations by manually populating the relationship fields (pointer assignments) of the created entity using available in-memory data. This eliminates redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion.

## Core Problem

When creating or saving a record that has associations (e.g., `BelongsTo`), developers often chain `.Preload()` onto the fetch or save operation to ensure the returned entity is fully populated for the caller.
If the related entity was just fetched or is already known in memory (for instance, to validate its existence before creating the child record), letting GORM fetch it again via `.Preload()` results in an unnecessary N+1 query pattern immediately after an `INSERT`.

## Preferred Pattern

Instead of relying on GORM to query the database for the related entity after creation, manually assign the in-memory object to the relationship pointer on the newly created model.

- Perform the necessary checks/fetches for the parent/related entity.
- Create the new child record in the database.
- Immediately assign the already-fetched parent entity to the pointer field on the child record.
- Return the child record without executing an additional `.Preload()`.

## Workflow

1. Identify operations where a new record is created and its relationship (e.g., `Workspace`, `User`) is returned.
2. Check if the parent entity is already loaded in memory (e.g., fetched earlier in the same service method or passed in).
3. Insert the new child record.
4. Manually assign the in-memory parent object to the corresponding pointer field on the child record (e.g., `child.Workspace = &parentWorkspace`).
5. Ensure that this manual assignment accurately reflects the relationship defined in the struct.

## Anti-Patterns

- Executing a `.Preload("RelatedEntity")` query immediately after an `INSERT` when the `RelatedEntity` data is already available in the local scope.
- Returning an incomplete entity (missing the relationship pointer) to a caller that expects it to be populated.

## Done Criteria

- Redundant `.Preload()` queries are eliminated after record insertion.
- The returned entity correctly includes its related data via manual pointer assignment.
- Existing tests pass, and the performance (fewer DB round-trips) is verifiably improved.
