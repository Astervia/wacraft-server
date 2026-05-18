# Skill: GORM Manual Preload Optimization

Use this skill when implementing "get-or-save" or insertion-heavy database operations that return nested entities.

## Purpose

To eliminate redundant database round-trips caused by relying on GORM `.Preload()` queries immediately after creating a new record, when the related data is already available in memory.

## Default Assumption

When inserting a new record that belongs to other entities (e.g., creating a user that belongs to a workspace), the application often already has the parent entities or their details in memory. Using GORM `.Preload()` to fetch relationships immediately after an insert introduces unnecessary `SELECT` queries (N+1-like behavior on creation).

## Optimization Pattern

Instead of performing a `.Preload()` fetch after `db.Create()`, manually populate the relationship pointer fields of the created entity using the available in-memory data.

### Example Scenario

**Suboptimal Pattern:**
```go
// 1. Create record
if err := db.Create(&newRecord).Error; err != nil {
    return nil, err
}

// 2. Redundant fetch just to load the relation
if err := db.Preload("Workspace").First(&newRecord, newRecord.ID).Error; err != nil {
    return nil, err
}
return &newRecord, nil
```

**Optimized Pattern:**
```go
// 1. Create record
if err := db.Create(&newRecord).Error; err != nil {
    return nil, err
}

// 2. Manually populate the relationship field using existing in-memory data
newRecord.Workspace = existingWorkspaceReference // Assign pointer to in-memory data

return &newRecord, nil
```

## Workflow

1. Identify database insertion points (`db.Create`, `db.Save`) where the returned entity requires populated relationships.
2. Check if the parent/related entities are already available in the current function scope or context.
3. Replace post-creation `.Preload()` calls with direct pointer assignments to the entity's relationship fields.
4. Ensure the assigned structs match the expected GORM entity definitions.

## Done Criteria

- Post-creation `SELECT` queries for relationships are removed.
- The returned entity has its relationship fields populated correctly.
- Code readability is maintained by adding a brief comment explaining the manual population.
