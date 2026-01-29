# Multi-Tenant Implementation Summary

This document summarizes the implementation of workspace-based multi-tenancy for wacraft-server, completed across four phases.

## Overview

The wacraft-server has been transformed from a single-tenant application to a full multi-tenant SaaS platform with:

- **Workspace-based isolation** - All resources (contacts, messages, campaigns) are scoped to workspaces
- **Policy-based access control** - Fine-grained permissions per workspace member
- **Multi-phone support** - Each workspace can have multiple WhatsApp phone configurations with ownership validation
- **Self-service registration** - Users can register and create their own workspaces
- **Real-time workspace isolation** - WebSocket connections are scoped to workspaces for secure real-time updates

## Implementation Phases

1. **Phase 1**: Core Multi-Tenancy Foundation - Workspace entities, policies, and data model updates
2. **Phase 2**: Phone Configuration Management - Multi-phone support with encryption and ownership validation
3. **Phase 3**: Open Registration & User Self-Service - Registration, email verification, and invitations
4. **Phase 4**: Workspace-Scoped WebSockets - Real-time message and status broadcasting with workspace isolation

---

## Phase 1: Core Multi-Tenancy Foundation

### Entities Created

#### wacraft-core

| Entity                  | File                                              | Purpose                                       |
| ----------------------- | ------------------------------------------------- | --------------------------------------------- |
| `Workspace`             | `src/workspace/entity/workspace.go`               | Tenant container with name, slug, description |
| `WorkspaceMember`       | `src/workspace/entity/workspace-member.go`        | Links users to workspaces                     |
| `WorkspaceMemberPolicy` | `src/workspace/entity/workspace-member-policy.go` | Stores policies per member                    |

#### Policy System

```go
// Available policies (src/workspace/model/policy.go)
PolicyWorkspaceAdmin    = "workspace.admin"
PolicyWorkspaceSettings = "workspace.settings"
PolicyWorkspaceMembers  = "workspace.members"
PolicyPhoneConfigRead   = "phone_config.read"
PolicyPhoneConfigManage = "phone_config.manage"
PolicyContactRead       = "contact.read"
PolicyContactManage     = "contact.manage"
PolicyMessageRead       = "message.read"
PolicyMessageSend       = "message.send"
PolicyCampaignRead      = "campaign.read"
PolicyCampaignManage    = "campaign.manage"
PolicyCampaignRun       = "campaign.run"
PolicyWebhookRead       = "webhook.read"
PolicyWebhookManage     = "webhook.manage"
```

### Entities Modified

Added `WorkspaceID *uuid.UUID` to:

- `Contact` - Contacts belong to workspaces
- `MessagingProduct` - Messaging products belong to workspaces
- `Campaign` - Campaigns belong to workspaces
- `Webhook` - Webhooks belong to workspaces

### Middleware

| File                                    | Purpose                                                |
| --------------------------------------- | ------------------------------------------------------ |
| `src/workspace/middleware/workspace.go` | Extracts `X-Workspace-ID` header, validates membership |
| `src/workspace/middleware/policy.go`    | `RequirePolicy()` - Checks user has required policies  |

### API Endpoints

```
POST   /workspace                      Create workspace
GET    /workspace                      List user's workspaces
GET    /workspace/:id                  Get workspace details
PATCH  /workspace/:id                  Update workspace (requires workspace.settings)
DELETE /workspace/:id                  Delete workspace (requires workspace.admin)

POST   /workspace/:id/member           Add member (requires workspace.members)
GET    /workspace/:id/member           List members
PATCH  /workspace/:id/member/:user_id  Update member policies
DELETE /workspace/:id/member/:user_id  Remove member
```

### Header Requirement

All resource endpoints now require:

```
X-Workspace-ID: <workspace-uuid>
```

---

## Phase 2: Phone Configuration Management

### Entities Created

#### wacraft-core

| Entity        | File                                      | Purpose                                                 |
| ------------- | ----------------------------------------- | ------------------------------------------------------- |
| `PhoneConfig` | `src/phone-config/entity/phone-config.go` | WhatsApp phone configuration with encrypted credentials |

```go
type PhoneConfig struct {
    Name               string     // Friendly name
    WorkspaceID        *uuid.UUID // Owner workspace
    WabaID             string     // Phone Number ID from Meta (unique when active)
    WabaAccountID      string     // WhatsApp Business Account ID
    DisplayPhone       string     // Display phone number
    AccessToken        string     // Encrypted
    MetaAppSecret      string     // Encrypted
    WebhookVerifyToken string     // Encrypted
    IsActive           bool
}
```

**Note**: `WabaID` is the Phone Number ID from Meta's API. The redundant `PhoneNumberID` column was removed.

### Services Created

| File                                   | Purpose                                       |
| -------------------------------------- | --------------------------------------------- |
| `src/crypto/service/encryption.go`     | AES-256-GCM encryption for sensitive fields   |
| `src/phone-config/service/whatsapp.go` | WhatsApp API instance management with caching |

### Key Features

1. **Encrypted Storage** - Access tokens, app secrets, and verify tokens are encrypted at rest using AES-256-GCM

2. **API Instance Caching** - WhatsApp API instances are cached per phone config and invalidated on update/delete

3. **Multi-Phone Webhooks** - Each phone config gets its own webhook endpoint:
    - Legacy: `/webhook-in` (uses env vars)
    - Per-phone: `/webhook-in/{waba_id}`

4. **Workspace-Aware Messaging** - Message sending automatically uses the correct phone config for the workspace

5. **Ownership Validation** - Security feature to prevent phone number hijacking:
    - Validates credentials by calling WhatsApp's `GetProfile` API before storing
    - Ensures users can only register phone numbers they actually own
    - Automatically deactivates conflicting configs when validation succeeds
    - Partial unique index on `waba_id` (unique only when `is_active = true`) allows workspace transfers

