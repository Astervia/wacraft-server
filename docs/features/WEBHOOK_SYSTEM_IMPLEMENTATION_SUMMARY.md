# Webhook System Implementation Summary

This document describes the enhanced webhook system with reliability features, security improvements, and event filtering capabilities.

## API Changes

### Create Webhook - `POST /webhook`

#### New Request Fields

| Field             | Type    | Required | Default | Description                                  |
| ----------------- | ------- | -------- | ------- | -------------------------------------------- |
| `signing_enabled` | boolean | No       | `false` | Enable HMAC-SHA256 request signing           |
| `max_retries`     | integer | No       | `3`     | Max retry attempts (0-10)                    |
| `retry_delay_ms`  | integer | No       | `1000`  | Base retry delay in milliseconds (100-60000) |
| `custom_headers`  | object  | No       | `null`  | Custom headers to send with requests         |
| `event_filter`    | object  | No       | `null`  | Filter to match specific events              |

#### New Response Fields

When `signing_enabled: true`, the response includes:

| Field            | Type   | Description                                         |
| ---------------- | ------ | --------------------------------------------------- |
| `signing_secret` | string | The signing secret (only returned once on creation) |

#### Example Request

```json
{
    "url": "https://example.com/webhook",
    "http_method": "POST",
    "event": "receive_whatsapp_message",
    "signing_enabled": true,
    "max_retries": 5,
    "retry_delay_ms": 2000,
    "custom_headers": {
        "X-Custom-Header": "value"
    },
    "event_filter": {
        "logic": "AND",
        "conditions": [
            {
                "path": "data.type",
                "operator": "equals",
                "value": "text"
            }
        ]
    }
}
```

#### Example Response

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://example.com/webhook",
    "http_method": "POST",
    "event": "receive_whatsapp_message",
    "signing_enabled": true,
    "signing_secret": "whsec_a1b2c3d4e5f6...",
    "max_retries": 5,
    "retry_delay_ms": 2000,
    "is_active": true,
    "custom_headers": {
        "X-Custom-Header": "value"
    },
    "event_filter": {
        "logic": "AND",
        "conditions": [
            {
                "path": "data.type",
                "operator": "equals",
                "value": "text"
            }
        ]
    },
    "circuit_state": "closed",
    "failure_count": 0,
    "created_at": "2026-02-04T12:00:00Z",
    "updated_at": "2026-02-04T12:00:00Z"
}
```

---

### Update Webhook - `PUT /webhook`

#### New Updatable Fields

| Field            | Type    | Description                                  |
| ---------------- | ------- | -------------------------------------------- |
| `max_retries`    | integer | Max retry attempts (0-10)                    |
| `retry_delay_ms` | integer | Base retry delay in milliseconds (100-60000) |
| `is_active`      | boolean | Enable/disable webhook                       |
| `custom_headers` | object  | Custom headers to send with requests         |
| `event_filter`   | object  | Filter to match specific events              |

Note: `signing_enabled` and `signing_secret` cannot be updated after creation.

#### Example Request

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "is_active": false,
    "max_retries": 3
}
```

---

### Test Webhook - `POST /webhook/test` (New)

Sends a test payload to a webhook and returns the result.

#### Request

| Field        | Type   | Required | Description                    |
| ------------ | ------ | -------- | ------------------------------ |
| `webhook_id` | uuid   | Yes      | The webhook to test            |
| `payload`    | object | No       | Custom test payload (optional) |

#### Response

| Field           | Type    | Description                                |
| --------------- | ------- | ------------------------------------------ |
| `success`       | boolean | Whether the request succeeded (2xx status) |
| `status_code`   | integer | HTTP response status code                  |
| `response`      | object  | Parsed JSON response (if valid JSON)       |
| `response_body` | string  | Raw response body                          |
| `duration_ms`   | integer | Request duration in milliseconds           |
| `headers_sent`  | object  | Headers that were sent with the request    |
| `error`         | string  | Error message if request failed            |

#### Example Request

