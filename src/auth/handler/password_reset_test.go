package auth_handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- ForgotPassword ---

func TestForgotPassword_BadJSON(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/forgot-password", ForgotPassword)

	req := httptest.NewRequest("POST", "/auth/forgot-password", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestForgotPassword_InvalidEmail(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/forgot-password", ForgotPassword)

	body, _ := json.Marshal(map[string]string{"email": "not-an-email"})
	req := httptest.NewRequest("POST", "/auth/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestForgotPassword_UnknownEmail(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/forgot-password", ForgotPassword)

	body, _ := json.Marshal(map[string]string{"email": "nonexistent@test.com"})
	req := httptest.NewRequest("POST", "/auth/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	// 200 to prevent enumeration
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestForgotPassword_Success(t *testing.T) {
	mock := installMock(t)
	user := createTestUser(t, true)

	app := fiber.New()
	app.Post("/auth/forgot-password", ForgotPassword)

	body, _ := json.Marshal(map[string]string{"email": user.Email})
	req := httptest.NewRequest("POST", "/auth/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Validate the mock captured the async email send
	data, ok := waitForChannel(mock.PasswordResetCh, 2*time.Second)
	if !ok {
		t.Fatal("Expected password reset email to be sent via mock channel")
	}
	if data.To != user.Email {
		t.Errorf("Expected email to %s, got %s", user.Email, data.To)
	}
	if data.Token == "" {
		t.Error("Expected non-empty reset token")
	}
}

// --- ResetPassword ---

func TestResetPassword_BadJSON(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	body, _ := json.Marshal(map[string]string{"token": "nonexistent", "password": "NewPassword1!"})
	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	user := createTestUser(t, true)
	token := uuid.New().String()
	database.DB.Create(&user_entity.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
	})

	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	body, _ := json.Marshal(map[string]string{"token": token, "password": "NewPassword1!"})
	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000) // 30s for bcrypt
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestResetPassword_Success(t *testing.T) {
	user := createTestUser(t, true)
	token := uuid.New().String()
	database.DB.Create(&user_entity.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	body, _ := json.Marshal(map[string]string{"token": token, "password": "NewPassword1!"})
	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000) // 30s for bcrypt
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Confirm token was marked as used
	var resetToken user_entity.PasswordResetToken
	database.DB.Where("token = ?", token).First(&resetToken)
	if resetToken.UsedAt == nil {
		t.Error("Expected reset token to be marked as used")
	}
}

func TestResetPassword_ValidationError(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	// Missing password field
	body, _ := json.Marshal(map[string]string{"token": "some-token"})
	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestResetPassword_UsedToken(t *testing.T) {
	user := createTestUser(t, true)
	token := uuid.New().String()
	usedAt := time.Now().Add(-30 * time.Minute)
	database.DB.Create(&user_entity.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour), // not expired
		UsedAt:    &usedAt,                       // but already used
	})

	app := fiber.New()
	app.Post("/auth/reset-password", ResetPassword)

	body, _ := json.Marshal(map[string]string{"token": token, "password": "NewPassword1!"})
	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
