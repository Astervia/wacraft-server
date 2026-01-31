package migrations

import (
	"context"
	"database/sql"

	entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
	"gorm.io/gorm"
)

func init() {
	goose.AddMigrationContext(upAddSuUser, downAddSuUser)
}

func upAddSuUser(ctx context.Context, tx *sql.Tx) error {
	// Check if a user with the email already exists
	var existingUser entity.User
	err := database.DB.Where("email = ?", "su@sudo").First(&existingUser).Error

	if err == gorm.ErrRecordNotFound {
		// No user exists with that email, create the sudo user
		sudoUser := entity.User{
			Name:          "su",
			Email:         "su@sudo",
			Password:      env.SuPassword, // Plain password (will be hashed by the model)
			EmailVerified: true,
		}

		err = database.DB.Create(&sudoUser).Error
		if err != nil {
			return err
		}
	} else if err != nil {
		// Error during the check, return the error
		return err
	} else {
		// User with the same email already exists, do nothing
		pterm.DefaultLogger.Warn("User with email 'su@sudo' already exists.")
	}

	return nil
}

func downAddSuUser(ctx context.Context, tx *sql.Tx) error {
	// Delete the sudo user using GORM's Delete method
	err := database.DB.Delete(&entity.User{}, "email = ?", "su@sudo").Error
	if err != nil {
		return err
	}

	return nil
}
