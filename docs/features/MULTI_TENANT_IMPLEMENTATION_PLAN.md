# Multi-Tenant Implementation Plan

This document outlines the phased implementation strategy for converting wacraft-server from a single-tenant application to a multi-tenant SaaS platform with workspace-based isolation and policy-based access control.

## Current State

- **Authentication**: JWT-based with single super user created in migrations
- **Authorization**: Simple role-based (Admin, User, Automation, Developer)
- **Data Isolation**: None - all authenticated users see all resources
- **WhatsApp Config**: Single configuration via environment variables
- **User Registration**: Admin-only user creation

## Target State

- **Authentication**: JWT-based with open registration (email verification)
- **Authorization**: Policy-based access control scoped to workspaces
- **Data Isolation**: Full workspace-based multi-tenancy
- **WhatsApp Config**: Per-workspace phone number configurations stored in database
- **User Registration**: Self-service with workspace creation

---

## Phase 1: Core Multi-Tenancy Foundation

**Goal**: Establish workspace infrastructure and migrate existing data model.

### 1.1 New Entities (wacraft-core)

#### Workspace Entity

**File**: `wacraft-core/src/workspace/entity/workspace.go`

```go
type Workspace struct {
    Name        string    `json:"name" gorm:"not null"`
    Slug        string    `json:"slug" gorm:"not null;uniqueIndex"` // URL-friendly identifier
    Description *string   `json:"description,omitempty"`
    CreatedBy   uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`

    Creator *user_entity.User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

    common_model.Audit
}
```

#### WorkspaceMember Entity

**File**: `wacraft-core/src/workspace/entity/workspace-member.go`

```go
type WorkspaceMember struct {
    WorkspaceID uuid.UUID `json:"workspace_id" gorm:"type:uuid;not null;uniqueIndex:uq_workspace_member,priority:1"`
    UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:uq_workspace_member,priority:2"`

    Workspace *Workspace        `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
    User      *user_entity.User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

#### WorkspaceMemberPolicy Entity

**File**: `wacraft-core/src/workspace/entity/workspace-member-policy.go`

```go
type WorkspaceMemberPolicy struct {
    WorkspaceMemberID uuid.UUID              `json:"workspace_member_id" gorm:"type:uuid;not null;index"`
    Policy            workspace_model.Policy `json:"policy" gorm:"type:varchar(50);not null"`

    WorkspaceMember *WorkspaceMember `json:"workspace_member,omitempty" gorm:"foreignKey:WorkspaceMemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

#### Policy Model

**File**: `wacraft-core/src/workspace/model/policy.go`

```go
type Policy string

const (
    // Workspace-level policies
    PolicyWorkspaceAdmin    Policy = "workspace.admin"    // Full workspace control
    PolicyWorkspaceSettings Policy = "workspace.settings" // Manage workspace settings
    PolicyWorkspaceMembers  Policy = "workspace.members"  // Manage workspace members

    // Phone config policies
    PolicyPhoneConfigRead   Policy = "phone_config.read"   // View phone configurations
    PolicyPhoneConfigManage Policy = "phone_config.manage" // Create/update/delete phone configs

    // Contact policies
    PolicyContactRead   Policy = "contact.read"   // View contacts
    PolicyContactManage Policy = "contact.manage" // Create/update/delete contacts

    // Message policies
    PolicyMessageRead Policy = "message.read" // View messages
    PolicyMessageSend Policy = "message.send" // Send messages

    // Campaign policies
    PolicyCampaignRead   Policy = "campaign.read"   // View campaigns
    PolicyCampaignManage Policy = "campaign.manage" // Create/update/delete campaigns
    PolicyCampaignRun    Policy = "campaign.run"    // Execute campaigns

    // Webhook policies
    PolicyWebhookRead   Policy = "webhook.read"   // View webhooks
    PolicyWebhookManage Policy = "webhook.manage" // Create/update/delete webhooks
)

// PolicyGroups for convenience
var AdminPolicies = []Policy{
    PolicyWorkspaceAdmin,
    PolicyWorkspaceSettings,
    PolicyWorkspaceMembers,
    PolicyPhoneConfigRead,
    PolicyPhoneConfigManage,
    PolicyContactRead,
    PolicyContactManage,
    PolicyMessageRead,
    PolicyMessageSend,
    PolicyCampaignRead,
    PolicyCampaignManage,
    PolicyCampaignRun,
    PolicyWebhookRead,
    PolicyWebhookManage,
}

var MemberPolicies = []Policy{
    PolicyContactRead,
    PolicyContactManage,
    PolicyMessageRead,
    PolicyMessageSend,
}

var ViewerPolicies = []Policy{
    PolicyContactRead,
    PolicyMessageRead,
}
```

### 1.2 Add Workspace FK to Existing Entities

#### Contact Entity Update

**File**: `wacraft-core/src/contact/entity/contact.go`

```go
type Contact struct {
    Name        *string   `json:"name,omitempty"`
    Email       *string   `json:"email,omitempty"`
    PhotoPath   *string   `json:"photo_path,omitempty"`
    WorkspaceID uuid.UUID `json:"workspace_id" gorm:"type:uuid;not null;index"` // NEW

    Workspace *workspace_entity.Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // NEW

    common_model.Audit
}
```

#### MessagingProduct Entity Update

**File**: `wacraft-core/src/messaging-product/entity/messaging-product.go`

```go
type MessagingProduct struct {
    Name        messaging_product_model.MessagingProductName `json:"name,omitempty" gorm:"not null"`
    WorkspaceID uuid.UUID                                    `json:"workspace_id" gorm:"type:uuid;not null;index"` // NEW

    Workspace *workspace_entity.Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // NEW

    common_model.Audit
}
```

#### Campaign Entity Update

**File**: `wacraft-core/src/campaign/entity/campaign.go`

```go
type Campaign struct {
    Name               string     `json:"name,omitempty" gorm:"not null"` // Remove unique constraint
    MessagingProductID *uuid.UUID `json:"messaging_product_id,omitempty" gorm:"type:uuid;not null"`
    WorkspaceID        uuid.UUID  `json:"workspace_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_campaign_name_workspace,priority:1"` // NEW

    // Add composite unique: (workspace_id, name)

    MessagingProduct *messaging_product_entity.MessagingProduct `json:"messaging_product,omitempty" gorm:"foreignKey:MessagingProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
    Workspace        *workspace_entity.Workspace                `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // NEW

    common_model.Audit
}
```

#### Webhook Entity Update

**File**: `wacraft-core/src/webhook/entity/webhook.go`

```go
type Webhook struct {
    Url           string              `json:"url,omitempty" gorm:"not null"`
    Authorization string              `json:"authorization,omitempty" gorm:"default:null"`
    HttpMethod    string              `json:"http_method,omitempty" gorm:"not null"`
    Timeout       *int                `json:"timeout,omitempty" gorm:"default:1"`
    Event         webhook_model.Event `json:"event,omitempty" gorm:"not null"`
    WorkspaceID   uuid.UUID           `json:"workspace_id" gorm:"type:uuid;not null;index"` // NEW

    Workspace *workspace_entity.Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // NEW

    common_model.Audit
}
```

### 1.3 Database Migrations (wacraft-server)

**File**: `src/database/migrations/YYYYMMDDHHMMSS_add_workspace_tables.go`

```go
func init() {
    goose.AddMigrationNoTxContext(upAddWorkspaceTables, downAddWorkspaceTables)
}

func upAddWorkspaceTables(ctx context.Context, db *sql.DB) error {
    // 1. Create workspaces table
    // 2. Create workspace_members table
    // 3. Create workspace_member_policies table
    // 4. Add workspace_id column to contacts (nullable initially)
    // 5. Add workspace_id column to messaging_products (nullable initially)
    // 6. Add workspace_id column to campaigns (nullable initially)
    // 7. Add workspace_id column to webhooks (nullable initially)
    return nil
}
```

**File**: `src/database/migrations/YYYYMMDDHHMMSS_migrate_existing_data_to_default_workspace.go`

```go
func upMigrateToDefaultWorkspace(ctx context.Context, db *sql.DB) error {
    // 1. Create default workspace for super user
    // 2. Add super user as workspace admin
    // 3. Assign all existing contacts to default workspace
    // 4. Assign all existing messaging_products to default workspace
    // 5. Assign all existing campaigns to default workspace
    // 6. Assign all existing webhooks to default workspace
    // 7. Make workspace_id columns NOT NULL
    // 8. Add foreign key constraints
    return nil
}
```

### 1.4 Workspace Middleware (wacraft-server)

**File**: `src/workspace/middleware/workspace.go`

```go
// WorkspaceMiddleware extracts workspace from header/path and validates membership
func WorkspaceMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get workspace ID from X-Workspace-ID header or path parameter
        workspaceID := c.Get("X-Workspace-ID")
        if workspaceID == "" {
            workspaceID = c.Params("workspace_id")
        }

        // Validate user is member of workspace
        // Store workspace and membership in context
        // Continue to next handler
    }
}

