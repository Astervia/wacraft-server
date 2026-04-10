# Skill: GORM External Entity Wrapper

Use this skill when fetching external or shared domain models via GORM where you need to eagerly load (`.Preload()`) related local entities, but the external model lacks the necessary GORM relationship tags.

## Purpose

Enable clean eager loading (`.Preload()`) of related entities when the base model is imported from an external or shared module (like `github.com/Astervia/wacraft-core`) and does not define GORM relationship tags (e.g., `foreignKey`, `references`).

## Core Problem

GORM's `.Preload()` functionality requires explicit relationship tags (`gorm:"foreignKey:...,references:..."`) on the struct fields to know how to join tables.
When depending on shared or core libraries, those domain models often (and correctly) omit persistence-layer details like GORM tags.
Attempting to `.Preload()` onto an entity without these tags results in GORM failing to execute the join or load the related data.

## Preferred Pattern

Create a localized wrapper struct within the specific feature's persistence or repository layer.

- Embed the external entity in the wrapper struct.
- Define the relationship field directly on the wrapper struct.
- Add the necessary `gorm` tags (e.g., `foreignKey`, `references`) to the newly added field.
- Ensure the wrapper specifies the underlying table name of the external entity to prevent GORM from inferring a table name based on the wrapper struct's name.

## Workflow

1. Identify the external entity and the local relationship that needs to be eager-loaded.
2. In the repository or persistence layer, define a local wrapper struct that embeds the external entity.
3. Add the related entity field to the wrapper struct and annotate it with the proper `gorm` relationship tags.
4. Implement the `TableName()` method on the wrapper struct to return the correct table name corresponding to the external entity.
5. Use this wrapper struct in your GORM queries to fetch the data with `.Preload()`.
6. After fetching, map the results back to the expected domain model or DTO as necessary before returning from the persistence layer.

## Inspect First

- The external model definition (e.g., in `github.com/Astervia/wacraft-core`).
- The related local model definition.
- Existing repository patterns in `src/<domain>/service/` or `src/<domain>/repository/`.

## Anti-Patterns

- Modifying the shared/external module to include GORM tags (leaks persistence details into core models).
- Manually executing multiple queries (N+1 queries) instead of using `.Preload()` when a join is more efficient.
- Failing to implement `TableName()` on the wrapper, causing GORM to query a non-existent table (e.g., querying `external_entity_wrappers` instead of `external_entities`).

## Done Criteria

- The local wrapper successfully enables `.Preload()`.
- The external module remains unaware of GORM-specific tags.
- The `TableName()` method correctly routes queries to the underlying table.
- The repository returns cleanly mapped domain entities to the service layer.
