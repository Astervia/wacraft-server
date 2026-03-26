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

const verifyTestTimeout = 30000 // 30s for potential slow DB ops

func TestVerifyEmail_MissingToken(t *testing.T) {
	app := fiber.New()
	app.Get("/auth/verify-email", VerifyEmail)

	req := httptest.NewRequest("GET", "/auth/verify-email", nil)
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	app := fiber.New()
	app.Get("/auth/verify-email", VerifyEmail)

	req := httptest.NewRequest("GET", "/auth/verify-email?token=nonexistent", nil)
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestVerifyEmail_AlreadyVerified(t *testing.T) {
	user := createTestUser(t, false)
	token := uuid.New().String()
	database.DB.Create(&user_entity.EmailVerification{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Verified:  true,
	})

	app := fiber.New()
	app.Get("/auth/verify-email", VerifyEmail)

	req := httptest.NewRequest("GET", "/auth/verify-email?token="+token, nil)
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	user := createTestUser(t, false)
	token := uuid.New().String()
	database.DB.Create(&user_entity.EmailVerification{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		Verified:  false,
	})

	app := fiber.New()
	app.Get("/auth/verify-email", VerifyEmail)

	req := httptest.NewRequest("GET", "/auth/verify-email?token="+token, nil)
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	user := createTestUser(t, false)
	token := uuid.New().String()
	database.DB.Create(&user_entity.EmailVerification{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Verified:  false,
	})

	app := fiber.New()
	app.Get("/auth/verify-email", VerifyEmail)

	req := httptest.NewRequest("GET", "/auth/verify-email?token="+token, nil)
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Confirm DB state transitioned
	var verification user_entity.EmailVerification
	database.DB.Where("token = ?", token).First(&verification)
	if !verification.Verified {
		t.Error("Expected verification record marked as verified")
	}

	var updatedUser user_entity.User
	database.DB.First(&updatedUser, user.ID)
	if !updatedUser.EmailVerified {
		t.Error("Expected user.EmailVerified=true after verification")
	}
}

// --- ResendVerification ---

func TestResendVerification_BadJSON(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/resend-verification", ResendVerification)

	req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestResendVerification_UnknownEmail(t *testing.T) {
	app := fiber.New()
	app.Post("/auth/resend-verification", ResendVerification)

	body, _ := json.Marshal(map[string]string{"email": "nobody@nowhere.com"})
	req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	user := createTestUser(t, true) // verified=true
	app := fiber.New()
	app.Post("/auth/resend-verification", ResendVerification)

	body, _ := json.Marshal(map[string]string{"email": user.Email})
	req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestResendVerification_NotVerified(t *testing.T) {
	user := createTestUser(t, false) // not yet verified
	app := fiber.New()
	app.Post("/auth/resend-verification", ResendVerification)

	body, _ := json.Marshal(map[string]string{"email": user.Email})
	req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, verifyTestTimeout)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	// Current implementation returns 200 (TODO branch)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}
