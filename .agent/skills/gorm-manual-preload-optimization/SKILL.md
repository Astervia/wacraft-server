# Skill: GORM Manual Preload Optimization

Use this skill when optimizing "get-or-save" or insertion-followed-by-read database operations that use GORM's `.Preload()`.

## Purpose

Eliminate redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion, by manually populating the relationship fields using already available in-memory data.

## Workflow

1. Identify areas where an entity is created using `db.Create()`.
2. Check if the newly created entity is immediately queried again with `.Preload()` to fetch relationships that were just linked via foreign keys.
3. Instead of re-querying the database, manually assign the in-memory related entity (or its pointer) directly to the created entity's relationship field.
4. Return or use the manually populated entity.

## Example

Instead of:

```go
db.Create(&userRole)
// Redundant query!
db.Preload("Role").First(&userRole, userRole.ID)
```

Do:

```go
db.Create(&userRole)
// Manually populate the relationship
userRole.Role = &role // Where `role` is the in-memory object used to get the role ID
```

## Impact

Reduces O(N) or 1+1 query issues during entity creation flows, significantly lowering database latency under high load.
