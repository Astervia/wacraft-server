package auth_handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const testTimeout = 30000 // 30s for bcrypt cost 14

func TestRegister_Disabled(t *testing.T) {
	setEnv(t, false, false) // AllowRegistration=false

	app := fiber.New()
	app.Post("/auth/register", Register)

	body, _ := json.Marshal(map[string]string{
		"name":     "Test User",
		"email":    "new@test.com",
		"password": "ValidPass1!",
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRegister_BadJSON(t *testing.T) {
	setEnv(t, true, false)

	app := fiber.New()
	app.Post("/auth/register", Register)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_ValidationError(t *testing.T) {
	setEnv(t, true, false)

	app := fiber.New()
	app.Post("/auth/register", Register)

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"name": "A"}) // too short, no email/password
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	setEnv(t, true, false)
	existingUser := createTestUser(t, true)

	app := fiber.New()
	app.Post("/auth/register", Register)

	body, _ := json.Marshal(map[string]string{
		"name":     "Test User",
		"email":    existingUser.Email,
		"password": "ValidPass1!",
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}

func TestRegister_Success_NoVerification(t *testing.T) {
	setEnv(t, true, false) // RequireEmailVerification=false

	app := fiber.New()
	app.Post("/auth/register", Register)

	email := fmt.Sprintf("register-%s@test.com", uuid.New().String())
	body, _ := json.Marshal(map[string]string{
		"name":     "Test Register",
		"email":    email,
		"password": "ValidPass1!",
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	// Decode response for cleanup
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["user_id"] == "" {
		t.Error("Expected user_id in response")
	}

	// Cleanup created user and relations
	uid, _ := uuid.Parse(result["user_id"])
	t.Cleanup(func() {
		cleanupRegisteredUser(uid)
	})
}

func TestRegister_Success_WithVerification(t *testing.T) {
	setEnv(t, true, true) // RequireEmailVerification=true
	mock := installMock(t)

	// Ensure JwtSecret is set for token generation
	origSecret := env.JwtSecret
	env.JwtSecret = "testing_secret_key"
	t.Cleanup(func() { env.JwtSecret = origSecret })

	app := fiber.New()
	app.Post("/auth/register", Register)

	email := fmt.Sprintf("register-verify-%s@test.com", uuid.New().String())
	body, _ := json.Marshal(map[string]string{
		"name":     "Test VerifyReg",
		"email":    email,
		"password": "ValidPass1!",
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	// Validate the mock captured the async verification email
	data, ok := waitForChannel(mock.VerificationCh, 2*time.Second)
	if !ok {
		t.Fatal("Expected verification email sent via mock channel")
	}
	if data.To != email {
		t.Errorf("Expected email to %s, got %s", email, data.To)
	}
	if data.Token == "" {
		t.Error("Expected non-empty verification token")
	}

	// Cleanup created user and relations
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if uid, parseErr := uuid.Parse(result["user_id"]); parseErr == nil {
		t.Cleanup(func() { cleanupRegisteredUser(uid) })
	}
}

// cleanupRegisteredUser removes a user created by Register and all cascaded entities.
func cleanupRegisteredUser(uid uuid.UUID) {
	database.DB.Exec("DELETE FROM workspace_member_policies WHERE workspace_member_id IN (SELECT id FROM workspace_members WHERE user_id = ?)", uid)
	database.DB.Exec("DELETE FROM workspace_members WHERE user_id = ?", uid)
	database.DB.Exec("DELETE FROM workspaces WHERE created_by = ?", uid)
	database.DB.Exec("DELETE FROM email_verifications WHERE user_id = ?", uid)
	database.DB.Exec("DELETE FROM password_reset_tokens WHERE user_id = ?", uid)
	database.DB.Exec("DELETE FROM users WHERE id = ?", uid)
}
