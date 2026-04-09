# Campaign Scheduling — UI Testing Guide

Covers every scenario that needs to be verified end-to-end in the UI. Each test case lists the prerequisite state, the steps to perform, and the expected result.

---

## Prerequisites

- A running server with the migration applied (`20260404000001_campaign_schedule`).
- A workspace with at least one messaging product and phone config configured.
- A valid auth token (JWT) and workspace ID.
- All curl examples can be substituted with the Swagger UI at `/swagger/index.html`.

Confirm the migration ran:

```sql
SELECT column_name FROM information_schema.columns
WHERE table_name = 'campaigns'
  AND column_name IN ('status', 'scheduled_at');
-- Should return 2 rows.
```

---

## 1. Campaign Status Field Is Present

**Goal:** Confirm `status` and `scheduled_at` appear in all campaign responses.

1. Create a campaign via `POST /campaign`.
2. Fetch it via `GET /campaign`.
3. Check the response body contains `"status": "draft"` and `"scheduled_at": null`.

**Pass:** Both fields present with the values above.

---

## 2. Schedule a Campaign (Future Time)

**Goal:** Happy path — campaign transitions to `scheduled`.

1. Create a campaign and add at least one message to it (`POST /campaign/message`).
2. Call `POST /campaign/schedule`:
    ```json
    {
        "id": "<campaign-uuid>",
        "scheduled_at": "<UTC timestamp at least 2 minutes from now>"
    }
    ```
3. Check the response: `"status": "scheduled"`, `"scheduled_at"` matches the value sent.
4. Fetch the campaign via `GET /campaign` — confirm the same values persist.

**Pass:** Campaign shows `status: scheduled` and a non-null `scheduled_at`.

---

## 3. Schedule with a Past Timestamp

**Goal:** Past timestamps are accepted and the campaign is queued for the next scheduler poll (within 30 seconds by default).

1. Create a campaign with at least one message.
2. Call `POST /campaign/schedule` with a `scheduled_at` value 10 minutes in the past.
3. Confirm the response is `200` with `status: scheduled`.
4. Wait up to 30 seconds, then poll `GET /campaign`.

**Pass:** Within 30 seconds the campaign status changes from `scheduled` → `running` → `completed`.

> If `CAMPAIGN_SCHEDULE_POLL_INTERVAL` is set to a shorter value (e.g. `5s`) the transition will be faster.

---

## 4. Automatic Execution at the Scheduled Time

**Goal:** Campaign fires without any manual action.

1. Create a campaign with several messages.
2. Schedule it for 2 minutes in the future.
3. Do not open any WebSocket connection.
4. At the scheduled time, poll `GET /campaign` every few seconds.

**Pass:** Status transitions `scheduled → running → completed` automatically. `scheduled_at` remains set in the response (it is not cleared on completion).

---

## 5. Cancel a Scheduled Campaign Before It Runs

**Goal:** `DELETE /campaign/schedule` resets the campaign to `draft`.

1. Schedule a campaign for 5+ minutes in the future.
2. Confirm status is `scheduled`.
3. Call `DELETE /campaign/schedule`:
    ```json
    { "id": "<campaign-uuid>" }
    ```
4. Check the response: `"status": "draft"`, `"scheduled_at": null`.
5. Wait past the original `scheduled_at` and confirm the campaign was **not** executed (status stays `draft`).

**Pass:** Campaign stays `draft` after the original time passes.

---

## 6. Connect to WebSocket During Scheduled Execution

**Goal:** A client connecting while the scheduler is running the campaign receives live progress frames.

1. Create a campaign with 10+ messages to give enough time to connect mid-run.
2. Schedule it for a time 30 seconds away (or use a past time and wait for the next poll).
3. As soon as status becomes `running` (poll frequently), connect to:
    ```
    GET /websocket/campaign/whatsapp/send/<campaignID>
    ```
4. Optionally send `status` — expect `Sending` back.
5. Observe incoming JSON frames:
    ```json
    { "sent": N, "successes": N, "errors": N, "total": N }
    ```

**Pass:** Progress frames are received in real time without the client having triggered the send itself.

---

## 7. Attempt to Manually Send a Scheduled Campaign via WebSocket

**Goal:** WebSocket `send` message is rejected when campaign is in `scheduled` or `running` status.

### 7a — Campaign is `scheduled`

1. Schedule a campaign for 5+ minutes in the future.
2. Connect to its WebSocket endpoint.
3. Send the text message `send`.
4. Expect an error response:
    ```json
    { "message": "campaign is scheduled", ... }
    ```

### 7b — Campaign is `running`

1. Wait for a campaign to enter `running` status (or trigger via a past timestamp).
2. Connect to the WebSocket.
3. Send `send`.
4. Expect an error response:
    ```json
    { "message": "campaign is running", ... }
    ```

**Pass:** Both cases return an error; the ongoing execution is unaffected.

---

