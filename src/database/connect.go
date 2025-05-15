package database

import (
	"fmt"
	"os"

	"github.com/Astervia/wacraft-server/src/config/env"
	_ "github.com/lib/pq"
	"github.com/pterm/pterm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func connect() {
	pterm.DefaultLogger.Info("Connecting to database...")
	var err error = nil
	DB, err = gorm.Open(postgres.Open(env.DatabaseURL), &gorm.Config{})
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Error while connecting to database: %s", err))
		os.Exit(1)
	}
	pterm.DefaultLogger.Info("Database connection successful!")

	pterm.DefaultLogger.Info("Pinging to database...")
	undlDb, _ := DB.DB()
	err = undlDb.Ping()
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Error while pinging to database: %s", err))
		os.Exit(1)
	}
	pterm.DefaultLogger.Info("Database ping successful!")

	pterm.DefaultLogger.Info("Setting database connection limits...")
	setDBConnLimits()
	pterm.DefaultLogger.Info("Database connection limits successful!")
}

func setDBConnLimits() {
	sqlDB, err := DB.DB()
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Error while retrieving underlying SQL database: %s", err))
		os.Exit(1)
	}

	// Set maximum number of open connections to 10.
	sqlDB.SetMaxOpenConns(env.DatabaseMaxOpenConns)

	// Set maximum number of idle connections to 5.
	sqlDB.SetMaxIdleConns(env.DatabaseMaxIdleConns)

	// Set the maximum lifetime of a connection to 5 minutes.
	sqlDB.SetConnMaxLifetime(env.DatabaseConnMaxLifetime)
}