### Security Features

**Problem**: A malicious user could register any `WabaID` without proving ownership, blocking the legitimate owner from using their own phone number.

**Solution**:

1. Before creating a phone config, validate ownership by calling WhatsApp's profile API
2. If validation succeeds, deactivate any other active configs with the same `WabaID`
3. Only one active config per `WabaID` across all workspaces (enforced by partial unique index)
4. Phone numbers can be transferred between workspaces by deactivating old config and creating new one

**Implementation**:

```go
func validatePhoneConfigOwnership(wabaID string, accessToken string) error {
    // Create WhatsApp API client with provided credentials
    // Call GetProfile() to verify ownership
    // Returns error if validation fails
}

func deactivateConflictingPhoneConfigs(wabaID string) error {
    // Deactivates all active phone configs with same WabaID
}
```

### API Endpoints

```
POST   /workspace/:id/phone-config      Create phone config (requires phone_config.manage)
GET    /workspace/:id/phone-config      List phone configs (requires phone_config.read)
GET    /workspace/:id/phone-config/:id  Get phone config
PATCH  /workspace/:id/phone-config/:id  Update phone config
DELETE /workspace/:id/phone-config/:id  Delete phone config
```

### Environment Variables

```bash
ENCRYPTION_KEY=<32-byte-base64-key>  # Required for encrypting phone config secrets
```

---

## Phase 3: Open Registration & User Self-Service

### Entities Created

#### wacraft-core

| Entity                | File                                           | Purpose                                |
| --------------------- | ---------------------------------------------- | -------------------------------------- |
| `EmailVerification`   | `src/user/entity/email-verification.go`        | Email verification tokens (24h expiry) |
| `PasswordResetToken`  | `src/user/entity/password-reset-token.go`      | Password reset tokens (1h expiry)      |
| `WorkspaceInvitation` | `src/workspace/entity/workspace-invitation.go` | Workspace invitations (7d expiry)      |

### Services Created

| File                             | Purpose                                 |
| -------------------------------- | --------------------------------------- |
| `src/email/service/interface.go` | Email service interface                 |
| `src/email/service/smtp.go`      | SMTP implementation with HTML templates |
| `src/crypto/service/token.go`    | Secure token generation                 |

### Key Features

1. **Self-Registration**
    - Users can register without admin invitation
    - Automatic personal workspace creation
    - All admin policies granted on personal workspace

2. **Email Verification**
    - Optional (configurable via `REQUIRE_EMAIL_VERIFICATION`)
    - 24-hour token expiry
    - HTML email templates

3. **Password Reset**
    - Self-service password reset flow
    - 1-hour token expiry
    - Tokens invalidated after use

4. **Workspace Invitations**
    - Invite users by email with specific policies
    - 7-day invitation expiry
    - Works for both existing and new users

5. **Rate Limiting**
    - Registration: 5/hour per IP
    - Login: 10/15min per IP+email
    - Password reset: 5/hour per IP

### API Endpoints

```
# Public (no auth)
POST /auth/register              Register new user
GET  /auth/verify-email          Verify email with token
POST /auth/resend-verification   Resend verification email
POST /auth/forgot-password       Request password reset
POST /auth/reset-password        Reset password with token
POST /auth/accept-invitation     Accept workspace invitation

# Authenticated
POST   /workspace/:id/invitation              Create invitation
GET    /workspace/:id/invitation              List pending invitations
DELETE /workspace/:id/invitation/:id          Revoke invitation
```

### Environment Variables

```bash
# Email Configuration
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user
SMTP_PASSWORD=password
SMTP_FROM=noreply@example.com
APP_BASE_URL=https://app.example.com

# Registration Settings
ALLOW_REGISTRATION=true           # Enable/disable open registration
REQUIRE_EMAIL_VERIFICATION=true   # Require email verification
RATE_LIMIT_REGISTRATION=5         # Per hour per IP
RATE_LIMIT_LOGIN=10               # Per 15 min per IP+email
```

---

## Database Schema

```
┌─────────────┐       ┌───────────────────┐       ┌─────────────────────────┐
│    users    │       │    workspaces     │       │   workspace_members     │
├─────────────┤       ├───────────────────┤       ├─────────────────────────┤
│ id (PK)     │◄──────│ created_by (FK)   │       │ workspace_id (FK)  ─────┼──┐
│ name        │       │ name              │       │ user_id (FK)       ─────┼──┤
│ email       │       │ slug (unique)     │       └─────────────────────────┘  │
│ password    │       │ description       │                    │               │
│ email_verified      └───────────────────┘                    ▼               │
└─────────────┘                │                ┌─────────────────────────────┐│
      │                        │                │ workspace_member_policies   ││
      │                        │                ├─────────────────────────────┤│
      ▼                        │                │ workspace_member_id (FK) ───┼┘
┌─────────────────────┐        │                │ policy                      │
│ email_verifications │        │                └─────────────────────────────┘
├─────────────────────┤        │
│ user_id (FK)        │        │                ┌─────────────────────────────┐
│ token (unique)      │        │                │   workspace_invitations     │
│ expires_at          │        │                ├─────────────────────────────┤
│ verified            │        ├───────────────►│ workspace_id (FK)           │
└─────────────────────┘        │                │ email                       │
                               │                │ token (unique)              │
┌─────────────────────┐        │                │ policies (jsonb)            │
│ password_reset_tokens│       │                │ expires_at                  │
├─────────────────────┤        │                │ accepted_at                 │
│ user_id (FK)        │        │                │ invited_by (FK)             │
│ token (unique)      │        │                └─────────────────────────────┘
│ expires_at          │        │
│ used_at             │        ▼
└─────────────────────┘ ┌───────────────────────────┐
                        │      phone_configs        │
                        ├───────────────────────────┤
                        │ workspace_id (FK)         │
                        │ name                      │
                        │ waba_id                   │
                        │ phone_number_id (unique)  │
                        │ access_token (encrypted)  │
                        │ meta_app_secret (encrypted)│
                        │ is_active                 │
                        └───────────────────────────┘
                                    │
                                    ▼
                        ┌───────────────────────────┐
                        │   messaging_products      │
                        ├───────────────────────────┤
                        │ workspace_id (FK)         │
                        │ phone_config_id (FK)      │
                        │ name                      │
                        └───────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────┐          ┌───────────────┐          ┌───────────────┐
│   contacts    │          │   campaigns   │          │   webhooks    │
├───────────────┤          ├───────────────┤          ├───────────────┤
│ workspace_id  │          │ workspace_id  │          │ workspace_id  │
│ name          │          │ name          │          │ url           │
│ email         │          │ ...           │          │ event         │
└───────────────┘          └───────────────┘          └───────────────┘
```

