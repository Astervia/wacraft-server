# Skill: Safe Dynamic SQL Columns

Use this skill when dynamically selecting or filtering by column names in SQL queries, particularly when constructing queries using GORM's `.Where()` or `.Select()` with string formatting.

## Purpose

Prevent SQL injection vulnerabilities that arise when user input or untrusted data is used directly to specify database column names in queries.

## Core Problem

While parameterized queries (`?` in GORM) protect against SQL injection in column *values*, they cannot be used for column *names* or table *names*. If a feature allows filtering or sorting by dynamic fields (e.g., via query parameters), and that field name is blindly passed into `fmt.Sprintf("JSON_EXTRACT(%s, '$.key')", field)`, an attacker could inject arbitrary SQL.

## Preferred Pattern

Use a `switch` statement to map user input to hardcoded string literals for database identifiers. Avoid `fmt.Sprintf` for identifiers even if the input is validated via an `IsValid()` method.

## Workflow

1. Identify areas where column names for queries (like order by, filters, or select fields) are determined at runtime.
2. Replace any string interpolation (`fmt.Sprintf`) of column names with a `switch` statement.
3. Inside the `switch`, map the valid user input strings directly to hardcoded SQL identifier strings.
4. Add a `default` case to return an appropriate error (e.g., `400 Bad Request` or an invalid input error) if the requested column is not allowed.
5. Use the mapped hardcoded string literal in the query builder.

## Inspect First

- `src/<domain>/handler/` to see how dynamic filters or sorting parameters are parsed.
- `src/<domain>/service/` or `src/<domain>/repository/` to see how those parameters translate into GORM queries.
- Existing security and validation patterns for user input.

## Anti-Patterns

- Directly interpolating user-provided strings into GORM's `.Where()`, `.Select()`, `.Order()`, or `.Group()`.
- Using `fmt.Sprintf` or string concatenation for column/table identifiers, even after validating the input against a whitelist or `IsValid()` method.
- Relying solely on URL decoding or basic sanitization (like removing quotes) instead of explicit mapping.

## Done Criteria

- Dynamic column names are verified against an explicit, strict whitelist.
- The codebase rejects invalid column names with a clear error.
- Tests confirm that valid columns work and invalid columns are rejected.