```json
{
    "webhook_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Example Response

```json
{
    "success": true,
    "status_code": 200,
    "response": { "received": true },
    "response_body": "{\"received\": true}",
    "duration_ms": 245,
    "headers_sent": {
        "Content-Type": "application/json",
        "X-Wacraft-Test": "true",
        "X-Wacraft-Event": "receive_whatsapp_message",
        "X-Wacraft-Signature": "v1=abc123...",
        "X-Wacraft-Timestamp": "1707048000"
    }
}
```

---

### Get Webhooks - `GET /webhook`

#### New Response Fields

Each webhook now includes:

| Field               | Type     | Description                                          |
| ------------------- | -------- | ---------------------------------------------------- |
| `signing_enabled`   | boolean  | Whether signing is enabled                           |
| `max_retries`       | integer  | Max retry attempts                                   |
| `retry_delay_ms`    | integer  | Base retry delay in milliseconds                     |
| `is_active`         | boolean  | Whether webhook is active                            |
| `custom_headers`    | object   | Custom headers                                       |
| `event_filter`      | object   | Event filter configuration                           |
| `circuit_state`     | string   | Circuit breaker state: `closed`, `open`, `half_open` |
| `failure_count`     | integer  | Consecutive failure count                            |
| `last_failure_at`   | datetime | Last failure timestamp                               |
| `circuit_opened_at` | datetime | When circuit was opened                              |

Note: `signing_secret` is never returned in GET requests.

---

## Event Filter Configuration

### Filter Structure

```typescript
interface EventFilter {
    logic?: "AND" | "OR"; // Default: "AND"
    conditions: FilterCondition[];
}

interface FilterCondition {
    path: string; // JSON path (e.g., "data.message.type")
    operator: FilterOperator;
    value?: any; // Not required for "exists" operator
}

type FilterOperator = "equals" | "contains" | "regex" | "exists";
```

### Operators

| Operator   | Description        | Example                                                           |
| ---------- | ------------------ | ----------------------------------------------------------------- |
| `equals`   | Exact match        | `{"path": "data.type", "operator": "equals", "value": "text"}`    |
| `contains` | Substring match    | `{"path": "data.body", "operator": "contains", "value": "hello"}` |
| `regex`    | Regular expression | `{"path": "data.from", "operator": "regex", "value": "^\\+1"}`    |
| `exists`   | Field exists       | `{"path": "data.media", "operator": "exists"}`                    |

### Examples

**Filter text messages only:**

```json
{
    "logic": "AND",
    "conditions": [{ "path": "data.type", "operator": "equals", "value": "text" }]
}
```

**Filter messages from US numbers:**

```json
{
    "logic": "AND",
    "conditions": [{ "path": "data.from", "operator": "regex", "value": "^\\+1" }]
}
```

**Filter messages with media OR from specific contact:**

```json
{
    "logic": "OR",
    "conditions": [
        { "path": "data.media", "operator": "exists" },
        { "path": "data.from", "operator": "equals", "value": "+1234567890" }
    ]
}
```

---

## Signature Verification (For Webhook Consumers)

When `signing_enabled` is true, requests include signature headers:

| Header                | Description                                |
| --------------------- | ------------------------------------------ |
| `X-Wacraft-Signature` | HMAC-SHA256 signature in format `v1={hex}` |
| `X-Wacraft-Timestamp` | Unix timestamp (seconds)                   |

### Verification Steps

1. Extract timestamp from `X-Wacraft-Timestamp` header
2. Construct message: `v1:{timestamp}:{raw_request_body}`
3. Compute HMAC-SHA256 of message using `signing_secret`
4. Compare with signature from `X-Wacraft-Signature` (constant-time comparison)
5. Reject if timestamp is older than 5 minutes

### Example (Node.js)

```javascript
const crypto = require("crypto");

function verifySignature(secret, timestamp, body, signature) {
    const message = `v1:${timestamp}:${body}`;
    const expected = "v1=" + crypto.createHmac("sha256", secret).update(message).digest("hex");

    // Constant-time comparison
    return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}

function isTimestampValid(timestamp, maxAgeSeconds = 300) {
    const now = Math.floor(Date.now() / 1000);
    return now - parseInt(timestamp) <= maxAgeSeconds;
}
```

### Example (Python)

```python
import hmac
import hashlib
import time

