## 2026-04-11 - Reduced Lock Contention in Workspace Channel Manager Broadcast

**Learning:** Identified an anti-pattern in `src/websocket/workspace-manager/main.go` where sequential, disjoint `RWMutex` locks were being acquired in high-concurrency broadcast paths. The original code first locked to check `pubsub` state, unlocked, and then immediately locked again via `GetChannel` to retrieve the workspace channel if `pubsub` was null. This creates redundant locking overhead in hot paths.
**Action:** When writing high-frequency operational code (like message routing or broadcasting), combine related map and state lookups into a single mutex critical section. Pre-fetching the channel reference alongside the `pubsub` state eliminates the secondary `GetChannel` lock entirely, minimizing `RWMutex` contention.

## 2026-04-09 - Fixed N+1 Query in Workspace Member Retrieval by Updating External Entity

**Learning:** Found an N+1 query issue in the workspace member API (`src/workspace/handler/member.go`) where policies were being queried individually for each member within a loop. The core entity `WorkspaceMember` from `wacraft-core` lacked a `Policies` relationship, preventing direct use of `.Preload("Policies")`. Rather than using complex local wrapper structs, the cleanest approach is to update the core dependency to natively declare the `Policies` relationship.
**Action:** When working with external GORM entities that lack relationships causing N+1 query problems, it is best practice to recommend updating the core repository (e.g. `wacraft-core`) to include the missing relation. Once released, bump the dependency and use GORM's built-in `.Preload("RelationName")` directly on the entity for a clean and efficient solution.

## 2026-04-10 - GORM Preload Error Handling Anti-pattern

**Learning:** Trying to optimize by combining `.Preload("Relation")` into a `.First()` query can break error handling if not done carefully. If the initial `.First()` lookup was used to generate a specific error (e.g., 403 Forbidden for a missing relation) and the subsequent relations lookup was used to generate another (e.g., 500 for DB failure on policy lookup), combining them under one `.First().Error` check means a failure in the preload query will trigger the 403 error handler incorrectly. Also, preloads still run as a separate secondary SELECT query, so the database call savings are minimal compared to the risk of obfuscated errors.
**Action:** Do not try to combine lookup and preload queries if the individual errors are handled with different HTTP status codes or user messages, unless you explicitly inspect the type/source of the error returned.

## 2026-04-10 - WhatsApp Message Batching fixes N+1 Queries

**Learning:** Operations handling inbound data arrays (e.g. `msgs`) often loop over the array and emit async events, querying DB rules or webhooks inside the loop body resulting in severe N+1 queries under high traffic.
**Action:** When looping over payload batches to trigger downstream operations, extract the generic DB query (e.g. finding active webhooks for that workspace/event type) outside the loop and provide a `*Batch` alternative service function that takes `[]any` payloads to dispatch.

## 2026-04-12 - Fixed N+1 Query in Workspace Member Policy Creation

**Learning:** Identified an N+1 query issue during the creation of workspace member policies (e.g., in `AddMember`, `UpdateMemberPolicies`, workspace `Create`, and user `Register` handlers). The previous implementation used a loop over `workspace_model.AllPolicies` or user-defined policies to execute a `repository.Create` or `tx.Create` for each individual `WorkspaceMemberPolicy` record. In GORM, this incurs a database roundtrip and transaction overhead per policy.
**Action:** When inserting multiple rows of the same type, prepare a slice of the entities and use `tx.Create(&slice)` for batch insertion. This minimizes lock contention, lowers transaction overhead, and improves overall application performance during creation routines.

## 2026-04-14 - Eliminated N+1 Inserts in Webhook Handlers
**Learning:** Found N+1 database insertion issues in the webhook intake process (`src/webhook-in/handler/whatsapp-message.go` and `src/webhook-in/handler/whatsapp-message-status.go`). The handlers used concurrent goroutines inside a loop to process and immediately insert individual message and status records, creating transaction lock contention and excessive DB round-trips. Executing multiple queries concurrently on a single `*gorm.DB` transaction is also unsafe.
**Action:** Replaced single row inserts within the concurrent data processing loop with array aggregation. A single batch insert (`tx.Create(&msgs)` and `tx.Create(&statuses)`) is now executed post-loop, significantly reducing overhead and resolving unsafe concurrent transaction access.
