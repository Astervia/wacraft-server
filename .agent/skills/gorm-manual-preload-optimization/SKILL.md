# GORM Manual Preload Optimization

## Description

Optimizes "get-or-save" or insertion-heavy database operations by manually populating relationship fields (pointer assignments) of a newly created entity using available in-memory data. This eliminates redundant database round-trips caused by relying on GORM's `.Preload()` immediately after inserting a record.

## Motivation

When creating a new database record and subsequently needing to return it with its associated relations fully loaded, a common anti-pattern is to insert the record and then perform a subsequent query with `.Preload("Relation")`. This causes an unnecessary extra query (N+1-like behavior on writes) when the related entity data is often already known or available in memory (e.g., from the incoming request or previous validation queries).

By manually assigning the pointer to the relation after `db.Create()`, we avoid this extra query while still returning a fully hydrated struct to the caller.

## Workflow

1.  **Identify the insertion:** Locate the GORM `db.Create(&entity)` operation.
2.  **Identify the redundant preload:** Look for a subsequent `db.Preload("Relation").First(&entity, entity.ID)` or similar pattern.
3.  **Ensure relation data is available:** Verify that the full struct of the related entity is available in memory at this point in the handler or service.
4.  **Replace preload with pointer assignment:** Remove the redundant `Preload()` query. Instead, assign the memory pointer of the related entity directly to the newly created entity's relationship field.

## Example

### Before (Anti-pattern)

```go
func CreateWorkspace(db *gorm.DB, reqData CreateRequest, currentUser *User) (*Workspace, error) {
    workspace := &Workspace{
        Name: reqData.Name,
        OwnerID: currentUser.ID,
    }

    // 1. Insert query
    if err := db.Create(workspace).Error; err != nil {
        return nil, err
    }

    // 2. Redundant query to load the Owner
    if err := db.Preload("Owner").First(workspace, workspace.ID).Error; err != nil {
         return nil, err
    }

    return workspace, nil
}
```

### After (Optimized)

```go
func CreateWorkspace(db *gorm.DB, reqData CreateRequest, currentUser *User) (*Workspace, error) {
    workspace := &Workspace{
        Name: reqData.Name,
        OwnerID: currentUser.ID,
    }

    // 1. Insert query
    if err := db.Create(workspace).Error; err != nil {
        return nil, err
    }

    // 2. Manual Preload Optimization: Eliminate extra query by assigning pointer
    workspace.Owner = currentUser

    return workspace, nil
}
```
