# Skill: Postgres Search Pattern

Use this skill when implementing case-insensitive and accent-insensitive text searches across the codebase using GORM and Postgres.

## Purpose

To ensure robust and consistent text search functionality that correctly handles null values, case differences, and accent marks across various entities.

## Core Problem

Basic `ILIKE` searches in Postgres do not account for accent marks (e.g., searching for "cafe" might not match "café"). Furthermore, searching against columns that might contain `NULL` can lead to unexpected behavior if not properly handled. The project has established a standard way to perform these searches to ensure consistency and correct index usage.

## Preferred Pattern

Use the custom `immutable_unaccent` function in combination with `COALESCE` and `ILIKE`.

The standard pattern is:
`immutable_unaccent(COALESCE(column_name::text, '')) ILIKE immutable_unaccent(?)`

## Workflow

1.  **Identify the target columns:** Determine which column(s) need to be searchable.
2.  **Construct the expression:** Use `fmt.Sprintf` or a constant string to create the expression for the left side of the `ILIKE` operator:
    `expr := fmt.Sprintf("immutable_unaccent(COALESCE(%s::text, ''))", normalizedKey)`
    *Note: Always map dynamic column keys to hardcoded strings before this step to prevent SQL injection (see `safe-dynamic-sql-columns` skill).*
3.  **Apply to GORM query:** Use the constructed expression in a `Where` clause:
    `query.Where(expr+" ILIKE immutable_unaccent(?)", likeText)`
4.  **Index consideration:** Be aware that for this search to be efficient on large datasets, a corresponding trigram GIN index using `immutable_unaccent` and `COALESCE` must exist in the database (typically created via migrations).

## Inspect First

-   `src/message/service/like.go` for complex, multi-column search examples.
-   `src/user/service/like.go` or `src/campaign/service/like.go` for straightforward single-column search examples.
-   `src/database/migrations/` to see how the `immutable_unaccent` function and related indexes are created.

## Anti-Patterns

-   Using simple `ILIKE ?` without `immutable_unaccent`, which will fail to match accented characters.
-   Applying `immutable_unaccent` only to one side of the `ILIKE` operator (it must be on both sides for correct index usage).
-   Failing to use `COALESCE(..., '')` which can cause issues with `NULL` column values.

## Done Criteria

-   Text search implementation uses `immutable_unaccent(COALESCE(column::text, '')) ILIKE immutable_unaccent(?)`.
-   Dynamic column names are safely mapped (avoiding SQL injection).
-   Tests confirm the search correctly matches across case and accent differences.
