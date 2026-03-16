---
name: go-update-endpoint
description: Use this skill when creating or modifying a Go GORM update endpoint (PUT/PATCH handler). Applies when the user asks to "add an update handler", "create a PUT endpoint", "update a GORM entity", or fixes a bug where setting a boolean/integer field to its zero value (false, 0) is silently ignored.
version: 1.0.0
---

# Go GORM Update Endpoint Pattern

This skill codifies the correct way to build update handlers in this codebase so that GORM
never silently ignores zero-value fields (`false`, `0`, `""`).

## The Core Problem

`db.Updates(struct)` skips every field whose value equals the Go zero value:

| Go type | Zero value silently skipped |
|---------|-----------------------------|
| `bool`  | `false`                     |
| `int`   | `0`                         |
| `string`| `""`                        |

This means "disable this webhook" (`is_active = false`) or "allow 0 retries"
(`max_retries = 0`) are silently dropped by GORM — the DB is never updated.

`db.Updates(map[string]interface{}{...})` avoids this, but then GORM cannot apply
`serializer:json` tags on jsonb columns automatically, which introduces a different bug.

## The Correct Pattern

Use **pointer types** (`*bool`, `*int`, `*string`) on every entity field that:
- is optional in an update request, **and**
- has a meaningful zero value that a caller may legitimately set.

`db.Updates(struct)` skips `nil` pointers but **does update non-nil pointers even when
the pointed-to value is `false` or `0`** — this is exactly what we need.

---

## Step-by-step Checklist

### 1. Update model (`model/update.go`)

All optional fields must be pointer types so `nil` means "not provided":

```go
type UpdateFoo struct {
    Name        string            `json:"name,omitempty"`
    Timeout     *int              `json:"timeout,omitempty" validate:"omitempty,gte=1,lte=60"`
    MaxRetries  *int              `json:"max_retries,omitempty" validate:"omitempty,gte=0,lte=10"`
    IsActive    *bool             `json:"is_active,omitempty"`
    CustomMeta  map[string]string `json:"custom_meta,omitempty"`

    common_model.RequiredID
}
```

### 2. Entity (`entity/foo.go`)

Mirror the same pointer types on every field that the update model exposes as optional:

```go
type Foo struct {
    Name       string            `json:"name,omitempty" gorm:"not null"`
    Timeout    *int              `json:"timeout,omitempty" gorm:"default:30"`
    MaxRetries *int              `json:"max_retries,omitempty" gorm:"default:3"`
    IsActive   *bool             `json:"is_active,omitempty" gorm:"default:true"`
    CustomMeta map[string]string `json:"custom_meta,omitempty" gorm:"serializer:json;type:jsonb"`

    common_model.Audit
}
```

> **Keep non-pointer types** for fields that are always required on creation and where
> the zero value is never a valid update (e.g. a `NOT NULL` string like `url`).

### 3. Handler (`handler/update.go`)

Assign pointer fields **directly** from the update model to the entity struct — no
intermediate nil-check + dereference blocks needed:

```go
updateData := foo_entity.Foo{
    // Non-pointer fields: GORM skips empty strings automatically — OK here
    // because callers omit the field when they don't want to change it.
    Name:    editFoo.Name,

    // Pointer fields: assigned directly. nil → GORM skips; &false → GORM updates.
    Timeout:    editFoo.Timeout,
    MaxRetries: editFoo.MaxRetries,
    IsActive:   editFoo.IsActive,

    // Complex/jsonb fields: pointer guards GORM serializer correctly.
    CustomMeta: editFoo.CustomMeta,
}

foo, err := repository.Updates(updateData, &foo_entity.Foo{
    Audit:       common_model.Audit{ID: editFoo.ID},
    WorkspaceID: &workspace.ID,
}, database.DB)
```

### 4. Create handler (`handler/create.go`)

When the entity field is now `*T`, provide explicit defaults on creation rather than
relying on the Go zero value:

```go
isActive := true
entity := foo_entity.Foo{
    IsActive:   &isActive,       // explicit default
    MaxRetries: newFoo.MaxRetries, // nil → DB default via gorm:"default:3"
}
```

If the caller can omit the field and the DB default is acceptable, just leave the
pointer `nil`; GORM will use the `default:` tag on `INSERT`.

### 5. Code that reads the field

Dereference with a nil guard wherever the field is consumed:

```go
// Queue / service layer
if foo.IsActive != nil && !*foo.IsActive {
    return nil // skip inactive
}

// Arithmetic on *int
maxAttempts := *foo.MaxRetries + 1

// time.Duration cast
delay := time.Duration(*foo.RetryDelayMs) * time.Millisecond
```

---

## Anti-patterns to avoid

| Anti-pattern | Problem |
|---|---|
| `if editFoo.IsActive != nil { updateData.IsActive = *editFoo.IsActive }` then `IsActive bool` on entity | Dereferences correctly but GORM still skips `false` |
| `db.Updates(map[string]interface{}{...})` for jsonb columns | GORM does not apply `serializer:json` for map-based updates |
| `db.Select("*").Updates(struct)` | Updates ALL columns including ones not in the request |

---

## Real example in this codebase

See `src/webhook/handler/update.go` + `wacraft-core/src/webhook/entity/webhook.go` for
the reference implementation using `IsActive *bool`, `MaxRetries *int`, and
`RetryDelayMs *int`.
