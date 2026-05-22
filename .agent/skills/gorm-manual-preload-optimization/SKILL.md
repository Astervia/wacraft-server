# Skill: GORM Manual Preload Optimization

Use this skill when implementing "get-or-create" or "upsert" patterns in GORM where you need to return an entity with its relationships fully populated, and you already have the related data in memory.

## Purpose

Eliminate redundant database queries (N+1 issues and unnecessary `.Preload()` calls) by manually populating the relationship fields of a newly created or retrieved entity using the data already available in the current execution context.

## Core Problem

When an entity is created using GORM (`db.Create(&entity)`), GORM does not automatically fetch related entities unless explicitly told to do so, even if the foreign keys are set. Often, developers immediately follow a `.Create()` with a `.Preload("Relationship").First(&entity)` to fetch the newly created entity along with its relationships. If the related entity was just used to set the foreign key (e.g., `entity.WorkspaceID = workspace.ID`), executing an additional database query to fetch `Workspace` is a wasteful round-trip.

## Preferred Pattern

Manually assign the related entity pointer to the created entity's relationship field using the in-memory data, rather than issuing a new database query to preload it.

## Workflow

1. Identify the "get-or-create" or insertion operation.
2. Determine if the caller requires the returned entity to have its relationships populated.
3. Check if the related entity data is already available in the current context (e.g., passed as an argument, or fetched earlier in the handler/service).
4. After successfully creating or saving the main entity, manually assign the related data to the struct's relationship pointer field.
5. Return the fully populated entity without executing a subsequent `.Preload()` query.

## Example

Instead of:
```go
// Anti-pattern
db.Create(&member)
db.Preload("Workspace").Preload("User").First(&member, member.ID)
return &member, nil
```

Do this:
```go
// Preferred
db.Create(&member)
member.Workspace = workspace // In-memory assignment
member.User = user           // In-memory assignment
return &member, nil
```

## Inspect First

- `src/<domain>/service/` or `src/<domain>/repository/` for functions that create and return entities.
- Handlers in `src/<domain>/handler/` to see what relationships the response serializers expect.

## Anti-Patterns

- Executing `.Preload()` queries immediately after an `INSERT` when the preloaded data was already queried earlier in the request lifecycle.
- Leaving relationship pointers `nil` on returned entities when the caller expects them to be populated (which might cause panics or empty JSON responses downstream).

## Done Criteria

- Redundant `.Preload()` database queries after creation are removed.
- The returned entity correctly contains the relationship data via manual pointer assignment.
- Downstream handlers and serializers still receive the fully populated entity they expect.