---

## Phase 4: Workspace-Scoped WebSockets

**Goal**: Implement workspace isolation for real-time WebSocket connections to ensure messages and statuses are only broadcast to clients in the relevant workspace.

### 4.1 Workspace Channel Manager

**File**: `src/websocket/workspace-manager/main.go`

```go
type WorkspaceChannelManager[T any] struct {
    channels       map[uuid.UUID]*websocket_model.Channel[...]
    channelSwapper *synch_service.MutexSwapper[uuid.UUID]
    globalMutex    sync.RWMutex
}
```

**Key Features**:

- Uses mutex-swapper pattern for thread-safe per-workspace channel management
- Each workspace gets its own dedicated WebSocket channel
- Automatic channel creation on first client connection
- Safe concurrent access to workspace channels

**Methods**:

- `GetOrCreateChannel(workspaceID)` - Thread-safe channel retrieval/creation
- `BroadcastToWorkspace(workspaceID, data)` - Broadcasts only to workspace clients
- `AppendClient(workspaceID, client, key)` - Registers client to workspace channel
- `RemoveClient(workspaceID, key)` - Unregisters client from workspace channel

### 4.2 WebSocket Workspace Middleware

**File**: `src/workspace/middleware/websocket.go`

```go
func WebSocketWorkspaceMiddleware(c *fiber.Ctx) error {
    // Extract workspace ID from X-Workspace-ID header or workspace_id query param
    // Validate user membership in the workspace
    // Store workspace in context
}
```

**Features**:

- Supports both header and query parameter for workspace ID (WebSocket client compatibility)
- Validates workspace existence
- Validates user membership before establishing connection
- Applied globally to `/websocket/*` routes via `src/websocket/router/main.go`

### 4.3 Updated WebSocket Handlers

#### Message WebSocket Handler

**File**: `src/message/handler/new.go`

**Changes**:

- Replaced global `NewMessageChannel` with `NewMessageWorkspaceManager`
- Clients register to workspace-specific channel
- Cleanup removes client from workspace channel only

```go
var NewMessageWorkspaceManager = websocket_workspace_manager.CreateWorkspaceChannelManager[message_entity.Message]()

func NewMessageSubscription(ctx *websocket.Conn) {
    user := ctx.Locals("user").(*user_entity.User)
    workspace := ctx.Locals("workspace").(*workspace_entity.Workspace)

    clientID := newMessageClientPool.CreateID(user.ID)
    client := websocket_model.Client[websocket_model.ClientID]{
        Connection: ctx,
        Data:       *clientID,
    }
    NewMessageWorkspaceManager.AppendClient(workspace.ID, client, clientID.String())
    // ...
}
```

#### Status WebSocket Handler

**File**: `src/status/handler/new.go`

**Changes**:

- Replaced global `NewStatusChannel` with `NewStatusWorkspaceManager`
- Clients register to workspace-specific channel
- Cleanup removes client from workspace channel only

```go
var NewStatusWorkspaceManager = websocket_workspace_manager.CreateWorkspaceChannelManager[status_entity.Status]()
```

### 4.4 Workspace-Scoped Broadcasting

All broadcasting now uses workspace managers to ensure isolation:

#### Webhook Handler

**File**: `src/webhook-in/handler/whatsapp-message.go`

```go
// Incoming messages from WhatsApp
if mp.WorkspaceID != nil {
    go message_handler.NewMessageWorkspaceManager.BroadcastToWorkspace(*mp.WorkspaceID, msg)
}

// Status updates
if mp.WorkspaceID != nil {
    go status_handler.NewStatusWorkspaceManager.BroadcastToWorkspace(*mp.WorkspaceID, status)
}
```

#### Message Sending

**File**: `src/message/handler/whatsapp.go`

```go
propagateCallback := func(data message_entity.Message) {
    go NewMessageWorkspaceManager.BroadcastToWorkspace(workspace.ID, data)
    // ...
}
```

#### Campaign Messages

**File**: `src/campaign/service/send-whatsapp-campaign.go`

```go
broadcastCallback := func(msg message_entity.Message) {
    if campaign.WorkspaceID != nil {
        go message_handler.NewMessageWorkspaceManager.BroadcastToWorkspace(*campaign.WorkspaceID, msg)
    }
    // ...
}
```

### 4.5 WebSocket Connection Examples

#### JavaScript Client (with headers support)

```javascript
const ws = new WebSocket("ws://localhost/websocket/message/new", {
    headers: {
        Authorization: "Bearer <token>",
        "X-Workspace-ID": "<workspace-uuid>",
    },
});
```

#### JavaScript Client (query param fallback)

