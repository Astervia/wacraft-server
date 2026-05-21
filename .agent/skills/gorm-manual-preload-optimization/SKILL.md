# Skill: GORM Manual Preload Optimization

Use this skill when implementing or refactoring 'get-or-save' database operations using GORM, where a record is created and then immediately returned with its relationships loaded.

## Purpose

To eliminate redundant database round-trips caused by GORM `.Preload()` calls immediately following a record insertion.

## Problem

When creating a new record that belongs to another entity (e.g., a `WebhookDelivery` belonging to a `Webhook`), a common anti-pattern is to:
1. Create the record (`db.Create(&entity)`).
2. Query the database again to load the relationship (`db.Preload("Parent").First(&entity, entity.ID)`).

This second query is often unnecessary if the parent entity data is already available in memory (which is typically the case when you are creating the child entity).

## Optimization Workflow

Instead of relying on `.Preload()` after insertion, manually populate the relationship fields (pointer assignments) of the created entity using the available in-memory data.

### Example

**Inefficient Approach (Anti-pattern):**

```go
delivery := &domain.WebhookDelivery{
    WebhookID: webhook.ID, // We already have 'webhook' in memory
    Payload:   payload,
}
db.Create(delivery)

// REDUNDANT DB QUERY:
db.Preload("Webhook").First(delivery, delivery.ID)
return delivery, nil
```

**Optimized Approach (The Skill):**

```go
delivery := &domain.WebhookDelivery{
    WebhookID: webhook.ID,
    Webhook:   webhook, // MANUALLY ASSIGN THE POINTER
    Payload:   payload,
}
db.Create(delivery)

// The 'delivery' object now has its 'Webhook' relationship "loaded" without an extra DB query.
return delivery, nil
```

## Done Criteria

- The code does not use `.Preload()` immediately after `db.Create()` if the related data is already available in memory.
- Pointer fields representing the relationships are explicitly assigned during struct initialization or immediately after creation.
- Tests verify that the returned entity contains the expected relational data.
