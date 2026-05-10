# Skill: Postgres Search Pattern (immutable_unaccent)

Use this skill when implementing case-insensitive, accent-insensitive text searches across the codebase using Postgres and GORM.

## Purpose

Standardize robust text searching that handles both varying cases and accented characters without requiring complex pre-processing or risking full table scans when indexes are present.

## Core Problem

Basic `ILIKE` searches in Postgres are case-insensitive but not accent-insensitive. Users searching for "cafe" might not find "café", and vice versa. Relying on simple `ILIKE` leads to poor search experiences for internationalized or user-generated content.

## Preferred Pattern

The repository utilizes a custom Postgres function `immutable_unaccent(text)` (created via database migrations) paired with `COALESCE` and explicit casting.

The standard pattern for building search expressions is:
`immutable_unaccent(COALESCE(column_name::text, '')) ILIKE immutable_unaccent(?)`

## Workflow

1.  Identify the column(s) to be searched.
2.  Construct the search expression, ensuring both the database column and the user-provided search term are wrapped in `immutable_unaccent`.
3.  Handle potential `NULL` values by wrapping the column reference in `COALESCE(..., '')`.
4.  Explicitly cast the column to `text` if necessary (e.g., if it's a JSONB field or a different type).
5.  Use `fmt.Sprintf` to build the expression dynamically if the column name is variable (but ensure the column name itself is validated/safe, refer to the `safe-dynamic-sql-columns` skill).

## Code Example

```go
// Example for a single column
expr := fmt.Sprintf("immutable_unaccent(COALESCE(%s::text, ''))", safeColumnName)
query = query.Where(expr+" ILIKE immutable_unaccent(?)", "%"+searchTerm+"%")

// Example for multiple columns (e.g., searching across name and email)
const emailExpr = `immutable_unaccent(COALESCE("Contact".email, ''))`
const nameExpr = `immutable_unaccent(COALESCE("Contact".name, ''))`

query = query.Where(
    fmt.Sprintf("%s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?)", emailExpr, nameExpr),
    "%"+searchTerm+"%", "%"+searchTerm+"%",
)
```

## Inspect First

-   `src/message/service/like.go` or `src/campaign/service/like.go` to see how this pattern is implemented in practice.
-   `src/database/migrations/` (specifically files referencing `immutable_unaccent`) to understand how the underlying database function and indexes (like `gin_trgm_ops`) are created.

## Anti-Patterns

-   Using bare `ILIKE` without `immutable_unaccent` when accent insensitivity is required.
-   Applying `immutable_unaccent` to the database column but forgetting to apply it to the user's search parameter `?`. Both sides must match for indexes to be utilized effectively.
-   Failing to handle `NULL` values with `COALESCE`, which can lead to unexpected search results or missed records.

## Done Criteria

-   The search query uses `immutable_unaccent` on both the column and the search term.
-   The column reference handles `NULL` values via `COALESCE`.
-   The pattern aligns with existing indexed expressions in the database migrations.