```javascript
const ws = new WebSocket(
    "ws://localhost/websocket/message/new?workspace_id=<workspace-uuid>&Authorization=Bearer%20<token>",
);
```

### 4.6 Security Features

- **Membership Validation**: Users can only connect to workspaces they're members of
- **Data Isolation**: Messages and statuses are only broadcast to clients in the owning workspace
- **No Cross-Workspace Leakage**: Each workspace has independent client pools
- **Automatic Cleanup**: Clients are removed from workspace channels on disconnect

### 4.7 Architecture Benefits

- **Scalable**: Uses mutex-swapper pattern for efficient concurrent access
- **Thread-Safe**: All workspace channel operations are synchronized
- **No Core Changes**: Core websocket models (`ClientID`, `Channel`, `ClientPool`) remain unchanged
- **Flexible Auth**: Supports both headers and query params for WebSocket clients that don't support custom headers

### 4.8 Tasks Checklist - Phase 4

- [x] Create WorkspaceChannelManager using mutex-swapper pattern
- [x] Implement WebSocketWorkspaceMiddleware with header/query param support
- [x] Update message WebSocket handler to use workspace manager
- [x] Update status WebSocket handler to use workspace manager
- [x] Update webhook handler for workspace-scoped broadcasting
- [x] Update message sending for workspace-scoped broadcasting
- [x] Update campaign service for workspace-scoped broadcasting
- [x] Apply WebSocketWorkspaceMiddleware to websocket router
- [x] Test WebSocket connections with workspace isolation
- [x] Verify no cross-workspace message leakage

---

## Migration Notes

### For Existing Deployments

1. **Database Migration** - The `migrations-before/` directory contains migrations that:
    - Detect if tables already exist (existing deployment)
    - Create default workspace for the super user
    - Migrate all existing data to the default workspace
    - Skip gracefully on virgin databases

2. **Environment Variables** - New required variable:

    ```bash
    ENCRYPTION_KEY=<generate-with-openssl>
    ```

    Generate with: `openssl rand -base64 32`

3. **Webhook URLs** - If using phone configs, update Meta webhook URLs to:

    ```
    https://your-domain.com/webhook-in/<waba_id>
    ```

    **Note**: `PhoneNumberID` column was removed as it was a duplicate of `WabaID`. The migration automatically drops this column.

4. **WebSocket Connections** - WebSocket clients now require workspace ID:

    ```javascript
    // Using header
    const ws = new WebSocket("ws://domain/websocket/message/new", {
        headers: { "X-Workspace-ID": "<workspace-uuid>" },
    });

    // Using query param (for clients without header support)
    const ws = new WebSocket("ws://domain/websocket/message/new?workspace_id=<workspace-uuid>");
    ```

### Backwards Compatibility

- Legacy environment variables (`WABA_ID`, `WABA_ACCESS_TOKEN`, etc.) still work as fallback
- Legacy webhook endpoint `/webhook-in` preserved for transition period
- All existing APIs work with `X-Workspace-ID` header added
- Phone config URLs changed from `/webhook-in/<phone_number_id>` to `/webhook-in/<waba_id>` (same value, different field name)

---

## File Summary

### New Files Created

**wacraft-core (14 files):**

```
src/workspace/entity/workspace.go
src/workspace/entity/workspace-member.go
src/workspace/entity/workspace-member-policy.go
src/workspace/entity/workspace-invitation.go
src/workspace/model/policy.go
src/workspace/model/create.go
src/workspace/model/update.go
src/workspace/model/query-paginated.go
src/workspace/model/create-member.go
src/workspace/model/invitation.go
src/user/entity/email-verification.go
src/user/entity/password-reset-token.go
src/user/model/register.go
src/phone-config/entity/phone-config.go
src/phone-config/model/create.go
src/phone-config/model/query-paginated.go
src/crypto/service/encryption.go
src/crypto/service/token.go
```

**wacraft-server (21 files):**

```
src/workspace/middleware/workspace.go
src/workspace/middleware/policy.go
src/workspace/middleware/websocket.go
src/workspace/handler/workspace.go
src/workspace/handler/member.go
src/workspace/handler/invitation.go
src/workspace/router/main.go
src/websocket/workspace-manager/main.go
src/phone-config/handler/main.go
src/phone-config/router/main.go
src/phone-config/service/whatsapp.go
src/auth/handler/register.go
src/auth/handler/verify-email.go
src/auth/handler/password-reset.go
src/auth/middleware/rate-limit.go
src/auth/router/main.go
src/email/service/interface.go
src/email/service/smtp.go
src/config/env/encryption.go
src/config/env/email.go
src/config/env/registration.go
src/database/migrations-before/20260128000001_add_workspace_to_existing_tables.go
src/database/migrations/20260128120000_drop_phone_number_id_column.go
```

### Modified Files

**wacraft-core:**

- `src/contact/entity/contact.go` - Added WorkspaceID
- `src/campaign/entity/campaign.go` - Added WorkspaceID
- `src/webhook/entity/webhook.go` - Added WorkspaceID
- `src/messaging-product/entity/messaging-product.go` - Added WorkspaceID, PhoneConfigID
- `src/user/entity/user.go` - Added EmailVerified
- `src/phone-config/entity/phone-config.go` - Removed PhoneNumberID column (duplicate of WabaID)

**wacraft-server:**

