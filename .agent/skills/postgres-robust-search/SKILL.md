# Skill: Postgres Robust Search

Use this skill when implementing search or filter endpoints that query text columns in PostgreSQL, especially via GORM.

## Purpose

Ensure text searches are robust, case-insensitive, accent-insensitive, and immune to NULL pointer issues when filtering database records.

## Core Problem

Directly using `ILIKE ?` on text columns can lead to:
- Accent sensitivity (e.g., searching for "cafe" won't match "café").
- Null constraint issues if the column value is NULL, which might evaluate in unexpected ways compared to an empty string.

## Preferred Pattern

Always use the established pattern: `immutable_unaccent(COALESCE(column::text, '')) ILIKE immutable_unaccent(?)`

## Workflow

1. Identify text columns that need to be searchable or filterable.
2. In your GORM queries (`.Where(...)`), instead of `column ILIKE ?`, use the pattern:
   `immutable_unaccent(COALESCE(column_name::text, '')) ILIKE immutable_unaccent(?)`
3. Ensure you format the `?` value appropriately for ILIKE matches (e.g., `"%"+searchTerm+"%"`).

## Inspect First

- Search queries in `src/<domain>/service/like.go` or `src/<domain>/service/count.go` (e.g., `message/service/like.go`, `campaign/service/like.go`).
- Notice how `fmt.Sprintf` is used carefully (only for mapping valid, normalized keys, never user input directly) to construct the query string safely.

## Anti-Patterns

- Using basic `ILIKE ?` without handling accents.
- Failing to use `COALESCE`, potentially missing rows where the column is NULL.
- Applying `immutable_unaccent` to one side of the query but not the other.
- Concatenating user input directly into the query string instead of using parameterized queries `?` for the search term.

## Done Criteria

- Searches handle case insensitivity.
- Searches handle accent insensitivity.
- Null values do not break the query or exclude results incorrectly.
- The `immutable_unaccent` function is applied consistently.
