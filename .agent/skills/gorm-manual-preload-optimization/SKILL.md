# Skill: GORM Manual Preload Optimization

Use this skill to optimize 'get-or-save' database operations by manually populating the relationship fields (pointer assignments) of the created entity using available in-memory data, thereby eliminating redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion.

## Purpose

To reduce unnecessary database queries (N+1 or immediate re-fetching) when a new entity is inserted and its relationships are already known in memory. This is particularly relevant after an insertion where the newly inserted record needs to be returned with its relational data populated.

## Core Problem

When an entity is created (e.g., using `gorm.DB.Create`), GORM sets the primary key on the created struct but does not automatically populate related structs (e.g., `belongs_to` relationships) unless instructed. A common anti-pattern is to immediately issue another database query using `.Preload("Relationship")` to fetch the complete entity, despite the related entity data often being already available in the context of the creation handler/service.

## Preferred Pattern

Instead of querying the database to preload relationships on a newly created entity, manually assign the in-memory relationship objects to the entity's struct pointers.

1. Ensure the related entity data is available in memory.
2. Insert the new entity using GORM.
3. Manually assign the related entity pointer to the newly inserted entity struct.

## Workflow

1. Identify areas where an entity is created and immediately followed by a query with `.Preload()` to fetch relationships for the created entity.
2. Check if the related entity data is already present in memory (e.g., passed as an argument to the function, fetched prior to creation).
3. If the data is available, remove the redundant `.Preload()` query.
4. Manually assign the related in-memory object to the appropriate pointer field on the created entity's struct.

## Anti-Patterns

- Executing a `.Preload()` immediately after `db.Create()` when the related entity is already available in the current scope.
- Unnecessary database round-trips for data that the application already possesses.

## Done Criteria

- The redundant `.Preload()` query has been removed.
- The newly created entity's relationship fields are manually populated using in-memory data.
- The application logic works identically as before but with reduced database query overhead.
