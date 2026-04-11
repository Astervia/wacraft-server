# Skill: GORM External Wrapper

Use this skill when needing to establish relationships between local models and external entities (e.g., from `github.com/Astervia/wacraft-core`) that lack those relationship definitions.

## Purpose

Enable the use of GORM's `.Preload()` functionality for external models that cannot be directly modified to include the necessary `gorm` tags (such as `foreignKey` or `references`).

## Core Problem

External libraries or core domain packages often define entities without knowledge of the specific relationships needed in the current service's database schema. When you try to load a local model that has a relationship to an external model, or vice-versa, GORM cannot automatically resolve the join or preload the nested data because the external struct lacks the required relationship tags.

## Preferred Pattern

Create a local wrapper struct that embeds the external entity and adds the necessary relationship fields with appropriate GORM tags.

## Workflow

1. Identify the external entity that needs to be preloaded or joined.
2. Create a new local wrapper struct, typically named `ExternalEntityWrapper` or `LocalExternalEntity` (e.g., `LocalWorkspace`).
3. Embed the external entity inside the wrapper struct.
4. Add the new relationship fields (e.g., `HasMany`, `BelongsTo`) directly to the wrapper struct.
5. Decorate these new relationship fields with the required `gorm` tags, explicitly defining `foreignKey` and `references` as needed.
6. Use this local wrapper struct in your GORM queries (`.Model(&LocalExternalEntity{}).Preload("RelatedModels").Find(...)`).
7. Once loaded, you can access the embedded external entity's fields directly or extract the core entity if needed.

## Example

Suppose you have an external `core.User` model, and you want to load a user with their local `Post` models.

```go
package model

import "github.com/Astervia/wacraft-core/domain"

// LocalUser wraps the external core.User to add local relationship definitions.
type LocalUser struct {
    domain.User // Embed the external entity
    Posts       []Post `gorm:"foreignKey:UserID;references:ID"` // Add the relationship
}

// In your repository/service:
// var user LocalUser
// db.Preload("Posts").First(&user, "id = ?", userID)
```

## Anti-Patterns

- Attempting to modify the external package's structs directly (which is often impossible or violates modularity).
- Manually querying the related tables and stitching the data together in memory when a simple wrapper and `.Preload()` would suffice.
- Using `db.Raw()` for complex joins when standard ORM relationship mapping via a wrapper is clearer and safer.

## Done Criteria

- The wrapper struct successfully embeds the external model.
- `gorm` tags on the wrapper correctly define the relationship.
- GORM queries using `.Preload()` execute without errors and populate the related data.
- The external package remains unmodified.
