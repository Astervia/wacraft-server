# Webhook System Implementation Summary

This document describes the enhanced webhook system with reliability features, security improvements, and event filtering capabilities.

## API Changes

### Create Webhook - `POST /webhook`

#### New Request Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `signing_enabled` | boolean | No | `false` | Enable HMAC-SHA256 request signing |
| `max_retries` | integer | No | `3` | Max retry attempts (0-10) |
| `retry_delay_ms` | integer | No | `1000` | Base retry delay in milliseconds (100-60000) |
| `custom_headers` | object | No | `null` | Custom headers to send with requests |
| `event_filter` | object | No | `null` | Filter to match specific events |

#### New Response Fields

When `signing_enabled: true`, the response includes:

| Field | Type | Description |
|-------|------|-------------|
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

| Field | Type | Description |
|-------|------|-------------|
| `max_retries` | integer | Max retry attempts (0-10) |
| `retry_delay_ms` | integer | Base retry delay in milliseconds (100-60000) |
| `is_active` | boolean | Enable/disable webhook |
| `custom_headers` | object | Custom headers to send with requests |
| `event_filter` | object | Filter to match specific events |

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

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `webhook_id` | uuid | Yes | The webhook to test |
| `payload` | object | No | Custom test payload (optional) |

#### Response

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Whether the request succeeded (2xx status) |
| `status_code` | integer | HTTP response status code |
| `response` | object | Parsed JSON response (if valid JSON) |
| `response_body` | string | Raw response body |
| `duration_ms` | integer | Request duration in milliseconds |
| `headers_sent` | object | Headers that were sent with the request |
| `error` | string | Error message if request failed |

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
  "response": {"received": true},
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

| Field | Type | Description |
|-------|------|-------------|
| `signing_enabled` | boolean | Whether signing is enabled |
| `max_retries` | integer | Max retry attempts |
| `retry_delay_ms` | integer | Base retry delay in milliseconds |
| `is_active` | boolean | Whether webhook is active |
| `custom_headers` | object | Custom headers |
| `event_filter` | object | Event filter configuration |
| `circuit_state` | string | Circuit breaker state: `closed`, `open`, `half_open` |
| `failure_count` | integer | Consecutive failure count |
| `last_failure_at` | datetime | Last failure timestamp |
| `circuit_opened_at` | datetime | When circuit was opened |

Note: `signing_secret` is never returned in GET requests.

---

## Event Filter Configuration

### Filter Structure

```typescript
interface EventFilter {
  logic?: "AND" | "OR";  // Default: "AND"
  conditions: FilterCondition[];
}

interface FilterCondition {
  path: string;           // JSON path (e.g., "data.message.type")
  operator: FilterOperator;
  value?: any;            // Not required for "exists" operator
}

type FilterOperator = "equals" | "contains" | "regex" | "exists";
```

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `equals` | Exact match | `{"path": "data.type", "operator": "equals", "value": "text"}` |
| `contains` | Substring match | `{"path": "data.body", "operator": "contains", "value": "hello"}` |
| `regex` | Regular expression | `{"path": "data.from", "operator": "regex", "value": "^\\+1"}` |
| `exists` | Field exists | `{"path": "data.media", "operator": "exists"}` |

### Examples

**Filter text messages only:**
```json
{
  "logic": "AND",
  "conditions": [
    {"path": "data.type", "operator": "equals", "value": "text"}
  ]
}
```

**Filter messages from US numbers:**
```json
{
  "logic": "AND",
  "conditions": [
    {"path": "data.from", "operator": "regex", "value": "^\\+1"}
  ]
}
```

**Filter messages with media OR from specific contact:**
```json
{
  "logic": "OR",
  "conditions": [
    {"path": "data.media", "operator": "exists"},
    {"path": "data.from", "operator": "equals", "value": "+1234567890"}
  ]
}
```

---

## Signature Verification (For Webhook Consumers)

When `signing_enabled` is true, requests include signature headers:

| Header | Description |
|--------|-------------|
| `X-Wacraft-Signature` | HMAC-SHA256 signature in format `v1={hex}` |
| `X-Wacraft-Timestamp` | Unix timestamp (seconds) |

### Verification Steps

1. Extract timestamp from `X-Wacraft-Timestamp` header
2. Construct message: `v1:{timestamp}:{raw_request_body}`
3. Compute HMAC-SHA256 of message using `signing_secret`
4. Compare with signature from `X-Wacraft-Signature` (constant-time comparison)
5. Reject if timestamp is older than 5 minutes

### Example (Node.js)

```javascript
const crypto = require('crypto');

function verifySignature(secret, timestamp, body, signature) {
  const message = `v1:${timestamp}:${body}`;
  const expected = 'v1=' + crypto
    .createHmac('sha256', secret)
    .update(message)
    .digest('hex');

  // Constant-time comparison
  return crypto.timingSafeEqual(
    Buffer.from(expected),
    Buffer.from(signature)
  );
}

function isTimestampValid(timestamp, maxAgeSeconds = 300) {
  const now = Math.floor(Date.now() / 1000);
  return (now - parseInt(timestamp)) <= maxAgeSeconds;
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

The circuit breaker protects against failing endpoints:

| State | Description | Behavior |
|-------|-------------|----------|
| `closed` | Normal operation | Requests are sent normally |
| `open` | Circuit tripped | Requests are blocked for 30 seconds |
| `half_open` | Testing recovery | One request allowed; success closes circuit, failure reopens |

### Thresholds

- **Failure threshold**: 5 consecutive failures opens the circuit
- **Recovery timeout**: 30 seconds before attempting recovery

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

| Status | Description |
|--------|-------------|
| `pending` | Awaiting first attempt |
| `attempted` | Has been tried, awaiting retry |
| `succeeded` | Successfully delivered (2xx response) |
| `dead_letter` | Max retries exhausted |

---

## Request Headers

Webhook requests include these headers:

| Header | Always Sent | Description |
|--------|-------------|-------------|
| `Content-Type` | Yes | `application/json` |
| `X-Wacraft-Delivery-ID` | Yes | Unique delivery ID |
| `X-Wacraft-Event` | Yes | Event type |
| `X-Wacraft-Attempt` | Yes | Attempt number (1, 2, 3...) |
| `Authorization` | If configured | Authorization header value |
| `X-Wacraft-Signature` | If signing enabled | HMAC-SHA256 signature |
| `X-Wacraft-Timestamp` | If signing enabled | Unix timestamp |
| Custom headers | If configured | User-defined headers |

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
