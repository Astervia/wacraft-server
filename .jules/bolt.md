## 2026-04-09 - Fixed N+1 Query in Workspace Member Retrieval
**Learning:** Found an N+1 query issue in the workspace member API (`src/workspace/handler/member.go`) where policies were being queried individually for each member within a loop.
**Action:** Always check loop structures for database calls. Used Go's slice mapping and a single `IN` clause query to retrieve all associated policies, then mapped them in memory, avoiding N+1 queries during list retrieval operations.
