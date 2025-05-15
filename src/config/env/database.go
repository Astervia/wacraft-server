package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

var (
	DatabaseURL             string
	DatabaseMaxOpenConns    int           = 40               // For t2.micro on AWS RDS.
	DatabaseMaxIdleConns    int           = 20               // For t2.micro on AWS RDS. Using 2:1 ration on MaxOpenConns:MaxIdleConns.
	DatabaseConnMaxLifetime time.Duration = 30 * time.Minute // Considering conservative value since we have multiple transactions running to send a single message.
	DatabaseQueryTimeout    time.Duration = 5 * time.Minute  // Default timeout of 5 minutes
)

func loadDbEnv() {
	DatabaseURL = os.Getenv("DATABASE_URL")

	maxOpenConns := os.Getenv("DATABASE_MAX_OPEN_CONNS")
	maxOpenConnsInt, err := strconv.Atoi(maxOpenConns)
	if err == nil {
		DatabaseMaxOpenConns = maxOpenConnsInt
	}

	maxIdleConns := os.Getenv("DATABASE_MAX_IDLE_CONNS")
	maxIdleConnsInt, err := strconv.Atoi(maxIdleConns)
	if err == nil {
		DatabaseMaxIdleConns = maxIdleConnsInt
	}

	connMaxLifetimeMinutes := os.Getenv("DATABASE_CONN_MAX_LIFETIME_MINUTES")
	connMaxLifetimeMinutesInt, err := strconv.Atoi(connMaxLifetimeMinutes)
	if err == nil {
		DatabaseConnMaxLifetime = time.Duration(connMaxLifetimeMinutesInt) * time.Minute
	}

	queryTimeoutMinutes := os.Getenv("DATABASE_QUERY_TIMEOUT_MINUTES")
	queryTimeoutMinutesInt, err := strconv.Atoi(queryTimeoutMinutes)
	if err == nil {
		DatabaseQueryTimeout = time.Duration(queryTimeoutMinutesInt) * time.Minute
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf(
			"Database environment done with max open conns %d, max idle conns %d and max lifetime %s",
			DatabaseMaxOpenConns, DatabaseMaxIdleConns, DatabaseConnMaxLifetime),
	)
}
