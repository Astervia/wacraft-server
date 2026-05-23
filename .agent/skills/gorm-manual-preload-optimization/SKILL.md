# Skill: GORM Manual Preload Optimization

Use this skill when performing "get-or-save" database operations where you create a new record and immediately need it fully populated with its related entities to avoid redundant database round-trips.

## Purpose

To eliminate unnecessary N+1 query patterns caused by GORM `.Preload()` calls immediately following a record insertion, by manually populating relationship fields (pointer assignments) using available in-memory data.

## Core Problem

When an entity is created in the database using GORM, any related entities (e.g., a `User` or `Workspace` that the new entity belongs to) are not automatically loaded back into the struct. Developers often follow up a `.Create()` with a `.Preload("RelatedEntity").First()` to fetch the complete object. This results in an extra database query simply to retrieve data the application already has in memory.

## Preferred Pattern

Manually assign the in-memory relationship instances to the newly created entity's relationship fields.

- Create the entity in the database (`db.Create(&entity)`).
- Identify the related entities that the application already possesses (e.g., from the incoming request context or previous database lookups).
- Directly assign pointers of those existing in-memory objects to the corresponding relationship fields on the newly created entity.
- Return the fully populated entity without executing a subsequent `.Preload()`.

## Workflow

1. Identify the creation operation (e.g., `db.Create(&record)`).
2. Check if a subsequent database query is being performed solely to populate related fields (e.g., `.Preload("Workspace").First(&record)`).
3. If the related data (e.g., `workspace`) is already available in the current function scope, remove the secondary database query.
4. Directly assign the existing data to the relationship field on the created record (e.g., `record.Workspace = workspace`).
5. Ensure that the returned entity now contains the related data exactly as it would have if `.Preload()` had been used.

## Inspect First

- The function performing the creation and any subsequent lookups.
- The struct definition of the entity being created to identify the correct relationship field names.
- The surrounding function scope to find available in-memory instances of the related data.

## Anti-Patterns

- Executing `.Preload().First()` immediately after `.Create()` to fetch related data that is already known.
- Ignoring the relationship fields and forcing callers to perform their own lookups.

## Done Criteria

- The redundant `.Preload()` query following creation is removed.
- The returned entity has its relationship fields fully populated.
- The relationship fields are assigned using pointers to the existing in-memory data.
- The application performs fewer database queries while maintaining the expected output.