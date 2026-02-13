package migrations

import (
	"context"
	"database/sql"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
	"gorm.io/gorm"
)

func init() {
	goose.AddMigrationContext(upSeedDefaultFreePlan, downSeedDefaultFreePlan)
}

func upSeedDefaultFreePlan(ctx context.Context, tx *sql.Tx) error {
	// Check if a default plan already exists
	var existing billing_entity.Plan
	err := database.DB.Where("is_default = ? AND slug = ?", true, "free").First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		plan := billing_entity.Plan{
			Name:            "Free",
			Slug:            "free",
			ThroughputLimit: env.DefaultFreeThroughput,
			WindowSeconds:   env.DefaultFreeWindow,
			DurationDays:    36500, // ~100 years (perpetual)
			PriceCents:      0,
			Currency:        "usd",
			IsDefault:       true,
			IsCustom:        false,
			Active:          true,
		}

		if err := database.DB.Create(&plan).Error; err != nil {
			return err
		}

		pterm.DefaultLogger.Info("Default free plan created")
	} else if err != nil {
		return err
	} else {
		pterm.DefaultLogger.Info("Default free plan already exists")
	}

	return nil
}

func downSeedDefaultFreePlan(ctx context.Context, tx *sql.Tx) error {
	return database.DB.Delete(&billing_entity.Plan{}, "slug = ? AND is_default = ?", "free", true).Error
}
