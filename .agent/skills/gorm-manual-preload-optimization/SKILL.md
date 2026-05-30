# Skill: GORM Manual Preload Optimization

Use this skill to optimize "get-or-save" database operations and eliminate redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion.

## Purpose

When creating a new database record that has relationships, it is often necessary to return the fully hydrated entity (including its relationships) to the caller. Standard GORM behavior requires saving the base entity and then performing a separate `.Preload()` query to fetch the related entities.

This skill outlines how to manually populate the relationship fields (pointer assignments) of the created entity using available in-memory data, thereby avoiding the extra database query.

## Core Problem

A common pattern is:
1. Try to find a record (with `.Preload()`).
2. If it doesn't exist, create it.
3. Immediately query the database again to load the relationships of the newly created record.

Step 3 is inefficient because the related data is often already available in memory (e.g., from the parent entity or the request payload).

## Preferred Pattern

Manually assign the related in-memory entities to the relationship fields of the newly created base entity.

1. Ensure you have the related entity (or entities) in memory.
2. Create the base entity using GORM.
3. Manually assign a pointer of the related entity to the corresponding relationship field on the created entity.

## Workflow

1. Identify the "get-or-save" or "create-and-return" operation causing redundant queries.
2. Check if the required relationship data is already available in the current scope (e.g., passed as an argument, loaded in a previous step).
3. Perform the GORM insertion for the base entity.
4. Instead of calling `db.Preload(...).First(...)` on the new entity, assign the available related entity pointer directly to the relationship field of the created entity.
5. Return the manually hydrated entity.

## Example

```go
// Inefficient (Causes extra query)
func GetOrSaveChild(db *gorm.DB, parentID uint, childName string) (*Child, error) {
    var child Child
    // ... check if exists ...

    // Create
    newChild := &Child{ParentID: parentID, Name: childName}
    db.Create(newChild)

    // Extra query!
    db.Preload("Parent").First(&child, newChild.ID)
    return &child, nil
}

// Optimized
func GetOrSaveChildOptimized(db *gorm.DB, parent *Parent, childName string) (*Child, error) {
    var child Child
    // ... check if exists ...

    // Create
    newChild := &Child{ParentID: parent.ID, Name: childName}
    db.Create(newChild)

    // Manually assign the parent pointer
    newChild.Parent = parent

    return newChild, nil
}
```

## Anti-Patterns

- Executing `.Preload()` on a newly created record when the related data is already present in memory.
- Creating complex helper functions for simple pointer assignments.
- Modifying the GORM model definitions to bypass standard relationship handling.

## Done Criteria

- The redundant `.Preload()` query after creation is removed.
- The returned entity is fully hydrated with the correct relationship data.
- The optimization is documented or clear from the context.
