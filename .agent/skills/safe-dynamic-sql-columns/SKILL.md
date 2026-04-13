# Skill: Safe Dynamic SQL Columns

Use this skill when dynamically selecting or filtering by column names in SQL queries, particularly when constructing queries using GORM's `.Where()` or `.Select()` with string formatting.

## Purpose

Prevent SQL injection vulnerabilities that arise when user input or untrusted data is used directly to specify database column names in queries.

## Core Problem

While parameterized queries (`?` in GORM) protect against SQL injection in column *values*, they cannot be used for column *names* or table *names*. If a feature allows filtering or sorting by dynamic fields (e.g., via query parameters), and that field name is blindly passed into `fmt.Sprintf("JSON_EXTRACT(%s, '$.key')", field)`, an attacker could inject arbitrary SQL.

## Preferred Pattern

Always validate dynamic column keys against an explicit whitelist or use strongly-typed models (like `SearchableColumn`) before interpolating them into SQL strings.

## Workflow

1. Identify areas where column names for queries (like order by, filters, or select fields) are determined at runtime.
2. Define an explicit whitelist of allowed column names as a slice or map, or use a strongly-typed enum/struct (e.g., `SearchableColumn` if the project uses that pattern).
3. Before executing the query, validate that the provided column name exists in the whitelist.
4. Return an appropriate error (e.g., `400 Bad Request` or an invalid input error) if the requested column is not allowed.
5. Once validated, it is safe to use `fmt.Sprintf` or string concatenation to build the query containing the column name.

## Inspect First

- `src/<domain>/handler/` to see how dynamic filters or sorting parameters are parsed.
- `src/<domain>/service/` or `src/<domain>/repository/` to see how those parameters translate into GORM queries.
- Existing security and validation patterns for user input.

## Anti-Patterns

- Directly interpolating user-provided strings into GORM's `.Where()`, `.Select()`, `.Order()`, or `.Group()` without validation.
- Relying solely on URL decoding or basic sanitization (like removing quotes) instead of explicit whitelisting.

## Done Criteria

- Dynamic column names are verified against an explicit, strict whitelist.
- The codebase rejects invalid column names with a clear error.
- Tests confirm that valid columns work and invalid columns are rejected.
