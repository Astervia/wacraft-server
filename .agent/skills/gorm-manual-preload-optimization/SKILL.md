# Skill: GORM Manual Preload Optimization

Use this skill when implementing a "get-or-save" or "create-and-return" pattern where you insert a new record into the database that has relations (e.g., belongs-to, has-one) to existing in-memory data that you already queried or possess.

## Purpose

To eliminate redundant database round-trips caused by calling `.Preload()` or `.Joins()` immediately after inserting a new record. By manually assigning the in-memory relationship pointers to the newly created entity, you avoid unnecessary queries to fetch data you already have.

## Core Problem

A common anti-pattern in ORM usage is inserting a record, and then immediately querying it back out with `Preload()` just to populate the relationship fields for the returned DTO or domain model.
This causes a minimum of two queries (one INSERT, one SELECT for the main model, and potentially more for the preloaded relationships) when only the INSERT was strictly necessary.

## Preferred Pattern

Manually populate the relationship fields on the entity using the existing in-memory data before returning it from the repository/persistence layer.

## Workflow

1. Identify the "create" or "get-or-save" operation.
2. Confirm that the related data needed by the caller is already available in memory (e.g., fetched earlier in the handler or service, or passed as arguments).
3. Insert the new record using `db.Create(&entity)`.
4. After the successful insertion, manually assign the in-memory relation to the corresponding pointer/field on `entity`.
5. Return the fully populated `entity` without issuing any subsequent `.Preload()` queries.

## Example

### Bad (Anti-Pattern)
```go
// We already know the Workspace, but we query the db again to preload it.
db.Create(&newMember)
var result WorkspaceMember
db.Preload("Workspace").First(&result, newMember.ID)
return &result, nil
```

### Good (Optimized)
```go
// We already have the workspace object passed in from the service layer.
db.Create(&newMember)
// Manually satisfy the preload requirement in memory
newMember.Workspace = &existingWorkspace
return &newMember, nil
```

## Anti-Patterns

- Calling `.Preload()` or `.First()` immediately after `db.Create()` on the same record just to fill in relationship pointers that are already known.
- Passing around partial models and forcing the next layer up to guess if relations are loaded. If the contract implies the relation is loaded, load it manually as shown.

## Done Criteria

- The repository layer returns the entity with all expected relations populated.
- No redundant SELECT queries are issued immediately following an INSERT just to satisfy relation pointers.
- Code comments briefly note the manual relation assignment if it's complex, though straightforward pointer assignment is self-documenting.
