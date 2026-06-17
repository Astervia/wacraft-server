# GORM Manual Preload Optimization

## Context
When creating new records using GORM that have relationships (e.g., `BelongsTo` or `HasOne`), the typical pattern is to create the record and then immediately issue a new query with `.Preload()` to fetch the related entities. This results in unnecessary database round-trips when the related data is already available in memory prior to the insertion.

## Optimization Pattern
To optimize these "get-or-save" or insertion operations, avoid the redundant `.Preload()` query. Instead, manually populate the relationship fields (pointer assignments) of the created entity using the in-memory data that is already available.

### Before (Anti-pattern)
```go
// 1. Fetch related data
workspace := &entity.Workspace{}
db.First(workspace, req.WorkspaceID)

// 2. Create the new record
newMember := &entity.WorkspaceMember{
    WorkspaceID: workspace.ID,
    UserID:      req.UserID,
}
db.Create(newMember)

// 3. Redundant database query to preload the relationship
db.Preload("Workspace").First(newMember, newMember.ID)
```

### After (Optimized)
```go
// 1. Fetch related data
workspace := &entity.Workspace{}
db.First(workspace, req.WorkspaceID)

// 2. Create the new record
newMember := &entity.WorkspaceMember{
    WorkspaceID: workspace.ID,
    UserID:      req.UserID,
}
db.Create(newMember)

// 3. Manually attach the in-memory relationship
newMember.Workspace = workspace
```

## Benefits
* Eliminates an `N+1` style redundant database query.
* Reduces database load and latency, especially during bulk operations or frequent insertions.
* Maintains identical response structures for API endpoints without needing to refactor the external contract.