def verify_signature(secret: str, timestamp: str, body: str, signature: str) -> bool:
    message = f"v1:{timestamp}:{body}"
    expected = "v1=" + hmac.new(
        secret.encode(),
        message.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

def is_timestamp_valid(timestamp: str, max_age_seconds: int = 300) -> bool:
    return int(time.time()) - int(timestamp) <= max_age_seconds
```

---

## Circuit Breaker States

The circuit breaker protects against failing endpoints and prevents resource exhaustion by temporarily stopping requests to consistently failing webhooks.

### State Transitions

```
┌─────────┐
│ closed  │ ──[5 failures]──> ┌──────┐
└─────────┘                    │ open │
     ▲                         └──────┘
     │                             │
     │                             │ [30s timeout]
     │                             ▼
     │                        ┌──────────┐
     └────[success]───────────│half_open │
                              └──────────┘
                                   │
                              [failure]
                                   │
                                   ▼
                              ┌──────┐
                              │ open │
                              └──────┘
```

### State Behavior

| State       | Description      | Behavior                            | When Transitions                          |
| ----------- | ---------------- | ----------------------------------- | ----------------------------------------- |
| `closed`    | Normal operation | All requests are sent normally      | Opens after 5 consecutive failures        |
| `open`      | Circuit tripped  | All requests are blocked (not sent) | Transitions to half_open after 30 seconds |
| `half_open` | Testing recovery | One request allowed as a test       | Success → closed, Failure → open          |

### How It Works

#### 1. Closed State (Normal Operation)

- Webhook delivers normally
- Failure counter increments on each failed delivery (non-2xx status or error)
- Success resets failure counter to 0
- After 5 consecutive failures, transitions to **open**

#### 2. Open State (Circuit Tripped)

- Worker checks circuit before attempting delivery
- If circuit is open, delivery is skipped (not attempted)
- Delivery remains in queue with status `pending` or `attempted`
- After 30 seconds, automatically transitions to **half_open**

#### 3. Half-Open State (Recovery Test)

- Worker allows ONE request through as a test
- If successful (2xx status):
    - Circuit closes
    - Failure counter resets to 0
    - Normal operation resumes
- If failed:
    - Circuit immediately reopens
    - 30-second timeout starts again

### Tracked Fields

| Field               | Type      | Description                                  |
| ------------------- | --------- | -------------------------------------------- |
| `circuit_state`     | string    | Current state: `closed`, `open`, `half_open` |
| `failure_count`     | integer   | Consecutive failures (resets on success)     |
| `last_failure_at`   | timestamp | When the last failure occurred               |
| `circuit_opened_at` | timestamp | When circuit was opened (null if closed)     |

### Configuration Constants

| Constant           | Value      | Description                                           |
| ------------------ | ---------- | ----------------------------------------------------- |
| `FailureThreshold` | 5          | Number of consecutive failures before opening circuit |
| `RecoveryTimeout`  | 30 seconds | Time to wait before testing recovery                  |

### Example Flow

1. **Initial State**: `closed`, `failure_count: 0`
2. **Delivery fails**: `closed`, `failure_count: 1`
3. **4 more failures**: `closed`, `failure_count: 5`
4. **Circuit opens**: `open`, `circuit_opened_at: now`
5. **Worker skips deliveries for 30s**
6. **After 30s**: `half_open`
7. **Test delivery succeeds**: `closed`, `failure_count: 0`

### UI Considerations

- Show circuit state badge prominently
- Display failure count when > 0
- Show time until recovery attempt when circuit is open
- Allow manual circuit reset (admin action)

---

## Delivery System

### How It Works

1. Events trigger `SendAllByQuery()` which enqueues deliveries
2. Background worker polls every 5 seconds for pending deliveries
3. Worker processes up to 10 deliveries concurrently
4. Failed deliveries are retried with exponential backoff

### Retry Behavior

- **Base delay**: Configured via `retry_delay_ms` (default: 1000ms)
- **Backoff formula**: `base_delay * 2^attempt_count`
- **Maximum delay**: 1 hour
- **Max attempts**: `max_retries + 1` (initial attempt + retries)

### Delivery Statuses

| Status        | Description                           |
| ------------- | ------------------------------------- |
| `pending`     | Awaiting first attempt                |
| `attempted`   | Has been tried, awaiting retry        |
| `succeeded`   | Successfully delivered (2xx response) |
| `dead_letter` | Max retries exhausted                 |

---

## Request Headers

Webhook requests include these headers:

| Header                  | Always Sent        | Description                 |
| ----------------------- | ------------------ | --------------------------- |
| `Content-Type`          | Yes                | `application/json`          |
| `X-Wacraft-Delivery-ID` | Yes                | Unique delivery ID          |
| `X-Wacraft-Event`       | Yes                | Event type                  |
| `X-Wacraft-Attempt`     | Yes                | Attempt number (1, 2, 3...) |
| `Authorization`         | If configured      | Authorization header value  |
| `X-Wacraft-Signature`   | If signing enabled | HMAC-SHA256 signature       |
| `X-Wacraft-Timestamp`   | If signing enabled | Unix timestamp              |
| Custom headers          | If configured      | User-defined headers        |

---

## Frontend UI Recommendations

### Webhook Creation Form

Add new fields:

1. **Security Section**
    - Toggle: "Enable request signing"
    - Display signing secret after creation (with copy button, show once warning)

2. **Reliability Section**
    - Number input: "Max retries" (0-10, default 3)
    - Number input: "Retry delay (ms)" (100-60000, default 1000)

3. **Custom Headers Section**
    - Key-value pair inputs for custom headers
    - Add/remove buttons for multiple headers

4. **Event Filter Section**
    - Logic selector: AND/OR
    - Condition builder with:
        - Path input (e.g., "data.type")
        - Operator dropdown (equals, contains, regex, exists)
        - Value input (hidden for "exists")
    - Add/remove condition buttons

### Webhook List/Detail View

Display new fields:

1. **Status indicators**
    - Active/Inactive badge
    - Circuit state badge (closed=green, open=red, half_open=yellow)
    - Failure count if > 0

2. **Configuration display**
    - Signing enabled indicator
    - Retry configuration
    - Custom headers (expandable)
    - Event filter (formatted/expandable)

### Test Webhook Feature

Add a "Test" button that:

1. Opens a modal with optional custom payload input
2. Calls `POST /webhook/test`
3. Displays results: success/failure, status code, response, headers sent, duration

### Webhook Logs Enhancement

Show additional fields:

- Delivery ID (link to delivery if applicable)
- Attempt number
- Duration (ms)
- Signature sent indicator
- Request URL
- Request headers (expandable)

---

## Database Schema Changes

New tables:

- `webhook_deliveries` - Queue for pending deliveries

Modified tables:

- `webhooks` - Added signing, retry, circuit breaker fields
- `webhook_logs` - Added delivery tracking fields

Migrations run automatically via GORM AutoMigrate.

---

## Backwards Compatibility

- Existing webhooks work with defaults: `is_active=true`, `signing_enabled=false`, `max_retries=3`
- All new fields are optional in API requests
- Existing `SendAllByQuery` calls automatically use the queue system

---

## Next Steps: Horizontal Scalability

The current implementation is designed for single-instance deployment. To support horizontal scaling with multiple server instances, the following enhancements are recommended:

### 1. Distributed Locking for Delivery Processing

**Problem**: Multiple workers across different instances may process the same delivery simultaneously.

**Solution**: Implement distributed locking using one of these approaches:

#### Option A: PostgreSQL Advisory Locks

```sql
-- Worker attempts to lock delivery before processing
SELECT * FROM webhook_deliveries
WHERE id = ?
AND pg_try_advisory_lock(id::bigint);
```

**Pros**: No additional infrastructure, uses existing database
**Cons**: Lock contention on database, potential bottleneck

#### Option B: Redis Distributed Locks

```go
// Use Redis with Redlock algorithm
lock, err := redisClient.SetNX(ctx,
    "webhook:delivery:"+deliveryID,
    workerID,
    30*time.Second)
```

**Pros**: Fast, designed for distributed locking
**Cons**: Requires Redis infrastructure

#### Option C: Database Row-Level Locking with SELECT FOR UPDATE

```sql
-- Worker claims deliveries atomically
UPDATE webhook_deliveries
SET status = 'processing',
    worker_id = ?,
    locked_at = NOW()
WHERE id IN (
    SELECT id FROM webhook_deliveries
    WHERE status IN ('pending', 'attempted')
    AND next_attempt_at <= NOW()
    LIMIT 50
    FOR UPDATE SKIP LOCKED
)
RETURNING *;
```

**Pros**: Simple, built-in PostgreSQL feature, no race conditions
**Cons**: Requires adding `worker_id` and `locked_at` columns

**Recommended**: Option C (SELECT FOR UPDATE SKIP LOCKED) - most elegant for PostgreSQL

### 2. Worker Coordination

**Problem**: Circuit breaker state changes by one worker may not be visible to others immediately.

**Solutions**:

#### Option A: Database-Driven Circuit Breaker (Current)

- Circuit state stored in database
- All workers read from DB before processing
- Already implemented, works across instances
- No changes needed

#### Option B: Redis Pub/Sub for State Changes

- Publish circuit state changes to Redis channel
- Workers subscribe and update in-memory cache
- Reduces database reads
- Requires Redis infrastructure

**Recommended**: Keep current database-driven approach, optionally add Redis caching layer later

### 3. Delivery Queue Optimization

**Problem**: All workers polling the same table may cause contention.

**Solutions**:

#### Option A: Partitioned Queues

- Partition deliveries by `webhook_id % num_partitions`
- Each worker group handles specific partitions
- Reduces contention

```sql
-- Worker 1 handles partition 0, 3, 6, 9...
WHERE (webhook_id::bigint % 10) IN (0, 3, 6, 9)
```

#### Option B: Message Queue (Redis/RabbitMQ/SQS)

- Replace database polling with message queue
- Push deliveries to queue on creation
- Workers consume from queue
- Better performance, scales horizontally

**Recommended**: Start with partitioned queues, migrate to message queue if needed

### 4. Idempotency Improvements

**Problem**: Network failures during processing may cause duplicate delivery attempts.

**Solution**: Add processing state tracking

```go
// Add to WebhookDelivery entity
ProcessingStartedAt *time.Time `json:"processing_started_at,omitempty"`
ProcessorID         string     `json:"processor_id,omitempty"`

// Detect stale locks (processing > 5 minutes)
WHERE status = 'processing'
AND processing_started_at < NOW() - INTERVAL '5 minutes'
```

### 5. Monitoring and Observability

**Recommended additions**:

- **Metrics**: Delivery success rate, latency, queue depth per webhook
- **Distributed Tracing**: Track delivery lifecycle across services
- **Dead Letter Queue Dashboard**: View and retry failed deliveries
- **Worker Health Checks**: Detect and restart stalled workers

### 6. Configuration for Scaling

Add environment variables:

```bash
# Worker configuration
WEBHOOK_WORKER_ENABLED=true          # Enable/disable worker on this instance
WEBHOOK_WORKER_POOL_SIZE=10          # Concurrent deliveries
WEBHOOK_WORKER_PARTITION_ID=0        # Partition assignment (0-9)
WEBHOOK_WORKER_PARTITION_COUNT=10    # Total partitions

# Queue configuration
WEBHOOK_POLL_INTERVAL=5s             # How often to check for deliveries
WEBHOOK_BATCH_SIZE=50                # Max deliveries per batch
```

### 7. Migration Path

**Phase 1** (Current): Single instance, database polling
**Phase 2**: Add SELECT FOR UPDATE SKIP LOCKED (enables safe multi-instance)
**Phase 3**: Add partitioned queues (reduces contention)
**Phase 4**: Migrate to Redis/SQS message queue (maximum scalability)

### Implementation Priority

| Enhancement                   | Priority | Complexity | Impact                          |
| ----------------------------- | -------- | ---------- | ------------------------------- |
| SELECT FOR UPDATE SKIP LOCKED | High     | Low        | Enables safe horizontal scaling |
| Processing state tracking     | High     | Low        | Prevents duplicate processing   |
| Worker configuration          | High     | Low        | Allows gradual rollout          |
| Partitioned queues            | Medium   | Medium     | Improves performance at scale   |
| Distributed tracing           | Medium   | Medium     | Better observability            |
| Message queue migration       | Low      | High       | Required for very high scale    |

### Code Changes Required

**Minimal changes for multi-instance support:**

1. Update `GetPendingDeliveries()` to use `FOR UPDATE SKIP LOCKED`
2. Add `processing_started_at` and `processor_id` columns
3. Add stale lock detection in worker loop
4. Add environment variable for `WEBHOOK_WORKER_ENABLED`

**Estimated effort**: 1-2 days for Phase 2 (multi-instance support)
