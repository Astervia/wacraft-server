# Skill: Go Update Endpoint

Use this skill when creating or modifying a Go GORM update endpoint such as a
`PUT` or `PATCH` handler.

## Purpose

Build update handlers that do not silently drop meaningful zero values like
`false`, `0`, or `""` when persisted through GORM.

## Core Problem

`db.Updates(struct)` skips fields whose value is the Go zero value.

That is a bug for optional update fields such as:

- `false` for feature toggles
- `0` for retry or delay values
- empty string when clearing a field is valid

`db.Updates(map[string]interface{}{...})` can force updates, but it bypasses some
GORM struct behavior such as serializer handling for JSON columns.

## Preferred Pattern

Use pointer types for optional update fields whose zero values are meaningful.

- `nil` means "field not provided"
- non-nil means "persist this exact value", even when it is `false` or `0`

Mirror those pointer fields in both:

- the update request model
- the entity struct passed to `Updates`

## Workflow

1. Inspect the update request model and identify optional fields with meaningful zero values.
2. Convert those fields to pointers in the update model if they are not already pointers.
3. Mirror the same pointer types on the entity fields used in update persistence.
4. Assign pointer fields directly from request model to entity update struct.
5. Keep create flows explicit about defaults when a field is now `*T`.
6. Add nil-guarded dereferences anywhere the field is consumed later.
7. Add or update tests that prove `false`, `0`, or empty values persist correctly.

## Inspect First

- the owning `handler/update.go`
- the update request model in `model/` or the owning package
- the entity definition used by GORM
- any create handler or service that constructs the entity
- tests covering update behavior

## Anti-Patterns

- dereferencing into a non-pointer entity field before calling `Updates`
- switching to map-based updates for JSON or serializer-backed columns without a strong reason
- `Select("*").Updates(struct)` when the request only intended partial updates

## Done Criteria

- omitted fields are unchanged
- explicit zero values are persisted
- serializer-backed columns still behave correctly
- create paths still apply sensible defaults
- tests prove the intended update semantics
