# Skill: GORM Manual Preload Optimization

Use this skill when returning a newly created entity from the persistence layer that requires related entities to be included in the response.

## Purpose

Optimize "get-or-save" and basic creation database operations by manually populating relationship fields (pointer assignments) of the created entity using already available in-memory data.

## Core Problem

A common pattern after inserting a record is to immediately query the database again using GORM's `.Preload()` to fetch related entities that were just associated via foreign keys. This causes a redundant database round-trip (N+1 query issue for single records), which degrades performance, especially in high-throughput handlers.

## Preferred Pattern

Instead of re-querying the database, manually assign the in-memory related struct to the newly created entity's relationship field before returning it.

## Workflow

1. Identify areas where an entity is created (e.g., `db.Create(&entity)`).
2. Check if a subsequent `.Preload()` or separate query is executed solely to populate relationship fields on that newly created entity.
3. Determine if the related data is already available in memory (e.g., passed into the function, fetched earlier in the transaction, or provided in the request context).
4. If the data is available, remove the redundant `.Preload()` or secondary query.
5. Manually assign a pointer to the in-memory related entity to the corresponding relationship field on the created entity.

## Inspect First

- Repository creation methods (e.g., `Create`, `GetOrCreate`, `Save`) in `src/<domain>/repository/` or `src/<domain>/service/`.
- Handlers that return nested JSON responses immediately after creating a resource.

## Anti-Patterns

- Executing `db.Preload("RelatedEntity").First(&entity, entity.ID)` immediately after `db.Create(&entity)`.
- Re-fetching the entire parent entity when only a child record was added.

## Done Criteria

- The redundant `.Preload()` or secondary query is removed.
- The relationship field is manually populated via pointer assignment.
- The service or handler layer receives the fully populated entity without noticing the change.
- Tests verify that the returned entity includes the expected related data.