## 8. Cancel an In-Progress Scheduled Campaign via WebSocket

**Goal:** The `cancel` WebSocket message stops an execution started by the scheduler.

1. Start a campaign with many messages via scheduling.
2. Connect to its WebSocket once status is `running`.
3. Send `cancel`.
4. Expect a `NotSending` text message back.
5. Poll `GET /campaign` — status should become `cancelled`.

**Pass:** Campaign stops mid-run and status is `cancelled`.

---

## 9. Reschedule a Failed Campaign

**Goal:** A `failed` campaign can be rescheduled.

1. Simulate a failure: schedule a campaign with an invalid/unreachable phone config, or
   manually set `status = 'failed'` in the database for a test campaign.
2. Call `POST /campaign/schedule` with a new `scheduled_at`.
3. Confirm the response is `200` with `status: scheduled`.

**Pass:** Campaign moves from `failed` back to `scheduled` and executes at the new time.

---

## 10. Conflict — Attempt to Schedule a Running Campaign (HTTP 409)

1. Wait for a campaign to reach `running` status.
2. Call `POST /campaign/schedule` on that campaign.
3. Expect HTTP `409` with a message indicating the campaign cannot be scheduled in its current status.

**Pass:** 409 response, campaign execution continues unaffected.

---

## 11. Conflict — Attempt to Schedule a Completed Campaign (HTTP 409)

1. Find (or wait for) a campaign with `status: completed`.
2. Call `POST /campaign/schedule`.
3. Expect HTTP `409`.

**Pass:** 409 response.

---

## 12. Conflict — Attempt to Unschedule a Running Campaign (HTTP 409)

1. Wait for a campaign to enter `running` status.
2. Call `DELETE /campaign/schedule`.
3. Expect HTTP `409` with a message saying to use the WebSocket `cancel` message instead.

**Pass:** 409 response, campaign execution continues unaffected.

---

## 13. Wrong Workspace — 404

1. Take a campaign UUID that belongs to workspace A.
2. Make requests (`POST /campaign/schedule`, `DELETE /campaign/schedule`) using the credentials of workspace B.
3. Expect HTTP `404` in both cases.

**Pass:** 404 response; no data leaked across workspaces.

---

## 14. Invalid Request Body — 400

Test each of the following; all should return HTTP `400`:

| Request                     | Bad Body                                                         |
| --------------------------- | ---------------------------------------------------------------- |
| `POST /campaign/schedule`   | `{}` (missing `id` and `scheduled_at`)                           |
| `POST /campaign/schedule`   | `{ "id": "not-a-uuid", "scheduled_at": "2026-01-01T00:00:00Z" }` |
| `DELETE /campaign/schedule` | `{}` (missing `id`)                                              |
| Either endpoint             | Malformed JSON (`{bad`)                                          |

**Pass:** All return `400`.

---

## 15. Server Restart Resilience

**Goal:** A scheduled campaign survives a server restart.

1. Schedule a campaign for 2 minutes in the future.
2. Restart the server before the scheduled time.
3. Confirm the server logs show: `Campaign scheduler worker started`.
4. At the scheduled time, the campaign executes normally.

**Pass:** Campaign runs on schedule after restart.

**Bonus — restart during execution:**

1. Schedule a campaign with many messages so execution takes several seconds.
2. Restart the server while the campaign is `running`.
3. After restart, confirm the server logs show: `Campaign scheduler: reset 1 running campaign(s) to scheduled (restart recovery)`.
4. Within 30 seconds the campaign re-runs from the beginning and reaches `completed`.

**Pass:** Campaign re-queued and completes after restart.

---

## 16. Redis Mode — No Duplicate Execution Across Instances

Only applicable when `SYNC_BACKEND=redis` with two or more server instances.

1. Run two instances connected to the same PostgreSQL and Redis.
2. Schedule a campaign with many messages (long execution).
3. Ensure both instances are running when the scheduled time arrives.
4. Check the logs of both instances.

**Pass:** Exactly one instance logs `Campaign scheduler: starting campaign <id>`. The other instance does not process it. The campaign reaches `completed` exactly once.

---

## Quick Reference — Status Transitions

| From        | Action                      | To            |
| ----------- | --------------------------- | ------------- |
| `draft`     | `POST /campaign/schedule`   | `scheduled`   |
| `scheduled` | `DELETE /campaign/schedule` | `draft`       |
| `scheduled` | Scheduler picks up          | `running`     |
| `running`   | Execution finishes          | `completed`   |
| `running`   | Execution errors            | `failed`      |
| `running`   | WebSocket `cancel`          | `cancelled`   |
| `failed`    | `POST /campaign/schedule`   | `scheduled`   |
| `running`   | `POST /campaign/schedule`   | 409 (blocked) |
| `completed` | `POST /campaign/schedule`   | 409 (blocked) |
| `running`   | `DELETE /campaign/schedule` | 409 (blocked) |
