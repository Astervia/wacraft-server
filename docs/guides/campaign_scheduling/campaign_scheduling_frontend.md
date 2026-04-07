# Campaign Scheduling — Frontend Guide

This guide explains how to use the campaign scheduling API from the frontend.

---

## Concepts

A campaign now has a `status` field:

| Status      | What it means for the UI                                            |
| ----------- | ------------------------------------------------------------------- |
| `draft`     | Not scheduled. Ready to send manually or schedule.                  |
| `scheduled` | Will run automatically at `scheduled_at`. Can be cancelled.         |
| `running`   | Currently sending messages. Connect to WebSocket for live progress. |
| `completed` | All messages were attempted.                                        |
| `failed`    | An unexpected error stopped the run. May be rescheduled.            |
| `cancelled` | Run was cancelled mid-execution via WebSocket.                      |

---

## Typical Flows

### Flow 1: Schedule a campaign for later

```
1. Create campaign (POST /campaign) → status: draft
2. Add messages to campaign (POST /campaign/message)
3. Schedule (POST /campaign/schedule) → status: scheduled
4. At scheduled_at, the server executes automatically
5. Poll campaign status (GET /campaign) or subscribe via WebSocket
```

### Flow 2: Cancel a scheduled campaign

```
1. Unschedule (DELETE /campaign/schedule) → status: draft
```

### Flow 3: Connect to WebSocket during execution

```
1. Campaign is running (status: running)
2. Connect to WebSocket: GET /websocket/campaign/whatsapp/send/{campaignID}
3. Send "status" message to confirm "Sending"
4. Receive real-time progress frames
```

---

## Scheduling a Campaign

**Request:**

```http
POST /campaign/schedule
Authorization: Bearer <token>
X-Workspace-ID: <workspace-uuid>
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scheduled_at": "2026-04-05T15:00:00Z"
}
```

**Success response (200):**

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Spring Promotion",
    "status": "scheduled",
    "scheduled_at": "2026-04-05T15:00:00Z",
    "messaging_product_id": "...",
    "workspace_id": "...",
    "created_at": "...",
    "updated_at": "..."
}
```

**Error responses:**

| HTTP  | When                                         |
| ----- | -------------------------------------------- |
| `400` | Missing/invalid fields                       |
| `404` | Campaign not found in your workspace         |
| `409` | Campaign is already `running` or `completed` |

> `scheduled_at` must be a valid RFC 3339 UTC timestamp. Past timestamps are accepted (campaign will run on the next scheduler poll, ~30 seconds later at most).

---

## Cancelling a Schedule

**Request:**

```http
DELETE /campaign/schedule
Authorization: Bearer <token>
X-Workspace-ID: <workspace-uuid>
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Success response (200):** Campaign with `status: "draft"` and `scheduled_at: null`.

**Error responses:**

| HTTP  | When                                                           |
| ----- | -------------------------------------------------------------- |
| `400` | Missing/invalid fields                                         |
| `404` | Campaign not found                                             |
| `409` | Campaign is currently `running` (cancel via WebSocket instead) |

---

## Checking Campaign Status

Poll the campaign list or a single campaign to track status changes:

```http
GET /campaign
X-Workspace-ID: <workspace-uuid>
Authorization: Bearer <token>
```

The response includes `status` and `scheduled_at` for each campaign. A simple polling strategy:

```js
// Poll every 5 seconds while status is 'scheduled' or 'running'
async function pollUntilDone(campaignId) {
    while (true) {
        const resp = await fetch("/campaign", { headers: authHeaders });
        const campaigns = await resp.json();
        const c = campaigns.find((c) => c.id === campaignId);

        if (!c || ["completed", "failed", "cancelled", "draft"].includes(c.status)) break;

        await new Promise((r) => setTimeout(r, 5000));
    }
}
```

---

## Receiving Real-Time Progress via WebSocket

