# GORM Manual Preload Optimization

When optimizing "get-or-save" database operations in GORM, avoid making redundant database round-trips for newly created records that immediately require preloaded relationship fields.

Instead of performing a `db.Create(&entity)` followed by another `db.Preload("Relationship").First(&entity, entity.ID)`, manually populate the relationship fields on the entity using available in-memory data (e.g., the parent models or relationships you already fetched to validate the creation).

## Example:

### Unoptimized Pattern (Anti-pattern)
```go
// Create the new entity
if err := db.Create(&childEntity).Error; err != nil {
    return err
}

// Redundant database call to preload the parent
if err := db.Preload("Parent").First(&childEntity, childEntity.ID).Error; err != nil {
    return err
}
```

### Optimized Pattern
```go
// You likely already have the parent entity in memory from previous validation
// ParentEntity is available.

// Create the new entity
if err := db.Create(&childEntity).Error; err != nil {
    return err
}

// Manually assign the pointer to the relationship field
childEntity.Parent = &parentEntity

// Return the childEntity which now acts as if it was preloaded
```

By manually assigning relationship pointers, we can completely eliminate an unnecessary `SELECT` query, optimizing performance and reducing database load.
