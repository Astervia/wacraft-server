package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
	"gorm.io/gorm"
)

func init() {
	goose.AddMigrationContext(upUpdateSuRole, downUpdateSuRole)
}

func upUpdateSuRole(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	// 2. Update the sudo user's role to 'admin'
	var sudoUser user_entity.User
	err := db.Where("email = ?", "su@sudo").First(&sudoUser).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			pterm.DefaultLogger.Error(fmt.Sprintf("Error fetching super user: %v", err))
			return err
		}

		// Create the sudo user with admin role
		sudoUser = user_entity.User{
			Name:     "su",
			Email:    "su@sudo",
			Password: env.SuPassword, // Plain password (will be hashed by BeforeCreate)
			Role:     &user_model.Admin,
		}

		if err := db.Create(&sudoUser).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("Error creating super user: %v", err))
			return err
		}
		pterm.DefaultLogger.Info("Created super user 'su@sudo' with role 'admin'.")
		return nil
	}
	if sudoUser.Role == nil || *sudoUser.Role != user_model.Admin {
		sudoUser.Role = &user_model.Admin
		if err := db.Save(&sudoUser).Error; err != nil {
			return err
		}
		pterm.DefaultLogger.Info("Updated su role")
		return nil
	}
	pterm.DefaultLogger.Info("su already has role 'admin'.")

	return nil
}

func downUpdateSuRole(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	// Be cautious with this step in production environments
	if err := db.Where("email = ?", "su@sudo").Delete(&user_entity.User{}).Error; err != nil {
		return err
	}
	pterm.DefaultLogger.Info("Deleted super user 'su@sudo'.")

	return nil
}
