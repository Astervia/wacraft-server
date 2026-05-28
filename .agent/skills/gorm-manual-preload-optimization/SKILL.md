# Skill: GORM Manual Preload Optimization

Use this skill to optimize "get-or-save" database operations by manually populating relationship fields of a created entity using available in-memory data, eliminating redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion.

## Purpose

To prevent unnecessary `SELECT` queries triggered by `.Preload()` when inserting a new database record whose related entities are already loaded in memory (e.g., from prior validation or fetching steps within the same transaction or service layer).

## Core Problem

When creating a new entity that has relations (e.g., a `Webhook` belonging to a `Workspace`), GORM's `Create` method persists the foreign key, but it does not automatically fetch or populate the full related struct unless explicitly instructed.
Often, developers will insert the record and immediately follow it with a `.Preload("Workspace").First(...)` to return a fully hydrated model.
If the related entity (the `Workspace`) was already fetched earlier in the same function (e.g., to verify it exists or check permissions), executing `.Preload` performs a completely redundant `SELECT` query against the database.

## Preferred Pattern

Instead of re-querying the database, manually assign the in-memory struct to the relationship pointer field on the newly created entity.

## Workflow

1. Identify areas where an entity is created and immediately followed by a fetch with `.Preload()`.
2. Check if the related entity (being preloaded) is already available in memory within the current scope (e.g., passed as an argument, or fetched just prior to the creation).
3. If it is available, remove the redundant `.Preload()` query.
4. Manually assign the in-memory related entity to the corresponding relationship field (pointer) on the created entity.

## Example

**Before (Inefficient):**
```go
// We already have `workspace` in memory
workspace := getWorkspaceByID(workspaceID)

newWebhook := &Webhook{
    WorkspaceID: workspace.ID,
    Url:         url,
}
db.Create(newWebhook)

// REDUNDANT QUERY!
var hydratedWebhook Webhook
db.Preload("Workspace").First(&hydratedWebhook, newWebhook.ID)
return &hydratedWebhook
```

**After (Optimized):**
```go
// We already have `workspace` in memory
workspace := getWorkspaceByID(workspaceID)

newWebhook := &Webhook{
    WorkspaceID: workspace.ID,
    Url:         url,
}
db.Create(newWebhook)

// Manually populate the relationship
newWebhook.Workspace = workspace

// No extra query needed!
return newWebhook
```

## Anti-Patterns

- Executing `.Preload()` immediately after `.Create()` when the preloaded data was already fetched seconds earlier.
- Passing around IDs only when the full struct is needed and available, forcing downstream functions to re-fetch.

## Done Criteria

- Redundant `.Preload()` queries following insertions are removed.
- Relationship fields are manually populated using existing in-memory data.
- The returned entity structure remains identical (fully populated), but with fewer database operations.
