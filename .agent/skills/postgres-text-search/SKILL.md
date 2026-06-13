# Skill: Postgres Text Search

Use this skill when implementing text-based search functionality across the codebase, particularly when constructing queries using GORM that need to be robust, case-insensitive, and accent-insensitive.

## Purpose

Standardize the approach to text searching across the repository to ensure consistent behavior, optimize index usage, and handle nulls safely.

## Core Problem

Simple `LIKE` or `ILIKE` queries in Postgres are sensitive to accents (e.g., searching for "cafe" will not match "café"). Additionally, searching against nullable columns can lead to unexpected results or index misses if not handled correctly. The project relies heavily on the `pg_trgm` extension and custom `immutable_unaccent` functions for performant text searching.

## Preferred Pattern

Always use the standard Postgres search pattern: `immutable_unaccent(COALESCE(column::text, '')) ILIKE immutable_unaccent(?)`.

## Workflow

1. Identify the column(s) that need to be searchable via text input.
2. When building the GORM query (e.g., in `Where()` clauses), use the standard expression pattern:
   ```go
   expr := "immutable_unaccent(COALESCE(column_name::text, ''))"
   db.Where(expr + " ILIKE immutable_unaccent(?)", "%"+searchTerm+"%")
   ```
3. Ensure that the search term provided to the `?` parameter includes any necessary wildcard characters (e.g., `%`).
4. If searching across multiple columns, combine them using `OR`:
   ```go
   expr1 := "immutable_unaccent(COALESCE(col1::text, ''))"
   expr2 := "immutable_unaccent(COALESCE(col2::text, ''))"
   db.Where(fmt.Sprintf("%s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?)", expr1, expr2), term, term)
   ```

## Inspect First

- `src/<domain>/service/like.go` files (e.g., `src/message/service/like.go`, `src/campaign/service/like.go`) to see existing implementations of this pattern.
- Database migrations defining `immutable_unaccent` (e.g., `src/database/migrations/20250825004905_create_immutable_unaccent.go`).
- Database migrations defining GIN indexes using `gin_trgm_ops` for the target columns.

## Anti-Patterns

- Using basic `ILIKE` without `immutable_unaccent` on either the column or the search parameter.
- Failing to handle NULL values with `COALESCE`, which can bypass GIN indexes.
- Not casting the column to `::text` inside the `COALESCE` function when necessary (e.g., for JSONB text extraction).
- Concatenating search terms directly into the SQL string instead of using parameterized queries (`?`).

## Done Criteria

- The search functionality uses `immutable_unaccent(COALESCE(column::text, '')) ILIKE immutable_unaccent(?)`.
- Null values are handled safely via `COALESCE`.
- The search correctly ignores accents and case differences.
- GORM queries are parameterized to prevent SQL injection.
