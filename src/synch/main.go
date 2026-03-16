// Package synch initialises the distributed synchronisation factory from
// environment configuration and wires all sync primitives used throughout
// wacraft-server. Import this package with a blank identifier to trigger
// initialisation:
//
//	import _ "github.com/Astervia/wacraft-server/src/synch"
package synch

import (
	"os"

	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	synch "github.com/Astervia/wacraft-core/src/synch"
	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	campaign_handler "github.com/Astervia/wacraft-server/src/campaign/handler"
	campaign_service "github.com/Astervia/wacraft-server/src/campaign/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	status_handler "github.com/Astervia/wacraft-server/src/status/handler"
	whk_service "github.com/Astervia/wacraft-server/src/webhook-in/service"
	webhook_worker "github.com/Astervia/wacraft-server/src/webhook/worker"
	"github.com/pterm/pterm"
)

// SyncFactory is the global factory for distributed sync primitives.
// It is nil until init() runs.
var SyncFactory *synch.Factory

func init() {
	var redisClient *synch_redis.Client

	backend := synch.Backend(env.SyncBackend)

	if backend == synch.BackendRedis {
		var err error
		redisClient, err = synch_redis.NewClient(synch_redis.Config{
			URL:       env.RedisURL,
			Password:  env.RedisPassword,
			DB:        env.RedisDB,
			KeyPrefix: env.RedisKeyPrefix,
			LockTTL:   env.RedisLockTTL,
			CacheTTL:  env.RedisCacheTTL,
		})
		if err != nil {
			pterm.DefaultLogger.Error("Failed to create Redis client: " + err.Error())
			os.Exit(1)
		}

		// Verify connectivity at startup.
		if err := redisClient.PingWithTimeout(env.RedisLockTTL); err != nil {
			pterm.DefaultLogger.Error("Failed to connect to Redis: " + err.Error())
			os.Exit(1)
		}

		pterm.DefaultLogger.Info("Redis client connected successfully")
	}

	SyncFactory = synch.NewFactory(backend, redisClient)

	// Wire MessageStatusSync.
	if backend == synch.BackendRedis {
		message_service.SetStatusSynchronizer(
			message_service.NewRedisMessageStatusSync(redisClient),
		)
		pterm.DefaultLogger.Info("MessageStatusSync: using Redis backend")
	} else {
		pterm.DefaultLogger.Info("MessageStatusSync: using in-memory backend")
	}

	// Wire status deduplication lock.
	whk_service.SetStatusLock(synch.NewLock[string](SyncFactory))

	// Wire contact deduplication lock.
	campaign_service.SetContactLock(synch.NewLock[string](SyncFactory))

	// Wire campaign channel pool with distributed primitives.
	if backend == synch.BackendRedis {
		campaign_handler.SetSendCampaignPool(
			campaign_model.CreateChannelPoolWithDistributed(
				SyncFactory.NewCache(),
				SyncFactory.NewPubSub(),
			),
		)
		pterm.DefaultLogger.Info("CampaignChannelPool: using Redis backend")
	}

	// Wire delivery worker lock (Redis mode only — memory mode uses nil, no locking needed).
	if backend == synch.BackendRedis {
		webhook_worker.SetDeliveryLock(synch.NewLock[string](SyncFactory))
		pterm.DefaultLogger.Info("DeliveryWorker: using Redis lock backend")
	}

	// Wire WebSocket workspace managers with PubSub for cross-instance broadcast.
	if backend == synch.BackendRedis {
		message_handler.NewMessageWorkspaceManager.SetPubSub(
			SyncFactory.NewPubSub(),
			"workspace:messages",
		)
		status_handler.NewStatusWorkspaceManager.SetPubSub(
			SyncFactory.NewPubSub(),
			"workspace:statuses",
		)
		pterm.DefaultLogger.Info("WorkspaceChannelManagers: using Redis PubSub backend")
	}

	// Wire billing throughput counter.
	billing_service.SetThroughputCounter(
		billing_service.NewThroughputCounter(SyncFactory.NewCounter()),
	)

	// Wire billing plan cache and lock.
	billing_service.SetPlanCache(
		SyncFactory.NewCache(),
		synch.NewLock[string](SyncFactory),
	)

	// Wire endpoint weight cache and lock.
	billing_service.SetEndpointWeightCache(
		SyncFactory.NewCache(),
		synch.NewLock[string](SyncFactory),
	)

	pterm.DefaultLogger.Info("Sync primitives wired successfully")
}