// RequirePolicy checks if the user has specific policy in current workspace
func RequirePolicy(policies ...workspace_model.Policy) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get membership from context
        // Check if any required policy is present
        // Return 403 if not authorized
    }
}
```

### 1.5 Update Query Models (wacraft-core)

Add `WorkspaceID` field to all QueryPaginated models:

**File**: `wacraft-core/src/contact/model/query-paginated.go`

```go
type QueryPaginated struct {
    WorkspaceID *uuid.UUID `json:"workspace_id,omitempty" query:"workspace_id"` // NEW - required for filtering

    common_model.UnrequiredID
    CreateContact
    database_model.Paginate
    database_model.DateOrder
    database_model.DateWhere
}

func (q *QueryPaginated) Where(db **gorm.DB, prefix string) error {
    // Existing logic...

    // NEW: Filter by workspace
    if q.WorkspaceID != nil {
        *db = (*db).Where(prefix+"workspace_id = ?", q.WorkspaceID)
    }

    return nil
}
```

### 1.6 Update Handlers (wacraft-server)

All resource handlers must:

1. Extract workspace from context (set by middleware)
2. Include workspace_id in all queries
3. Validate workspace ownership on create/update/delete

**Example**: `src/contact/handler/get.go`

```go
func Get(c *fiber.Ctx) error {
    workspace := c.Locals("workspace").(*workspace_entity.Workspace)

    query := new(contact_model.QueryPaginated)
    if err := c.QueryParser(query); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(...)
    }

    // Force workspace scope
    query.WorkspaceID = &workspace.ID

    contacts, err := repository.GetPaginated(...)
    // ...
}
```

### 1.7 New API Routes

```
# Workspace Management
POST   /workspace                           Create workspace
GET    /workspace                           List user's workspaces
GET    /workspace/:workspace_id             Get workspace details
PATCH  /workspace/:workspace_id             Update workspace
DELETE /workspace/:workspace_id             Delete workspace

