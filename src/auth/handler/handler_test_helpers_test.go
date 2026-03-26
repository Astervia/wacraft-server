package auth_handler

import (
	"fmt"
	"os"
	"testing"
	"time"

	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	email_service "github.com/Astervia/wacraft-server/src/email/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/google/uuid"
)

// --- Mock EmailService ---

type MockEmailData struct {
	To, Name, Token, BaseURL string
}

type mockEmailService struct {
	VerificationCh  chan MockEmailData
	PasswordResetCh chan MockEmailData
	InvitationCh    chan MockEmailData
}

func (m *mockEmailService) SendVerificationEmail(to, name, token, baseURL string) error {
	m.VerificationCh <- MockEmailData{to, name, token, baseURL}
	return nil
}

func (m *mockEmailService) SendPasswordReset(to, name, token string) error {
	m.PasswordResetCh <- MockEmailData{to, name, token, ""}
	return nil
}

func (m *mockEmailService) SendWorkspaceInvitation(to, workspaceName, inviterName, token, baseURL string) error {
	m.InvitationCh <- MockEmailData{to, workspaceName, token, baseURL}
	return nil
}

func newMockEmailService() *mockEmailService {
	return &mockEmailService{
		VerificationCh:  make(chan MockEmailData, 1),
		PasswordResetCh: make(chan MockEmailData, 1),
		InvitationCh:    make(chan MockEmailData, 1),
	}
}

// installMock replaces DefaultEmailService and returns the mock.
// Registers a t.Cleanup to restore the original.
func installMock(t *testing.T) *mockEmailService {
	t.Helper()
	original := email_service.DefaultEmailService
	mock := newMockEmailService()
	email_service.DefaultEmailService = mock
	t.Cleanup(func() { email_service.DefaultEmailService = original })
	return mock
}

// --- Test DB Bootstrap ---

func TestMain(m *testing.M) {
	env.JwtSecret = "testing_secret_key"
	validators.InitValidators()
	database.DB.AutoMigrate(
		&user_entity.User{},
		&user_entity.EmailVerification{},
		&user_entity.PasswordResetToken{},
		&workspace_entity.Workspace{},
		&workspace_entity.WorkspaceMember{},
		&workspace_entity.WorkspaceMemberPolicy{},
	)
	os.Exit(m.Run())
}

// --- Helpers ---

func createTestUser(t *testing.T, verified bool) *user_entity.User {
	t.Helper()
	email := fmt.Sprintf("handler-%s@test.com", uuid.New().String())
	hashed, _ := crypto_service.HashPassword("ValidPass1!")
	id := uuid.New()
	if err := database.DB.Exec(
		"INSERT INTO users (id, email, password, name, role, email_verified) VALUES (?, ?, ?, ?, 'user', ?)",
		id, email, hashed, "Test User", verified,
	).Error; err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	user := &user_entity.User{}
	user.ID = id
	database.DB.First(user, id)
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM workspace_member_policies WHERE workspace_member_id IN (SELECT id FROM workspace_members WHERE user_id = ?)", id)
		database.DB.Exec("DELETE FROM workspace_members WHERE user_id = ?", id)
		database.DB.Exec("DELETE FROM workspaces WHERE created_by = ?", id)
		database.DB.Exec("DELETE FROM email_verifications WHERE user_id = ?", id)
		database.DB.Exec("DELETE FROM password_reset_tokens WHERE user_id = ?", id)
		database.DB.Exec("DELETE FROM users WHERE id = ?", id)
	})
	return user
}

func setEnv(t *testing.T, allowReg, requireVerify bool) {
	t.Helper()
	origAllow := env.AllowRegistration
	origVerify := env.RequireEmailVerification
	env.AllowRegistration = allowReg
	env.RequireEmailVerification = requireVerify
	t.Cleanup(func() {
		env.AllowRegistration = origAllow
		env.RequireEmailVerification = origVerify
	})
}

func waitForChannel(ch <-chan MockEmailData, timeout time.Duration) (MockEmailData, bool) {
	select {
	case data := <-ch:
		return data, true
	case <-time.After(timeout):
		return MockEmailData{}, false
	}
}
