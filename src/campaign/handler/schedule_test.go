package campaign_handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- Test bootstrap ---

func TestMain(m *testing.M) {
	validators.InitValidators()
	database.DB.AutoMigrate(
		&workspace_entity.Workspace{},
		&campaign_entity.Campaign{},
	)
	// Drop FK constraints so test fixtures can use random UUIDs without
	// setting up the full messaging-product / workspace parent rows.
	database.DB.Exec(`ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_messaging_product`)
	database.DB.Exec(`ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_workspace`)
	os.Exit(m.Run())
}

// --- Helpers ---

func newScheduleApp() *fiber.App {
	app := fiber.New()
	// Inject a fake workspace into locals so the handler can retrieve it.
	app.Use(func(c *fiber.Ctx) error {
		ws := &workspace_entity.Workspace{}
		ws.ID = testWorkspaceID
		c.Locals("workspace", ws)
		return c.Next()
	})
	app.Post("/campaign/schedule", Schedule)
	app.Delete("/campaign/schedule", Unschedule)
	return app
}

var testWorkspaceID = uuid.New()

func createTestCampaign(t *testing.T, status string) campaign_entity.Campaign {
	t.Helper()
	id := uuid.New()
	mpID := uuid.New()

	// Insert a fake messaging product first (or skip FK if not enforced in test).
	// We use Exec to bypass FK checks since this is a unit test.
	err := database.DB.Exec(
		`INSERT INTO campaigns (id, name, messaging_product_id, workspace_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NOW(), NOW())`,
		id, "Test Campaign", mpID, testWorkspaceID, status,
	).Error
	if err != nil {
		t.Fatalf("createTestCampaign: %v", err)
	}

	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM campaigns WHERE id = ?", id)
	})

	c := campaign_entity.Campaign{}
	c.ID = id
	c.WorkspaceID = &testWorkspaceID
	c.MessagingProductID = &mpID
	c.Status = status
	return c
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("jsonBody: %v", err)
	}
	return bytes.NewBuffer(b)
}

// --- Schedule tests ---

func TestSchedule_Success(t *testing.T) {
	campaign := createTestCampaign(t, "draft")
	app := newScheduleApp()

	scheduledAt := time.Now().UTC().Add(time.Hour)
	body := jsonBody(t, campaign_model.ScheduleCampaign{
		ID:          campaign.ID,
		ScheduledAt: &scheduledAt,
	})

	req := httptest.NewRequest("POST", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Schedule success: got %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	// Verify DB was updated.
	var updated campaign_entity.Campaign
	database.DB.First(&updated, campaign.ID)
	if updated.Status != "scheduled" {
		t.Errorf("status after schedule: got %q, want %q", updated.Status, "scheduled")
	}
	if updated.ScheduledAt == nil {
		t.Error("scheduled_at should not be nil after scheduling")
	}
}

func TestSchedule_AlreadyRunning_Returns409(t *testing.T) {
	campaign := createTestCampaign(t, "running")
	app := newScheduleApp()

	scheduledAt := time.Now().UTC().Add(time.Hour)
	body := jsonBody(t, campaign_model.ScheduleCampaign{
		ID:          campaign.ID,
		ScheduledAt: &scheduledAt,
	})

	req := httptest.NewRequest("POST", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("Schedule running: got %d, want %d", resp.StatusCode, fiber.StatusConflict)
	}
}

func TestSchedule_AlreadyCompleted_Returns409(t *testing.T) {
	campaign := createTestCampaign(t, "completed")
	app := newScheduleApp()

	scheduledAt := time.Now().UTC().Add(time.Hour)
	body := jsonBody(t, campaign_model.ScheduleCampaign{
		ID:          campaign.ID,
		ScheduledAt: &scheduledAt,
	})

	req := httptest.NewRequest("POST", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("Schedule completed: got %d, want %d", resp.StatusCode, fiber.StatusConflict)
	}
}

func TestSchedule_WrongWorkspace_Returns404(t *testing.T) {
	// Campaign belongs to a DIFFERENT workspace.
	otherWorkspaceID := uuid.New()
	mpID := uuid.New()
	id := uuid.New()
	database.DB.Exec(
		`INSERT INTO campaigns (id, name, messaging_product_id, workspace_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'draft', NOW(), NOW())`,
		id, "Other WS Campaign", mpID, otherWorkspaceID,
	)
	defer database.DB.Exec("DELETE FROM campaigns WHERE id = ?", id)

	app := newScheduleApp()
	scheduledAt := time.Now().UTC().Add(time.Hour)
	body := jsonBody(t, campaign_model.ScheduleCampaign{ID: id, ScheduledAt: &scheduledAt})

	req := httptest.NewRequest("POST", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, 5000)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Wrong workspace: got %d, want 404", resp.StatusCode)
	}
}

func TestSchedule_BadJSON_Returns400(t *testing.T) {
	app := newScheduleApp()
	req := httptest.NewRequest("POST", "/campaign/schedule", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, 5000)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Bad JSON: got %d, want 400", resp.StatusCode)
	}
}

// --- Unschedule tests ---

func TestUnschedule_Success(t *testing.T) {
	campaign := createTestCampaign(t, "scheduled")
	app := newScheduleApp()

	body := jsonBody(t, campaign_model.UnscheduleCampaign{ID: campaign.ID})
	req := httptest.NewRequest("DELETE", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Unschedule success: got %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var updated campaign_entity.Campaign
	database.DB.First(&updated, campaign.ID)
	if updated.Status != "draft" {
		t.Errorf("status after unschedule: got %q, want %q", updated.Status, "draft")
	}
}

func TestUnschedule_Running_Returns409(t *testing.T) {
	campaign := createTestCampaign(t, "running")
	app := newScheduleApp()

	body := jsonBody(t, campaign_model.UnscheduleCampaign{ID: campaign.ID})
	req := httptest.NewRequest("DELETE", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, 5000)
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("Unschedule running: got %d, want 409", resp.StatusCode)
	}
}

func TestUnschedule_WrongWorkspace_Returns404(t *testing.T) {
	otherWorkspaceID := uuid.New()
	mpID := uuid.New()
	id := uuid.New()
	database.DB.Exec(
		`INSERT INTO campaigns (id, name, messaging_product_id, workspace_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'scheduled', NOW(), NOW())`,
		id, "Other WS Campaign", mpID, otherWorkspaceID,
	)
	defer database.DB.Exec("DELETE FROM campaigns WHERE id = ?", id)

	app := newScheduleApp()
	body := jsonBody(t, campaign_model.UnscheduleCampaign{ID: id})
	req := httptest.NewRequest("DELETE", "/campaign/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, 5000)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Wrong workspace: got %d, want 404", resp.StatusCode)
	}
}