# Workspace Members
POST   /workspace/:workspace_id/member              Add member
GET    /workspace/:workspace_id/member              List members
PATCH  /workspace/:workspace_id/member/:user_id    Update member policies
DELETE /workspace/:workspace_id/member/:user_id    Remove member

# All existing routes now require X-Workspace-ID header
GET    /contact                             (requires X-Workspace-ID)
POST   /contact                             (requires X-Workspace-ID)
...
```

### 1.8 Tasks Checklist - Phase 1

- [ ] Create `wacraft-core/src/workspace/` package structure
- [ ] Implement Workspace entity and models
- [ ] Implement WorkspaceMember entity and models
- [ ] Implement WorkspaceMemberPolicy entity and models
- [ ] Implement Policy constants and helper functions
- [ ] Add WorkspaceID to Contact entity
- [ ] Add WorkspaceID to MessagingProduct entity
- [ ] Add WorkspaceID to Campaign entity
- [ ] Add WorkspaceID to Webhook entity
- [ ] Update QueryPaginated models with WorkspaceID filter
- [ ] Create workspace tables migration
- [ ] Create data migration to default workspace
- [ ] Implement WorkspaceMiddleware
- [ ] Implement RequirePolicy middleware
- [ ] Create workspace handlers (CRUD)
- [ ] Create workspace member handlers
- [ ] Update all existing handlers to scope by workspace
- [ ] Update router to include workspace routes
- [ ] Update router to apply workspace middleware to existing routes
- [ ] Add X-Workspace-ID header to Swagger documentation
- [ ] Write unit tests for workspace service
- [ ] Write integration tests for workspace API

---

## Phase 2: Phone Configuration Management

**Goal**: Move WhatsApp credentials from environment variables to database, enabling multiple phone numbers per workspace.

### 2.1 New Entity (wacraft-core)

#### PhoneConfig Entity

**File**: `wacraft-core/src/phone-config/entity/phone-config.go`

```go
type PhoneConfig struct {
    Name              string    `json:"name" gorm:"not null"`                    // Friendly name
    WorkspaceID       uuid.UUID `json:"workspace_id" gorm:"type:uuid;not null;index"`
    WabaID            string    `json:"waba_id" gorm:"not null"`                 // WhatsApp Business Account ID
    WabaAccountID     string    `json:"waba_account_id" gorm:"not null"`         // WhatsApp Account ID
    PhoneNumberID     string    `json:"phone_number_id" gorm:"not null;uniqueIndex"` // Unique phone number ID
    DisplayPhone      string    `json:"display_phone" gorm:"not null"`           // Display phone number
    AccessToken       string    `json:"-" gorm:"not null"`                       // Encrypted, never exposed in JSON
    MetaAppSecret     string    `json:"-" gorm:"not null"`                       // Encrypted, never exposed in JSON
    WebhookVerifyToken string   `json:"-" gorm:"not null"`                       // Encrypted, never exposed in JSON
    IsActive          bool      `json:"is_active" gorm:"default:true"`

    Workspace *workspace_entity.Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

### 2.2 Link MessagingProduct to PhoneConfig

**File**: `wacraft-core/src/messaging-product/entity/messaging-product.go`

```go
type MessagingProduct struct {
    Name          messaging_product_model.MessagingProductName `json:"name,omitempty" gorm:"not null"`
    WorkspaceID   uuid.UUID                                    `json:"workspace_id" gorm:"type:uuid;not null;index"`
    PhoneConfigID *uuid.UUID                                   `json:"phone_config_id,omitempty" gorm:"type:uuid;index"` // NEW

    Workspace   *workspace_entity.Workspace       `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
    PhoneConfig *phone_config_entity.PhoneConfig  `json:"phone_config,omitempty" gorm:"foreignKey:PhoneConfigID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"` // NEW

    common_model.Audit
}
```

### 2.3 Encryption Service

**File**: `wacraft-core/src/crypto/service/encryption.go`

```go
// Encrypt sensitive fields before storing
func EncryptField(plaintext string, key []byte) (string, error) {
    // AES-256-GCM encryption
    // Return base64-encoded ciphertext
}

