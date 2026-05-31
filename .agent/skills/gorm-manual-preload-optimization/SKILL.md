# Skill: GORM Manual Preload Optimization

Use this skill to optimize "get-or-create" or insertion operations in GORM when the relationship data is already available in memory, avoiding redundant `.Preload()` queries to the database.

## Purpose

To eliminate N+1 query issues or redundant database round-trips caused by calling GORM's `.Preload()` immediately following a record insertion. When creating a record, if you already have the related entity in memory, you can manually assign its pointer to the relationship field on the created entity instead of asking GORM to query the database again.

## Core Problem

A common pattern is:
1. Create a record that belongs to another entity (e.g., creating a `Webhook` that belongs to a `Workspace`).
2. The function returning the created entity requires the related entity to be preloaded (e.g., returning a `Webhook` with the `Workspace` populated).
3. Developers might run `db.Create(&webhook)` and then `db.Preload("Workspace").First(&webhook, webhook.ID)`.
This results in an extra database query to fetch a `Workspace` that was likely already available (e.g., from the authentication context or a previous validation step).

## Preferred Pattern

Manually populate the relationship fields (pointer assignments) of the created entity using the available in-memory data.

## Workflow

1. Identify the relationship to be populated.
2. Determine if the related entity is already available in memory (e.g., passed as an argument, loaded from context, or fetched during validation).
3. Perform the database insert (`db.Create(...)`).
4. Assign the related entity's pointer to the relationship field of the newly created entity.

```go
// Anti-pattern
db.Create(&newEntity)
db.Preload("RelatedEntity").First(&newEntity, newEntity.ID)
return newEntity

// Preferred Pattern
db.Create(&newEntity)
newEntity.RelatedEntity = &existingRelatedEntity // Manually populate
return newEntity
```

## Anti-Patterns

- Relying on implicit auto-loading via `.Preload()` immediately after insertion when the data is already held in memory.
- Performing redundant fetches that increase database round-trips and latency.

## Done Criteria

- The returned entity has its relationship fields populated correctly.
- No redundant database queries are executed to fetch data that was already available.
