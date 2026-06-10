# Skill: GORM Manual Preload Optimization

Use this skill when implementing a "get-or-save" (or "create if not exists") pattern in GORM, to avoid unnecessary database round-trips caused by subsequent `.Preload()` calls immediately following a record insertion.

## Purpose

To eliminate redundant database queries by manually populating the relationship fields (pointer assignments) of a newly created entity using the in-memory data that is already available.

## Core Problem

A common pattern is to find a record or create it if it doesn't exist, and then return that record with its associations loaded.
Often, developers use `.Preload("Association")` on the read query, and if the record is not found, they create it and then issue *another* read query with `.Preload("Association")` to fetch the complete object.
However, when you are creating the record, you often already have the associated data (or at least its primary key/foreign key) in memory. Re-querying the database just to satisfy the GORM structure is an N+1 performance issue.

## Preferred Pattern

When inserting a new record that requires an association to be present in the returned model:
1. Attempt to find the record (with `.Preload()`).
2. If it does not exist, create the record.
3. Instead of re-querying, manually assign the associated struct (or a struct containing the ID) to the relation field of the newly created object.

## Workflow

1. Identify "get-or-save" or "find-or-create" database operations.
2. Check if the newly created entity requires preloaded associations in the return value.
3. If it does, and you have the related entity (or its primary details) available in the scope, assign it directly to the newly created entity's relationship field.
4. Return the constructed entity without issuing an additional SELECT query.

## Example

**Anti-pattern (Redundant Read):**
```go
func GetOrSaveUser(db *gorm.DB, roleID uint) (*User, error) {
    var user User
    // Try to find
    err := db.Preload("Role").Where("role_id = ?", roleID).First(&user).Error
    if err == nil {
        return &user, nil
    }

    // Create
    user = User{RoleID: roleID}
    if err := db.Create(&user).Error; err != nil {
        return nil, err
    }

    // Redundant read just to load Role
    db.Preload("Role").First(&user, user.ID)
    return &user, nil
}
```

**Optimized Pattern (Manual Preload):**
```go
func GetOrSaveUser(db *gorm.DB, role *Role) (*User, error) {
    var user User
    // Try to find
    err := db.Preload("Role").Where("role_id = ?", role.ID).First(&user).Error
    if err == nil {
        return &user, nil
    }

    // Create
    user = User{RoleID: role.ID}
    if err := db.Create(&user).Error; err != nil {
        return nil, err
    }

    // Manual population! No extra query.
    user.Role = role

    return &user, nil
}
```

## Inspect First

- Identify if the handler or service calls the DB multiple times for the same logical entity operation.
- Look at `repository` layer "find-or-create" functions.

## Done Criteria

- The returned entity has its relationship fields populated.
- No extra `SELECT` queries are executed to load data that was already known during `INSERT`.