// Decrypt fields when reading
func DecryptField(ciphertext string, key []byte) (string, error) {
    // AES-256-GCM decryption
}
```

### 2.4 Update Webhook Handler

**File**: `src/webhook/handler/receive.go`

```go
func ReceiveWhatsAppWebhook(c *fiber.Ctx) error {
    // 1. Parse webhook payload to extract phone_number_id
    // 2. Look up PhoneConfig by phone_number_id
    // 3. Verify X-Hub-Signature-256 using PhoneConfig.MetaAppSecret
    // 4. Process webhook with workspace context
    // 5. Route to appropriate workspace handlers/websockets
}
```

### 2.5 Update Message Sending Service

**File**: `src/message/service/send.go`

```go
func SendMessage(workspace *Workspace, phoneConfigID uuid.UUID, message *MessageRequest) error {
    // 1. Load PhoneConfig from database
    // 2. Decrypt access token
    // 3. Initialize WhatsApp client with decrypted credentials
    // 4. Send message
    // 5. Store message with workspace_id and messaging_product_id
}
```

### 2.6 Environment Variable Fallback

During transition, support both database and env var configurations:

```go
func GetPhoneConfig(workspaceID uuid.UUID, phoneConfigID *uuid.UUID) (*PhoneConfig, error) {
    if phoneConfigID != nil {
        // Load from database
        return loadFromDB(phoneConfigID)
    }

    // Fallback to env vars (deprecated)
    if env.WabaID != "" {
        return &PhoneConfig{
            WabaID:      env.WabaID,
            AccessToken: env.WabaAccessToken,
            // ...
        }, nil
    }

    return nil, errors.New("no phone configuration found")
}
```

### 2.7 New API Routes

```
# Phone Configuration Management
POST   /workspace/:workspace_id/phone-config           Create phone config
GET    /workspace/:workspace_id/phone-config           List phone configs
GET    /workspace/:workspace_id/phone-config/:id       Get phone config
PATCH  /workspace/:workspace_id/phone-config/:id       Update phone config
DELETE /workspace/:workspace_id/phone-config/:id       Delete phone config
POST   /workspace/:workspace_id/phone-config/:id/test  Test phone config connectivity
```

### 2.8 Webhook Registration Flow

When creating a PhoneConfig, the system should:

1. Generate unique webhook verify token
2. Provide webhook URL to user: `https://api.example.com/webhook/whatsapp/{phone_config_id}`
3. User registers this URL in Meta Business Manager
4. System validates webhook verification request

### 2.9 Tasks Checklist - Phase 2

- [ ] Create `wacraft-core/src/phone-config/` package structure
- [ ] Implement PhoneConfig entity and models
- [ ] Implement field encryption service
- [ ] Add PhoneConfigID to MessagingProduct entity
- [ ] Create phone_configs table migration
- [ ] Create migration to optionally migrate env vars to database
- [ ] Implement phone config handlers (CRUD)
- [ ] Implement phone config test endpoint
- [ ] Update webhook receive handler for multi-phone routing
- [ ] Update message send service to use PhoneConfig
- [ ] Update WhatsApp template service to use PhoneConfig
- [ ] Add phone config routes to router
- [ ] Implement webhook verification endpoint per phone config
- [ ] Add deprecation warnings for env var usage
- [ ] Write unit tests for encryption service
- [ ] Write integration tests for phone config API
- [ ] Document webhook setup process

---

## Phase 3: Open Registration & User Self-Service

**Goal**: Allow any user to register, create workspaces, and manage their own resources.

### 3.1 User Registration Flow

#### Registration Entity

**File**: `wacraft-core/src/user/entity/email-verification.go`