- All routers - Added WorkspaceMiddleware and RequirePolicy
- All handlers - Scoped queries by workspace
- `src/server/serve.go` - Added auth_router, workspace_router
- `src/database/migrate/migrate.go` - Added new entities to AutoMigrate
- `src/config/env/main.go` - Added new env loaders
- `src/webhook-in/config/whatsapp.go` - Multi-phone webhook routing (using WabaID)
- `src/message/service/whatsapp.go` - Workspace-aware messaging
- `src/message/handler/new.go` - Workspace-scoped WebSocket handler
- `src/status/handler/new.go` - Workspace-scoped WebSocket handler
- `src/webhook-in/handler/whatsapp-message.go` - Workspace-scoped broadcasting
- `src/message/handler/whatsapp.go` - Workspace-scoped message broadcasting
- `src/campaign/service/send-whatsapp-campaign.go` - Workspace-scoped campaign broadcasting
- `src/websocket/router/main.go` - Added WebSocketWorkspaceMiddleware
- `src/phone-config/service/whatsapp.go` - Renamed GetWhatsAppAPIByPhoneNumberID to GetWhatsAppAPIByWabaID

---

## Testing Checklist

### Core Multi-Tenancy

- [ ] Create workspace via API
- [ ] List user's workspaces
- [ ] Add member to workspace with policies
- [ ] Verify cross-workspace isolation (user can't see other workspace data)

### Phone Configuration

- [ ] Create phone config with encrypted credentials
- [ ] Verify phone config ownership validation (reject invalid credentials)
- [ ] Test WabaID uniqueness constraint (only one active config per WabaID)
- [ ] Send message using phone config
- [ ] Receive webhook on per-phone endpoint (`/webhook-in/{waba_id}`)

### User Registration & Authentication

- [ ] Register new user
- [ ] Verify email
- [ ] Reset password
- [ ] Send workspace invitation
- [ ] Accept invitation (new user)
- [ ] Accept invitation (existing user)
- [ ] Rate limiting on auth endpoints

### WebSocket Workspace Isolation

- [ ] Connect to message WebSocket with workspace ID (header)
- [ ] Connect to message WebSocket with workspace ID (query param)
- [ ] Connect to status WebSocket with workspace ID
- [ ] Verify user cannot connect to workspace they're not a member of
- [ ] Send message and verify WebSocket broadcast only to workspace members
- [ ] Receive WhatsApp message and verify broadcast only to workspace members
- [ ] Run campaign and verify broadcasts only to campaign workspace
- [ ] Connect two users to different workspaces and verify no cross-workspace leakage
- [ ] Disconnect WebSocket and verify proper cleanup from workspace channel

---

## Comprehensive Test Scenarios

This section provides detailed, step-by-step test scenarios to validate the complete multi-tenant implementation.

### Test Setup

**Preconditions:**

- Fresh database or use existing deployment with migration applied
- Server running with all environment variables configured:
    - `ENCRYPTION_KEY` set (generate with `openssl rand -base64 32`)
    - SMTP configured for email verification
    - `ALLOW_REGISTRATION=true`
    - `REQUIRE_EMAIL_VERIFICATION=true` (optional, can be false for faster testing)

**Test Users:**

- User A: `alice@example.com`
- User B: `bob@example.com`
- User C: `charlie@example.com`

**Test Workspaces:**

- Workspace 1: "Alice's Company" (owned by User A)
- Workspace 2: "Bob's Business" (owned by User B)

---

### 1. Workspace Management & Access Control

#### Test 1.1: Create Workspace and Auto-Admin Assignment

**Steps:**

1. Register User A (`alice@example.com`)
2. Verify email (if enabled)
3. Login as User A
4. List workspaces: `GET /workspace`
5. Verify personal workspace was auto-created: "Alice's Workspace"
6. Get workspace details: `GET /workspace/{workspace_id}`
7. Verify User A has all admin policies

**Expected Results:**

- User A has exactly 1 workspace
- Workspace name matches user's name
- User A has all policies from `AdminPolicies` array
- Workspace `created_by` = User A's ID

#### Test 1.2: Workspace Member Management

**Steps:**

1. As User A, invite User B to Workspace 1 with `MemberPolicies`
2. `POST /workspace/{ws1_id}/invitation`
    ```json
    {
        "email": "bob@example.com",
        "policies": ["contact.read", "contact.manage", "message.read", "message.send"]
    }
    ```
3. User B accepts invitation: `POST /auth/accept-invitation?token={token}`
4. As User A, list workspace members: `GET /workspace/{ws1_id}/member`
5. As User A, update User B's policies to `ViewerPolicies`: `PATCH /workspace/{ws1_id}/member/{user_b_id}`
6. As User A, remove User B: `DELETE /workspace/{ws1_id}/member/{user_b_id}`

**Expected Results:**

- User B receives invitation email
- After acceptance, User B appears in workspace member list
- Policy update reflected immediately
- After removal, User B cannot access Workspace 1 resources
- User B's personal workspace remains intact

#### Test 1.3: Policy-Based Access Control

**Steps:**

1. Add User B to Workspace 1 with `PolicyContactRead` only
2. As User B, try to create contact: `POST /contact` with `X-Workspace-ID: {ws1_id}`
3. As User B, try to list contacts: `GET /contact` with `X-Workspace-ID: {ws1_id}`
4. As User A, update User B policies to include `PolicyContactManage`
5. As User B, retry creating contact

**Expected Results:**

- Step 2: Returns `403 Forbidden` (missing `contact.manage`)
- Step 3: Returns `200 OK` with contact list (has `contact.read`)
- Step 5: Returns `201 Created` (now has `contact.manage`)

---

### 2. Multi-Tenant Data Isolation

#### Test 2.1: Contact Isolation

**Steps:**

1. As User A, create contact in Workspace 1:
    ```json
    POST /contact
    X-Workspace-ID: {ws1_id}
    { "name": "Contact A1", "email": "a1@test.com" }
    ```
2. As User B, create contact in Workspace 2:
    ```json
    POST /contact
    X-Workspace-ID: {ws2_id}
    { "name": "Contact B1", "email": "b1@test.com" }
    ```
3. As User A, list contacts: `GET /contact?X-Workspace-ID={ws1_id}`
4. As User B, list contacts: `GET /contact?X-Workspace-ID={ws2_id}`
5. As User A, try to access Workspace 2 contacts: `GET /contact?X-Workspace-ID={ws2_id}`
6. As User B, try to get Contact A1 by ID: `GET /contact/{contact_a1_id}?X-Workspace-ID={ws2_id}`

**Expected Results:**

- Step 3: User A sees only Contact A1
- Step 4: User B sees only Contact B1
- Step 5: User A gets `403 Forbidden` (not a member of Workspace 2)
- Step 6: Returns `404 Not Found` (contact doesn't exist in Workspace 2)

#### Test 2.2: Message Isolation

**Steps:**

1. Create phone config for Workspace 1
2. Create phone config for Workspace 2
3. Send message in Workspace 1
4. Send message in Workspace 2
5. As User A, list messages: `GET /message?X-Workspace-ID={ws1_id}`
6. As User B, list messages: `GET /message?X-Workspace-ID={ws2_id}`
7. As User A, try to access message from Workspace 2 by ID

**Expected Results:**

- Users only see messages from their own workspace
- Cross-workspace message access returns `404 Not Found`

#### Test 2.3: Campaign Isolation

**Steps:**

1. As User A, create campaign in Workspace 1
2. As User B, create campaign in Workspace 2
3. As User A, list campaigns: `GET /campaign?X-Workspace-ID={ws1_id}`
4. As User B, list campaigns: `GET /campaign?X-Workspace-ID={ws2_id}`
5. As User A, try to run User B's campaign: `POST /campaign/{campaign_b_id}/send?X-Workspace-ID={ws1_id}`

**Expected Results:**

- Each user sees only their workspace's campaigns
- Cross-workspace campaign execution returns `404 Not Found`

---

### 3. Phone Configuration & Security

#### Test 3.1: Phone Config Ownership Validation

**Steps:**

1. As User A, try to create phone config with invalid credentials:
    ```json
    POST /workspace/{ws1_id}/phone-config
    {
      "name": "Test Phone",
      "waba_id": "123456789",
      "waba_account_id": "987654321",
      "access_token": "invalid_token",
      "meta_app_secret": "fake_secret",
      "webhook_verify_token": "test_token",
      "display_phone": "+1234567890"
    }
    ```
2. Verify error message about ownership validation failure
3. Create phone config with **valid** WhatsApp credentials
4. Verify phone config is created with `is_active = true`

**Expected Results:**

- Step 1: Returns `400 Bad Request` with message "Failed to verify phone number ownership"
- Step 3: Returns `201 Created`
- GetProfile API was called to validate ownership

#### Test 3.2: WabaID Uniqueness Constraint

**Steps:**

1. As User A, create phone config with WabaID "111111111" in Workspace 1
2. As User B, try to create phone config with same WabaID "111111111" in Workspace 2
3. Verify User B gets error about duplicate active config
4. As User A, deactivate phone config: `PATCH /workspace/{ws1_id}/phone-config/{config_id}` with `is_active: false`
5. As User B, retry creating phone config with WabaID "111111111"
6. Verify User B's config is created successfully
7. Check User A's config is still inactive

**Expected Results:**

- Step 2: Returns `400 Bad Request` or `409 Conflict` (unique constraint violation)
- Step 5: Returns `201 Created`
- Only one active config per WabaID exists at any time
- Inactive configs don't block creation

#### Test 3.3: Phone Config Transfer Between Workspaces

**Steps:**

1. As User A, create active phone config with WabaID "222222222" in Workspace 1
2. Send test message using this phone config
3. As User B, create phone config with same WabaID "222222222" and **valid credentials** in Workspace 2
4. Verify ownership validation succeeds
5. Check User A's config status: `GET /workspace/{ws1_id}/phone-config`
6. Check User B's config status: `GET /workspace/{ws2_id}/phone-config`
7. Send message from Workspace 2 using new config

**Expected Results:**

- Step 3: Returns `201 Created`
- Step 5: User A's config is now `is_active = false` (auto-deactivated)
- Step 6: User B's config is `is_active = true`
- Step 7: Message sends successfully from Workspace 2

#### Test 3.4: Webhook Routing with Multiple Phone Configs

**Steps:**

1. Create phone config A (WabaID: "AAA") in Workspace 1
2. Create phone config B (WabaID: "BBB") in Workspace 2
3. Send webhook to `/webhook-in/AAA` with test message
4. Send webhook to `/webhook-in/BBB` with test message
5. Verify messages appear in correct workspaces

**Expected Results:**

- Webhook to AAA creates message in Workspace 1 only
- Webhook to BBB creates message in Workspace 2 only
- No cross-workspace message creation

---

### 4. User Registration & Authentication

#### Test 4.1: Self-Service Registration Flow

**Steps:**

1. Register new user: `POST /auth/register`
    ```json
    {
        "name": "Charlie",
        "email": "charlie@example.com",
        "password": "SecurePass123!"
    }
    ```
2. Check email for verification link
3. Verify email: `GET /auth/verify-email?token={token}`
4. Login: `POST /auth/login`
5. List workspaces: `GET /workspace`
6. Verify personal workspace exists

**Expected Results:**

- Registration returns `201 Created`
- Verification email sent
- After verification, login succeeds
- Personal workspace auto-created with user as admin

#### Test 4.2: Email Verification Expiry

**Steps:**

1. Register new user
2. Wait 25 hours (token expires in 24 hours)
3. Try to verify email with expired token
4. Request new verification email: `POST /auth/resend-verification`
5. Verify with new token

**Expected Results:**

- Step 3: Returns `400 Bad Request` "Token expired"
- Step 4: New verification email sent
- Step 5: Verification succeeds

#### Test 4.3: Password Reset Flow

**Steps:**

1. Request password reset: `POST /auth/forgot-password` with `email: "alice@example.com"`
2. Check email for reset link
3. Reset password: `POST /auth/reset-password`
    ```json
    {
        "token": "{reset_token}",
        "password": "NewPassword456!"
    }
    ```
4. Try to login with old password
5. Login with new password

**Expected Results:**

- Reset email sent
- Step 4: Login fails with old password
- Step 5: Login succeeds with new password
- Reset token marked as used

#### Test 4.4: Rate Limiting

**Steps:**

1. Attempt to register 6 times from same IP within 1 hour
2. Attempt to login with wrong password 11 times within 15 minutes
3. Wait for rate limit window to expire
4. Retry registration/login

**Expected Results:**

- 6th registration attempt returns `429 Too Many Requests`
- 11th login attempt returns `429 Too Many Requests`
- After wait period, requests succeed

---

### 5. WebSocket Real-Time Communication

#### Test 5.1: Message WebSocket Connection with Headers

**Steps:**

1. Connect WebSocket as User A:
    ```javascript
    const ws = new WebSocket("ws://localhost/websocket/message/new", {
        headers: {
            Authorization: "Bearer {user_a_token}",
            "X-Workspace-ID": "{ws1_id}",
        },
    });
    ```
2. Verify connection established (101 Switching Protocols)
3. Send ping: `ws.send('ping')`
4. Verify pong received

**Expected Results:**

- Connection succeeds
- Pong message received
- Client registered in workspace channel

#### Test 5.2: Message WebSocket Connection with Query Params

**Steps:**

1. Connect WebSocket as User B using query params:
    ```javascript
    const token = encodeURIComponent("Bearer {user_b_token}");
    const ws = new WebSocket(`ws://localhost/websocket/message/new?workspace_id={ws2_id}&Authorization=${token}`);
    ```
2. Verify connection established

**Expected Results:**

- Connection succeeds (fallback to query param works)

#### Test 5.3: WebSocket Workspace Membership Validation

**Steps:**

1. As User A, try to connect to Workspace 2's message WebSocket:
    ```javascript
    const ws = new WebSocket("ws://localhost/websocket/message/new", {
        headers: {
            Authorization: "Bearer {user_a_token}",
            "X-Workspace-ID": "{ws2_id}", // User A is NOT a member
        },
    });
    ```
2. Verify connection rejected

**Expected Results:**

- Connection fails with `403 Forbidden`
- Error message: "You are not a member of this workspace"

#### Test 5.4: Workspace-Scoped Message Broadcasting

**Setup:**

- User A connected to Workspace 1 message WebSocket
- User B connected to Workspace 2 message WebSocket
- User C connected to Workspace 1 message WebSocket (after being invited)

**Steps:**

1. As User A, send WhatsApp message in Workspace 1
2. Check User A's WebSocket receives message
3. Check User C's WebSocket receives message
4. Check User B's WebSocket does NOT receive message
5. As User B, send WhatsApp message in Workspace 2
6. Check User B's WebSocket receives message
7. Check User A's and User C's WebSockets do NOT receive message

**Expected Results:**

- Messages broadcast only to workspace members
- No cross-workspace leakage
- All members of same workspace receive the broadcast

#### Test 5.5: Status Update Broadcasting

**Setup:**

- User A and User C connected to Workspace 1 status WebSocket
- User B connected to Workspace 2 status WebSocket

**Steps:**

1. Receive WhatsApp message status update for Workspace 1 message (delivered, read, etc.)
2. Verify User A and User C receive status update
3. Verify User B does NOT receive status update
4. Receive status update for Workspace 2 message
5. Verify only User B receives it

**Expected Results:**

- Status updates broadcast only to owning workspace
- No cross-workspace status leakage

#### Test 5.6: Campaign Message Broadcasting

**Setup:**

- User A and User C connected to Workspace 1 message WebSocket
- User B connected to Workspace 2 message WebSocket

**Steps:**

1. As User A, run campaign in Workspace 1 sending 10 messages
2. Verify User A and User C receive all 10 message broadcasts via WebSocket
3. Verify User B receives 0 message broadcasts
4. As User B, run campaign in Workspace 2 sending 5 messages
5. Verify User B receives all 5 message broadcasts
6. Verify User A and User C receive 0 broadcasts from Workspace 2 campaign

**Expected Results:**

- Campaign messages broadcast only to campaign's workspace
- Real-time updates work correctly during bulk operations

#### Test 5.7: WebSocket Cleanup on Disconnect

**Steps:**

1. Connect User A to Workspace 1 message WebSocket
2. Note the client count in workspace channel (inspect server logs or add debug endpoint)
3. Disconnect WebSocket
4. Check client count decreased
5. Send message in Workspace 1
6. Verify disconnected client does NOT receive message
7. Reconnect User A
8. Verify new messages are received

**Expected Results:**

- Client removed from workspace channel on disconnect
- No messages sent to disconnected clients
- Reconnection creates new client registration

---

### 6. Cross-Workspace Security Tests

#### Test 6.1: Workspace Header Manipulation

**Steps:**

1. As User A (member of Workspace 1 only), try to list contacts with Workspace 2 header:
    ```
    GET /contact
    X-Workspace-ID: {ws2_id}
    Authorization: Bearer {user_a_token}
    ```
2. Try to create contact in Workspace 2
3. Try to send message in Workspace 2

**Expected Results:**

- All requests return `403 Forbidden`
- User A cannot access Workspace 2 data by manipulating headers

#### Test 6.2: Resource ID Guessing Across Workspaces

**Steps:**

1. As User B, create contact in Workspace 2, note the contact ID
2. As User A, try to access that contact:
    ```
    GET /contact/{contact_b_id}
    X-Workspace-ID: {ws1_id}
    ```
3. Try to update that contact
4. Try to delete that contact

**Expected Results:**

- All requests return `404 Not Found`
- Resources filtered by workspace even if ID is known

#### Test 6.3: WebSocket Workspace Switching

**Steps:**

1. Connect User A to Workspace 1 message WebSocket
2. From same client, try to connect to Workspace 2 without proper auth
3. Connect User A to Workspace 1 message WebSocket (connection 1)
4. Invite User A to Workspace 2
5. Open new WebSocket connection to Workspace 2 (connection 2)
6. Send message in Workspace 1
7. Send message in Workspace 2
8. Verify each connection receives only its workspace messages

**Expected Results:**

- Cannot connect to unauthorized workspace
- Multiple connections to different workspaces work independently
- Each connection receives only relevant workspace data

---

### 7. Edge Cases & Error Handling

#### Test 7.1: Workspace Deletion

**Steps:**

1. Create Workspace 3 with User C as admin
2. Add User A as member
3. Create contacts, messages, campaigns in Workspace 3
4. As User C, delete workspace: `DELETE /workspace/{ws3_id}`
5. As User A, try to access Workspace 3 resources
6. As User C, verify Workspace 3 not in workspace list

**Expected Results:**

- Workspace and all related data deleted (or soft-deleted based on implementation)
- Former members cannot access workspace
- User C's personal workspace still exists

#### Test 7.2: Member Removal During Active WebSocket

**Steps:**

1. Connect User A to Workspace 1 message WebSocket
2. As Workspace 1 admin, remove User A from workspace
3. Send message in Workspace 1
4. Check if User A's WebSocket receives message
5. User A tries to send message via API

**Expected Results:**

- WebSocket connection may remain open but receives no new messages
- OR connection is closed with appropriate error
- API request returns `403 Forbidden`

#### Test 7.3: Concurrent Phone Config Creation

**Steps:**

1. Launch 2 simultaneous requests to create phone config with same WabaID
    - Request A from Workspace 1
    - Request B from Workspace 2
2. Both with valid credentials for the same phone number

**Expected Results:**

- One request succeeds (whichever validates and writes first)
- Other request fails with unique constraint error
- Database maintains consistency (only one active config)

#### Test 7.4: Missing Workspace Header

**Steps:**

1. Try to access workspace-scoped endpoint without header:
    ```
    GET /contact
    Authorization: Bearer {token}
    (no X-Workspace-ID header)
    ```

**Expected Results:**

- Returns `400 Bad Request`
- Error message: "X-Workspace-ID header is required"

#### Test 7.5: Invalid Workspace UUID

**Steps:**

1. Try to access endpoint with malformed workspace ID:
    ```
    GET /contact
    X-Workspace-ID: not-a-uuid
    ```

**Expected Results:**

- Returns `400 Bad Request`
- Error message: "Invalid workspace ID format"

#### Test 7.6: Deleted User in Workspace

**Steps:**

1. Add User C to Workspace 1
2. Delete User C account
3. List Workspace 1 members
4. Try to access resources created by User C

**Expected Results:**

- User C removed from workspace members or marked as deleted
- Resources created by User C remain accessible (or follow cascade rules)

---

### 8. Performance & Load Tests

#### Test 8.1: Multiple Concurrent WebSocket Connections

**Steps:**

1. Connect 100 clients to Workspace 1 message WebSocket
2. Send message in Workspace 1
3. Verify all 100 clients receive broadcast within acceptable time (<1 second)
4. Check server memory and CPU usage

**Expected Results:**

- All clients receive message
- No memory leaks
- Server remains responsive

#### Test 8.2: Large Workspace Member List

**Steps:**

1. Create workspace with 1000 members
2. As admin, list all members: `GET /workspace/{id}/member`
3. Update a member's policies
4. Remove a member

**Expected Results:**

- Operations complete within acceptable time
- No timeout errors
- Pagination works correctly

#### Test 8.3: Campaign with Workspace Isolation

**Steps:**

1. Run campaign in Workspace 1 sending 10,000 messages
2. Run campaign in Workspace 2 sending 10,000 messages simultaneously
3. Monitor WebSocket broadcasts to both workspaces
4. Verify message counts in each workspace

**Expected Results:**

- Both campaigns complete successfully
- Each workspace receives exactly its 10,000 messages
- No cross-workspace contamination
- WebSocket broadcasts maintain isolation under load

---

## Test Execution Guide

### Running the Tests

1. **Sequential Tests**: Tests 1-7 should be run sequentially to ensure proper isolation
2. **Parallel Tests**: Performance tests (Test 8) can be run separately
3. **Cleanup**: Between test runs, either:
    - Reset database and migrations
    - Use unique email addresses and workspace names
    - Implement test cleanup scripts

### Test Tools

- **API Testing**: Postman, Insomnia, or curl
- **WebSocket Testing**: wscat, Postman WebSocket, or custom JavaScript client
- **Load Testing**: k6, Apache JMeter, or Artillery
- **Database Inspection**: psql, pgAdmin, or DataGrip

### Success Criteria

All tests must pass for the multi-tenant implementation to be considered complete and production-ready:

- ✅ Zero cross-workspace data leakage
- ✅ All security validations working correctly
- ✅ WebSocket isolation functioning properly
- ✅ Phone config ownership validation preventing hijacking
- ✅ Policy-based access control enforced
- ✅ No unauthorized access via header manipulation or ID guessing

### Rollback Plan

If critical tests fail:

1. Do not deploy to production
2. Review failed test scenario
3. Fix identified issue
4. Re-run full test suite
5. Consider migration rollback if database changes cause issues
