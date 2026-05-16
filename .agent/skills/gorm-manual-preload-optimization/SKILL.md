# Skill: GORM Manual Preload Optimization

Use this skill when implementing 'get-or-save' database operations to eliminate redundant GORM `.Preload()` queries by manually populating relationship fields of created entities using available in-memory data.

## Purpose

To prevent N+1 query performance issues immediately following record creation. When a new entity is inserted via GORM, it does not automatically fetch its relationship data even if the foreign keys are set. Performing a `.Preload()` after insertion incurs an unnecessary database round-trip because the related entity data is often already available in memory within the current scope (e.g., from a prior lookup or from the request context).

## Core Problem

A common but inefficient pattern is to:
1. Fetch a related entity (e.g., to verify it exists).
2. Create a new primary entity referencing the related entity's ID.
3. Reload the primary entity from the database using `.Preload()` just to populate the related struct field before returning it in the API response.

This results in a completely redundant `SELECT` query.

## Preferred Pattern

Manually assign the related entity pointer to the newly created struct before returning.

- Look up the related entity (if not already available).
- Create the new primary entity using the foreign key.
- Assign the related entity (or a reference to it) directly to the primary entity's relationship field.
- Return the populated entity without executing a subsequent `.Preload()`.

## Workflow

1. Identify the 'get-or-save' or creation operation.
2. Ensure the related entities are available in memory (often fetched earlier in the handler or service for validation).
3. Insert the new record via `db.Create()`.
4. Directly assign the in-memory related entity to the newly created object's relationship field (e.g., `newEntity.RelatedEntity = &relatedEntity`).
5. Return the result, avoiding any `db.Preload().First()` reloading logic.

## Anti-Patterns

- Executing `db.Preload("Relationship").First(&entity, entity.ID)` immediately after `db.Create(&entity)`.
- Re-querying data that was already validated or fetched higher up in the request context.

## Done Criteria

- The `.Preload()` call following the insert is removed.
- The returned entity correctly includes the nested relationship data.
- The number of SQL queries executed during the creation flow is reduced.