```go
type EmailVerification struct {
    UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
    Token     string    `json:"-" gorm:"not null;uniqueIndex"`
    ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
    Verified  bool      `json:"verified" gorm:"default:false"`

    User *User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

#### Workspace Invitation Entity

**File**: `wacraft-core/src/workspace/entity/workspace-invitation.go`

```go
type WorkspaceInvitation struct {
    WorkspaceID uuid.UUID                  `json:"workspace_id" gorm:"type:uuid;not null;index"`
    Email       string                     `json:"email" gorm:"not null;index"`
    Token       string                     `json:"-" gorm:"not null;uniqueIndex"`
    Policies    []workspace_model.Policy   `json:"policies" gorm:"serializer:json;type:jsonb"`
    ExpiresAt   time.Time                  `json:"expires_at" gorm:"not null"`
    AcceptedAt  *time.Time                 `json:"accepted_at,omitempty"`
    InvitedBy   uuid.UUID                  `json:"invited_by" gorm:"type:uuid;not null"`

    Workspace *Workspace        `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
    Inviter   *user_entity.User `json:"inviter,omitempty" gorm:"foreignKey:InvitedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

### 3.2 Email Service Interface

**File**: `src/email/service/interface.go`

```go
type EmailService interface {
    SendVerificationEmail(to string, token string) error
    SendWorkspaceInvitation(to string, workspace string, inviter string, token string) error
    SendPasswordReset(to string, token string) error
}
```

Implementation options:

- SMTP (default)
- SendGrid
- AWS SES
- Mailgun

### 3.3 Registration API

```
POST /auth/register
{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securepassword"
}

Response: 201 Created
{
    "message": "Verification email sent",
    "user_id": "uuid"
}
```

### 3.4 Email Verification API

```
GET /auth/verify-email?token={token}

Response: 200 OK (redirects to app)
```

### 3.5 Workspace Invitation Flow

```
# Admin invites user
POST /workspace/:workspace_id/invitation
{
    "email": "newuser@example.com",
    "policies": ["contact.read", "contact.manage", "message.read", "message.send"]
}

# User accepts invitation
POST /auth/accept-invitation?token={token}
{
    "name": "New User",      # Only if user doesn't exist
    "password": "password"   # Only if user doesn't exist
}
```

### 3.6 Rate Limiting

**File**: `src/auth/middleware/rate-limit.go`

```go
// Rate limit registration attempts
var registrationLimiter = limiter.New(limiter.Config{
    Max:        5,              // 5 attempts
    Expiration: 1 * time.Hour,  // per hour
    KeyGenerator: func(c *fiber.Ctx) string {
        return c.IP()
    },
})

// Rate limit login attempts
var loginLimiter = limiter.New(limiter.Config{
    Max:        10,
    Expiration: 15 * time.Minute,
    KeyGenerator: func(c *fiber.Ctx) string {
        return c.IP() + ":" + c.FormValue("email")
    },
})
```

### 3.7 Password Reset Flow

```
# Request password reset
POST /auth/forgot-password
{
    "email": "user@example.com"
}

# Reset password with token
POST /auth/reset-password
{
    "token": "reset-token",
    "password": "newpassword"
}
```

### 3.8 Remove Super User Requirement

Update migrations to make super user optional:

```go
func upAddSuUser(ctx context.Context, db *sql.DB) error {
    suPassword := os.Getenv("SU_PASSWORD")
    if suPassword == "" {
        // Skip super user creation if not configured
        log.Println("SU_PASSWORD not set, skipping super user creation")
        return nil
    }
    // ... existing logic
}
```

### 3.9 Auto-Create Personal Workspace

When a user registers and verifies their email:

1. Create a personal workspace named "{User's Name}'s Workspace"
2. Add user as workspace admin
3. Redirect to workspace setup wizard

### 3.10 Tasks Checklist - Phase 3

- [ ] Create EmailVerification entity
- [ ] Create WorkspaceInvitation entity
- [ ] Create PasswordResetToken entity
- [ ] Implement email service interface
- [ ] Implement SMTP email service
- [ ] Create registration handler
- [ ] Create email verification handler
- [ ] Create workspace invitation handlers
- [ ] Create password reset handlers
- [ ] Implement rate limiting middleware
- [ ] Add rate limiting to auth routes
- [ ] Update super user migration to be optional
- [ ] Implement auto-workspace creation on registration
- [ ] Add registration routes to router
- [ ] Configure email service via environment variables
- [ ] Write unit tests for registration flow
- [ ] Write integration tests for invitation flow
- [ ] Add email templates

---

## Phase 4: Deprecate User.Role, Complete Policy Migration

**Goal**: Fully transition from role-based to policy-based access control.

### 4.1 Deprecation Strategy

1. Keep `User.Role` field but mark as deprecated
2. Add migration to convert existing roles to workspace policies
3. Update all middleware to use policies instead of roles
4. Remove role-based middleware after transition period

### 4.2 Role to Policy Mapping

```go
var RoleToPoliciesMap = map[user_model.Role][]workspace_model.Policy{
    user_model.Admin: workspace_model.AdminPolicies,
    user_model.User:  workspace_model.MemberPolicies,
    user_model.Developer: []workspace_model.Policy{
        workspace_model.PolicyMessageRead,
        workspace_model.PolicyMessageSend,
        workspace_model.PolicyWebhookRead,
        workspace_model.PolicyWebhookManage,
    },
    user_model.Automation: []workspace_model.Policy{
        workspace_model.PolicyMessageRead,
        workspace_model.PolicyMessageSend,
        workspace_model.PolicyContactRead,
    },
}
```

### 4.3 Migration Script

**File**: `src/database/migrations/YYYYMMDDHHMMSS_migrate_roles_to_policies.go`

```go
func upMigrateRolesToPolicies(ctx context.Context, db *sql.DB) error {
    // For each user in each workspace:
    // 1. Get user's role
    // 2. Map role to policies
    // 3. Create WorkspaceMemberPolicy records
    // 4. Log migration for audit
    return nil
}
```

### 4.4 Update Middleware

Replace:

```go
// OLD
router.Use(auth_middleware.SuMiddleware())
```

With:

```go
// NEW
router.Use(workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceAdmin))
```

### 4.5 Backward Compatibility Period

During transition:

1. Both role and policy checks work
2. Log warnings when role-based access is used
3. After 2 releases, remove role-based middleware entirely

### 4.6 Tasks Checklist - Phase 4

- [ ] Add deprecation notice to User.Role field
- [ ] Create role-to-policy mapping
- [ ] Create migration to convert roles to policies
- [ ] Update SuMiddleware to use policies (with deprecation warning)
- [ ] Update RoleMiddleware to use policies (with deprecation warning)
- [ ] Update all routes to use RequirePolicy middleware
- [ ] Add logging for deprecated role usage
- [ ] Create documentation for policy migration
- [ ] Plan timeline for role removal
- [ ] Write migration rollback procedure

---

## Phase 5: Advanced Features & Optimization

**Goal**: Add advanced multi-tenancy features and optimize for scale.

### 5.1 Workspace Quotas

**File**: `wacraft-core/src/workspace/entity/workspace-quota.go`

```go
type WorkspaceQuota struct {
    WorkspaceID         uuid.UUID `json:"workspace_id" gorm:"type:uuid;not null;uniqueIndex"`
    MaxPhoneConfigs     int       `json:"max_phone_configs" gorm:"default:1"`
    MaxMembers          int       `json:"max_members" gorm:"default:5"`
    MaxContactsPerMonth int       `json:"max_contacts_per_month" gorm:"default:1000"`
    MaxMessagesPerMonth int       `json:"max_messages_per_month" gorm:"default:10000"`

    Workspace *Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

### 5.2 Usage Tracking

**File**: `wacraft-core/src/workspace/entity/workspace-usage.go`

```go
type WorkspaceUsage struct {
    WorkspaceID   uuid.UUID `json:"workspace_id" gorm:"type:uuid;not null;index:idx_usage_workspace_month,priority:1"`
    Month         time.Time `json:"month" gorm:"not null;index:idx_usage_workspace_month,priority:2"` // First day of month
    ContactsCount int       `json:"contacts_count" gorm:"default:0"`
    MessagesCount int       `json:"messages_count" gorm:"default:0"`

    Workspace *Workspace `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

    common_model.Audit
}
```

### 5.3 Audit Logging

**File**: `wacraft-core/src/audit/entity/audit-log.go`

```go
type AuditLog struct {
    WorkspaceID  uuid.UUID       `json:"workspace_id" gorm:"type:uuid;not null;index"`
    UserID       uuid.UUID       `json:"user_id" gorm:"type:uuid;not null;index"`
    Action       string          `json:"action" gorm:"not null;index"` // create, update, delete, send_message, etc.
    ResourceType string          `json:"resource_type" gorm:"not null;index"` // contact, message, phone_config, etc.
    ResourceID   uuid.UUID       `json:"resource_id" gorm:"type:uuid"`
    OldValue     json.RawMessage `json:"old_value,omitempty" gorm:"type:jsonb"`
    NewValue     json.RawMessage `json:"new_value,omitempty" gorm:"type:jsonb"`
    IPAddress    string          `json:"ip_address,omitempty"`
    UserAgent    string          `json:"user_agent,omitempty"`

    common_model.Audit
}
```

### 5.4 WebSocket Workspace Isolation

Update WebSocket handlers to only broadcast to workspace members:

```go
func BroadcastToWorkspace(workspaceID uuid.UUID, event string, data any) {
    // Only send to connections belonging to workspace members
}
```

### 5.5 Database Query Optimization

Add database-level row security (optional, for PostgreSQL):

```sql
-- Enable RLS on contacts table
ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see contacts in their workspaces
CREATE POLICY workspace_isolation ON contacts
    USING (workspace_id IN (
        SELECT workspace_id FROM workspace_members
        WHERE user_id = current_setting('app.current_user_id')::uuid
    ));
```

### 5.6 Caching Strategy

- Cache workspace membership checks (Redis/in-memory)
- Cache phone config lookups
- Cache policy checks per user-workspace pair
- Invalidate on membership/policy changes

### 5.7 Tasks Checklist - Phase 5

- [ ] Implement WorkspaceQuota entity and enforcement
- [ ] Implement WorkspaceUsage tracking
- [ ] Implement AuditLog entity and middleware
- [ ] Update WebSocket handlers for workspace isolation
- [ ] Implement caching layer for membership checks
- [ ] Add quota enforcement middleware
- [ ] Create usage analytics dashboard API
- [ ] Implement audit log API with filtering
- [ ] Add database indexes for multi-tenant queries
- [ ] Performance test with multiple workspaces
- [ ] Document scaling considerations

---

## Database Schema Overview (Final State)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              MULTI-TENANT SCHEMA                             │
└──────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐       ┌───────────────────┐       ┌─────────────────────────┐
│    users    │       │    workspaces     │       │   workspace_members     │
├─────────────┤       ├───────────────────┤       ├─────────────────────────┤
│ id (PK)     │       │ id (PK)           │       │ id (PK)                 │
│ name        │◄──────│ created_by (FK)   │       │ workspace_id (FK)  ─────┼──┐
│ email       │       │ name              │       │ user_id (FK)       ─────┼──┼──┐
│ password    │       │ slug              │       │ created_at              │  │  │
│ role (DEP)  │       │ description       │       │ updated_at              │  │  │
│ created_at  │       │ created_at        │       └─────────────────────────┘  │  │
│ updated_at  │       │ updated_at        │                    │               │  │
└─────────────┘       └───────────────────┘                    │               │  │
      │                        │                               ▼               │  │
      │                        │                ┌─────────────────────────────┐│  │
      │                        │                │ workspace_member_policies   ││  │
      │                        │                ├─────────────────────────────┤│  │
      │                        │                │ id (PK)                     ││  │
      │                        │                │ workspace_member_id (FK) ───┼┘  │
      │                        │                │ policy                      │   │
      │                        │                │ created_at                  │   │
      │                        │                │ updated_at                  │   │
      │                        │                └─────────────────────────────┘   │
      │                        │                                                  │
      │                        ▼                                                  │
      │         ┌───────────────────────────┐                                     │
      │         │      phone_configs        │                                     │
      │         ├───────────────────────────┤                                     │
      │         │ id (PK)                   │                                     │
      │         │ workspace_id (FK)    ─────┼──┐                                  │
      │         │ name                      │  │                                  │
      │         │ waba_id                   │  │                                  │
      │         │ phone_number_id (UNIQUE)  │  │                                  │
      │         │ access_token (encrypted)  │  │                                  │
      │         │ meta_app_secret (enc)     │  │                                  │
      │         │ is_active                 │  │                                  │
      │         │ created_at                │  │                                  │
      │         │ updated_at                │  │                                  │
      │         └───────────────────────────┘  │                                  │
      │                        │               │                                  │
      │                        ▼               │                                  │
      │         ┌───────────────────────────┐  │   ┌─────────────────────────┐   │
      │         │   messaging_products      │  │   │       contacts          │   │
      │         ├───────────────────────────┤  │   ├─────────────────────────┤   │
      │         │ id (PK)                   │  │   │ id (PK)                 │   │
      │         │ workspace_id (FK)    ─────┼──┼───│ workspace_id (FK)       │   │
      │         │ phone_config_id (FK) ─────┼──┘   │ name                    │   │
      │         │ name                      │      │ email                   │   │
      │         │ created_at                │      │ photo_path              │   │
      │         │ updated_at                │      │ created_at              │   │
      │         └───────────────────────────┘      │ updated_at              │   │
      │                        │                   └─────────────────────────┘   │
      │                        │                              │                  │
      │                        ▼                              │                  │
      │         ┌───────────────────────────────────────────────────────────┐   │
      │         │              messaging_product_contacts                   │   │
      │         ├───────────────────────────────────────────────────────────┤   │
      │         │ id (PK)                                                   │   │
      │         │ messaging_product_id (FK)                                 │   │
      │         │ contact_id (FK) ──────────────────────────────────────────┼───┘
      │         │ product_details (JSONB)                                   │
      │         │ blocked                                                   │
      │         │ created_at                                                │
      │         │ updated_at                                                │
      │         └───────────────────────────────────────────────────────────┘
      │                        │
      │                        ▼
      │         ┌───────────────────────────────────────────────────────────┐
      │         │                       messages                            │
      │         ├───────────────────────────────────────────────────────────┤
      │         │ id (PK)                                                   │
      │         │ messaging_product_id (FK)                                 │
      │         │ from_id (FK to messaging_product_contacts)                │
      │         │ to_id (FK to messaging_product_contacts)                  │
      │         │ sender_data (JSONB)                                       │
      │         │ receiver_data (JSONB)                                     │
      │         │ product_data (JSONB)                                      │
      │         │ created_at                                                │
      │         │ updated_at                                                │
      │         │ deleted_at                                                │
      │         └───────────────────────────────────────────────────────────┘

Note: Messages inherit workspace scope through messaging_product_id → workspace_id
```

---

## API Changes Summary

### New Endpoints

| Method | Endpoint                           | Description             | Auth   |
| ------ | ---------------------------------- | ----------------------- | ------ |
| POST   | `/auth/register`                   | User registration       | Public |
| GET    | `/auth/verify-email`               | Email verification      | Public |
| POST   | `/auth/forgot-password`            | Request password reset  | Public |
| POST   | `/auth/reset-password`             | Reset password          | Public |
| POST   | `/auth/accept-invitation`          | Accept workspace invite | Public |
| POST   | `/workspace`                       | Create workspace        | User   |
| GET    | `/workspace`                       | List user workspaces    | User   |
| GET    | `/workspace/:id`                   | Get workspace           | Member |
| PATCH  | `/workspace/:id`                   | Update workspace        | Admin  |
| DELETE | `/workspace/:id`                   | Delete workspace        | Admin  |
| POST   | `/workspace/:id/member`            | Add member              | Admin  |
| GET    | `/workspace/:id/member`            | List members            | Member |
| PATCH  | `/workspace/:id/member/:uid`       | Update member           | Admin  |
| DELETE | `/workspace/:id/member/:uid`       | Remove member           | Admin  |
| POST   | `/workspace/:id/invitation`        | Send invitation         | Admin  |
| POST   | `/workspace/:id/phone-config`      | Create phone config     | Admin  |
| GET    | `/workspace/:id/phone-config`      | List phone configs      | Member |
| PATCH  | `/workspace/:id/phone-config/:pid` | Update phone config     | Admin  |
| DELETE | `/workspace/:id/phone-config/:pid` | Delete phone config     | Admin  |

### Modified Endpoints

All existing resource endpoints now require `X-Workspace-ID` header:

| Method | Endpoint               | Header Required |
| ------ | ---------------------- | --------------- |
| \*     | `/contact/*`           | X-Workspace-ID  |
| \*     | `/message/*`           | X-Workspace-ID  |
| \*     | `/campaign/*`          | X-Workspace-ID  |
| \*     | `/webhook/*`           | X-Workspace-ID  |
| \*     | `/messaging-product/*` | X-Workspace-ID  |
| \*     | `/whatsapp-template/*` | X-Workspace-ID  |

### Deprecated Endpoints

| Method | Endpoint               | Replacement                                    |
| ------ | ---------------------- | ---------------------------------------------- |
| POST   | `/user` (admin create) | `/auth/register` + `/workspace/:id/invitation` |

---

## Environment Variables

### New Variables

```bash
# Email Service
EMAIL_PROVIDER=smtp                    # smtp, sendgrid, ses, mailgun
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user
SMTP_PASSWORD=password
SMTP_FROM=noreply@example.com

# Encryption
ENCRYPTION_KEY=32-byte-base64-key     # For encrypting phone config secrets

# Registration
ALLOW_REGISTRATION=true               # Enable/disable open registration
REQUIRE_EMAIL_VERIFICATION=true       # Require email verification

# Rate Limiting
RATE_LIMIT_REGISTRATION=5             # Registrations per hour per IP
RATE_LIMIT_LOGIN=10                   # Login attempts per 15 min per IP+email
```

### Deprecated Variables (Phase 2+)

```bash
# These become optional fallbacks, then fully deprecated
WABA_ID                               # Use phone_configs table
WABA_ACCESS_TOKEN                     # Use phone_configs table
WABA_ACCOUNT_ID                       # Use phone_configs table
META_APP_SECRET                       # Use phone_configs table
WEBHOOK_VERIFY_TOKEN                  # Use phone_configs table
```

---

## Migration Strategy

### Zero-Downtime Approach

1. **Phase 1**: Additive changes only (new tables, nullable columns)
2. **Data Migration**: Background job to assign existing data to default workspace
3. **Column Constraints**: Make workspace_id NOT NULL after migration
4. **API Versioning**: Support both old (no workspace) and new (workspace required) API simultaneously
5. **Deprecation Period**: 2 releases before removing old API support

### Rollback Plan

Each phase includes:

- Down migration scripts
- Feature flags to disable new functionality
- Documented manual rollback procedures

---

## Testing Strategy

### Unit Tests

- Workspace service (CRUD, membership, policies)
- Policy checking logic
- Phone config encryption/decryption
- Email service (mocked)

### Integration Tests

- Workspace creation flow
- Member invitation flow
- Cross-workspace isolation verification
- Webhook routing to correct workspace
- Message sending with phone config

### Load Tests

- Multiple workspaces with concurrent requests
- Webhook processing under load
- WebSocket connections per workspace

---

## Timeline Recommendation

| Phase | Scope               | Dependencies |
| ----- | ------------------- | ------------ |
| 1     | Core Multi-Tenancy  | None         |
| 2     | Phone Configuration | Phase 1      |
| 3     | Open Registration   | Phase 1      |
| 4     | Role Deprecation    | Phase 1, 3   |
| 5     | Advanced Features   | Phase 1-4    |

Phases 2 and 3 can be developed in parallel after Phase 1 is complete.

---

## Open Questions

1. **Workspace deletion**: Soft delete or hard delete? What happens to messages/contacts?
2. **Data export**: Should users be able to export workspace data?
3. **Workspace transfer**: Can ownership be transferred?
4. **Cross-workspace contacts**: Can a contact exist in multiple workspaces?
5. **Billing integration**: Will quotas tie into a billing system?
6. **SSO support**: Should Phase 3 include OAuth providers (Google, Microsoft)?