Once a campaign is `running` (whether triggered manually or by the scheduler), connect to the WebSocket endpoint to receive live progress:

```
GET /websocket/campaign/whatsapp/send/{campaignID}
```

### WebSocket message types (text, sent from frontend)

| Message  | Effect                                                                                 |
| -------- | -------------------------------------------------------------------------------------- |
| `ping`   | Server replies `pong` — use to keep connection alive                                   |
| `send`   | Starts the campaign send manually (rejected if `status=running` or `status=scheduled`) |
| `cancel` | Cancels an in-progress send                                                            |
| `status` | Server replies `Sending` or `NotSending`                                               |

### Connecting when campaign is already running (scheduled case)

If the scheduler started the campaign before you connect, just connect and listen — you will receive progress updates automatically:

```js
const ws = new WebSocket(`wss://your-server/websocket/campaign/whatsapp/send/${campaignId}`);

ws.onopen = () => {
    ws.send("status"); // optional: confirm it's running
};

ws.onmessage = (event) => {
    try {
        const data = JSON.parse(event.data);
        // data = { sent: N, successes: N, errors: N, total: N }
        updateProgressUI(data);
    } catch {
        // text messages: "Sending", "NotSending", "pong"
        console.log("text message:", event.data);
    }
};
```

### Progress frame structure

```json
{
    "sent": 42,
    "successes": 40,
    "errors": 2,
    "total": 100
}
```

- `sent` — messages attempted so far
- `successes` — successfully delivered
- `errors` — failed deliveries
- `total` — total messages in campaign

---

## Scheduling Workflow Example (JavaScript)

```js
const BASE = "https://your-server";
const HEADERS = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
    "X-Workspace-ID": workspaceId,
};

// 1. Create campaign
const campaign = await fetch(`${BASE}/campaign`, {
    method: "POST",
    headers: HEADERS,
    body: JSON.stringify({ name: "Spring Promo", messaging_product_id: mpId }),
}).then((r) => r.json());

// 2. Add messages (repeat for each contact)
await fetch(`${BASE}/campaign/message`, {
    method: "POST",
    headers: HEADERS,
    body: JSON.stringify({
        campaign_id: campaign.id,
        sender_data: {
            to: "+15551234567",
            message: {
                /* your message payload */
            },
        },
    }),
});

// 3. Schedule for tomorrow at noon UTC
const scheduledAt = new Date();
scheduledAt.setUTCDate(scheduledAt.getUTCDate() + 1);
scheduledAt.setUTCHours(12, 0, 0, 0);

const scheduled = await fetch(`${BASE}/campaign/schedule`, {
    method: "POST",
    headers: HEADERS,
    body: JSON.stringify({
        id: campaign.id,
        scheduled_at: scheduledAt.toISOString(),
    }),
}).then((r) => r.json());

console.log(scheduled.status); // "scheduled"
console.log(scheduled.scheduled_at); // "2026-04-05T12:00:00Z"

// 4. At execution time, connect to WebSocket for live updates
const ws = new WebSocket(`wss://your-server/websocket/campaign/whatsapp/send/${campaign.id}`);
ws.onmessage = (e) => {
    const progress = JSON.parse(e.data);
    console.log(`Progress: ${progress.sent}/${progress.total}`);
};
```

---

## Edge Cases and Notes

- **Past `scheduled_at`:** Accepted. The campaign will be queued on the next scheduler poll (within ~30 seconds by default).
- **Re-scheduling a failed campaign:** Allowed. Set the new `scheduled_at` via `POST /campaign/schedule`. Status moves back to `scheduled`.
- **Sending manually while scheduled:** Rejected with an error via the WebSocket `send` message. Unschedule first if you want manual control.
- **Sending manually while running:** Rejected (`campaign is running`). Cancel the current run via the WebSocket `cancel` message, then send again.
- **Server restart:** The schedule is safe across restarts. Any campaign in `running` state is automatically re-queued on startup.
