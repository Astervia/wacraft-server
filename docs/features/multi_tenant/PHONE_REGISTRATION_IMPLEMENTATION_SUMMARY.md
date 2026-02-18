# Phone Registration Implementation Summary

This document describes the phone number registration routes that enable the full WhatsApp phone registration flow. These routes operate on phone configs regardless of their `is_active` status, allowing registration before activation.

## Overview

The registration flow follows Meta's WhatsApp Cloud API phone number registration process:

1. **Request verification code** (SMS or voice call)
2. **Verify the code** received on the phone
3. **Register the phone number** with a two-step verification PIN
4. Optionally, **authenticate with PIN** or **deregister**

All routes are nested under `/workspace/{workspace_id}/phone-config/{id}/` and require the `phone_config.manage` policy.

---

## API Endpoints

### Request Verification Code - `POST /workspace/{workspace_id}/phone-config/{id}/request-code`

Sends a verification code to the phone number via SMS or voice call.

#### Request

| Field         | Type   | Required | Description                              |
| ------------- | ------ | -------- | ---------------------------------------- |
| `code_method` | string | Yes      | Delivery method: `"SMS"` or `"VOICE"`    |
| `language`    | string | Yes      | Two-character language code (e.g. `"en"`) |

#### Example Request

```json
{
    "code_method": "SMS",
    "language": "en"
}
```

#### Example Response

```json
{
    "success": true
}
```

---

### Verify Code - `POST /workspace/{workspace_id}/phone-config/{id}/verify-code`

Verifies the code received on the phone number.

#### Request

| Field  | Type   | Required | Description                        |
| ------ | ------ | -------- | ---------------------------------- |
| `code` | string | Yes      | The verification code received     |

#### Example Request

```json
{
    "code": "123456"
}
```

#### Example Response

```json
{
    "success": true
}
```

---

### PIN Authenticate - `POST /workspace/{workspace_id}/phone-config/{id}/pin-authenticate`

Authenticates the phone number with a two-step verification PIN. This PIN is configured in Meta's Business Manager Security Center.

#### Request

| Field | Type   | Required | Description                            |
| ----- | ------ | -------- | -------------------------------------- |
| `pin` | string | Yes      | The 6-digit two-step verification PIN  |

#### Example Request

```json
{
    "pin": "654321"
}
```

#### Example Response

```json
{
    "success": true
}
```

---

### Register Phone Number - `POST /workspace/{workspace_id}/phone-config/{id}/register`

Registers the phone number to WhatsApp Cloud API. The `messaging_product` field is automatically set to `"whatsapp"`.

#### Request

| Field                        | Type   | Required | Description                                                                 |
| ---------------------------- | ------ | -------- | --------------------------------------------------------------------------- |
| `pin`                        | string | Yes      | The 6-digit two-step verification PIN                                       |
| `data_localization_region`   | string | No       | 2-letter ISO 3166 country code for local data storage (e.g. `"BR"`, `"IN"`) |

#### Example Request

```json
{
    "pin": "654321",
    "data_localization_region": "BR"
}
```

#### Example Response

```json
{
    "success": true
}
```

---

### Deregister Phone Number - `POST /workspace/{workspace_id}/phone-config/{id}/deregister`

Deregisters the phone number from WhatsApp Cloud API. No request body required.

#### Example Response

```json
{
    "success": true
}
```

---

## Authentication & Authorization

All endpoints require:

| Requirement              | Header / Mechanism      | Description                              |
| ------------------------ | ----------------------- | ---------------------------------------- |
| JWT Authentication       | `Authorization: Bearer` | Valid user JWT token                      |
| Email Verification       | -                       | User must have a verified email          |
| Workspace Membership     | `X-Workspace-ID`        | User must be a member of the workspace   |
| Policy                   | -                       | `phone_config.manage` policy required    |

---

## Typical Registration Flow

```
┌──────────────────────────────────────────────────────┐
│ 1. Create phone config (POST /phone-config)          │
│    - is_active can be false                          │
│    - Stores waba_id, access_token, etc.              │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│ 2. Request code (POST /phone-config/{id}/request-code│
│    - Sends SMS or voice call to the phone number     │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│ 3. Verify code (POST /phone-config/{id}/verify-code) │
│    - Confirms ownership of the phone number          │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│ 4. Register (POST /phone-config/{id}/register)       │
│    - Registers number with WhatsApp Cloud API        │
│    - Requires two-step verification PIN              │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│ 5. Activate (PATCH /phone-config/{id})               │
│    - Set is_active: true                             │
│    - Phone config is now ready for messaging         │
└──────────────────────────────────────────────────────┘
```

---

## Error Responses

All endpoints return errors in the standard format:

```json
{
    "message": "Failed to request verification code",
    "error": "Error details from WhatsApp API",
    "source": "whatsapp"
}
```

Common error sources:

| Source      | Description                                      |
| ----------- | ------------------------------------------------ |
| `handler`   | Invalid request parameters (e.g. bad UUID)       |
| `database`  | Phone config not found or DB error               |
| `service`   | Failed to build WhatsApp API client              |
| `whatsapp`  | Error returned by Meta's WhatsApp Cloud API      |

---

## Implementation Details

### Files

| File                                          | Description                            |
| --------------------------------------------- | -------------------------------------- |
| `src/phone-config/handler/verification.go`    | RequestCode and VerifyCode handlers    |
| `src/phone-config/handler/pin-authenticate.go`| PinAuthenticate handler                |
| `src/phone-config/handler/register.go`        | Register and DeRegister handlers       |
| `src/phone-config/router/main.go`             | Route registration (updated)           |

### Key Design Decision: No `is_active` Filter

Phone config lookups in these handlers query by `id` and `workspace_id` only, without filtering on `is_active`. This is intentional -- the registration flow must complete before a phone config can be activated.

### SDK Usage

All handlers use the `github.com/Rfluid/whatsapp-cloud-api/src/phone` package, which wraps Meta's Graph API endpoints:

| Handler            | SDK Function              | Meta API Endpoint                              |
| ------------------ | ------------------------- | ---------------------------------------------- |
| `RequestCode`      | `phone.RequestCode()`     | `POST /{phoneNumberId}/request_code`           |
| `VerifyCode`       | `phone.VerifyCode()`      | `POST /{phoneNumberId}/verify_code`            |
| `PinAuthenticate`  | `phone.AuthenticateWithPin()` | `POST /{phoneNumberId}`                    |
| `Register`         | `phone.Register()`        | `POST /{phoneNumberId}/register`               |
| `DeRegister`       | `phone.DeRegister()`      | `POST /{phoneNumberId}/deregister`             |

---

## Backwards Compatibility

- No database schema changes required
- No changes to existing phone config CRUD endpoints
- Routes are additive only -- appended to the existing phone config router
