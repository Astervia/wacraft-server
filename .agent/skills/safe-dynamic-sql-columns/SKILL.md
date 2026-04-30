# Skill: Safe Dynamic SQL Columns

Use this skill when dynamically selecting or filtering by column names in SQL queries, particularly when constructing queries using GORM's `.Where()` or `.Select()` with string formatting.

## Purpose

Prevent SQL injection vulnerabilities that arise when user input or untrusted data is used directly to specify database column names in queries.

## Core Problem

While parameterized queries (`?` in GORM) protect against SQL injection in column *values*, they cannot be used for column *names* or table *names*. If a feature allows filtering or sorting by dynamic fields (e.g., via query parameters), and that field name is blindly passed into `fmt.Sprintf("JSON_EXTRACT(%s, '$.key')", field)`, an attacker could inject arbitrary SQL.

## Preferred Pattern

Always map dynamic column keys to hardcoded string literals using a `switch` statement before inserting them into SQL strings.

## Workflow

1. Identify areas where column names for queries (like order by, filters, or select fields) are determined at runtime.
2. Define a `switch` block that matches the user-provided column name.
3. For each valid case, map the input to a hardcoded SQL column string literal.
4. Return an appropriate error (e.g., `400 Bad Request` or an invalid input error) in the `default` case to reject unrecognized columns.
5. Use the mapped hardcoded literal directly in the query without `fmt.Sprintf` or string concatenation for the identifier.

## Inspect First

- `src/<domain>/handler/` to see how dynamic filters or sorting parameters are parsed.
- `src/<domain>/service/` or `src/<domain>/repository/` to see how those parameters translate into GORM queries.
- Existing security and validation patterns for user input.

## Anti-Patterns

- Directly interpolating user-provided strings into GORM's `.Where()`, `.Select()`, `.Order()`, or `.Group()` without validation.
- Relying solely on URL decoding or basic sanitization (like removing quotes) instead of an explicit mapping.
- Using `fmt.Sprintf` to inject identifiers (like column names) even if the input is validated via an `IsValid()` method or a whitelist.

## Done Criteria

- Dynamic column names are explicitly mapped to hardcoded literals using a `switch` statement.
- The codebase rejects invalid column names with a clear error via the `default` case.
- Tests confirm that valid columns work and invalid columns are rejected.
