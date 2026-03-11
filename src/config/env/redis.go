package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

var (
	SyncBackend    string        = "memory"
	RedisURL       string        = "redis://localhost:6379"
	RedisPassword  string        = ""
	RedisDB        int           = 0
	RedisKeyPrefix string        = "wacraft:"
	RedisLockTTL   time.Duration = 30 * time.Second
	RedisCacheTTL  time.Duration = 5 * time.Minute
)

func loadRedisEnv() {
	if val := os.Getenv("SYNC_BACKEND"); val != "" {
		SyncBackend = val
	}

	if val := os.Getenv("REDIS_URL"); val != "" {
		RedisURL = val
	}

	RedisPassword = os.Getenv("REDIS_PASSWORD")

	if val := os.Getenv("REDIS_DB"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			RedisDB = parsed
		}
	}

	if val := os.Getenv("REDIS_KEY_PREFIX"); val != "" {
		RedisKeyPrefix = val
	}

	if val := os.Getenv("REDIS_LOCK_TTL"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			RedisLockTTL = parsed
		}
	}

	if val := os.Getenv("REDIS_CACHE_TTL"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			RedisCacheTTL = parsed
		}
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf(
			"Sync backend: %s", SyncBackend),
	)

	if SyncBackend == "redis" {
		pterm.DefaultLogger.Info(
			fmt.Sprintf(
				"Redis environment done with URL %s, DB %d, key prefix %s, lock TTL %s, cache TTL %s",
				RedisURL, RedisDB, RedisKeyPrefix, RedisLockTTL, RedisCacheTTL),
		)
	}
}
