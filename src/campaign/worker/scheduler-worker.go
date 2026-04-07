package campaign_worker

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	synch "github.com/Astervia/wacraft-core/src/synch"
	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	campaign_service "github.com/Astervia/wacraft-server/src/campaign/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"golang.org/x/sync/errgroup"
)

const (
	// SchedulerPoolSize is the max number of campaigns processed concurrently.
	SchedulerPoolSize = 5
	// SchedulerBatchSize is the max number of campaigns fetched per poll.
	SchedulerBatchSize = 10
)

// schedulerPollInterval returns the poll interval from CAMPAIGN_SCHEDULE_POLL_INTERVAL
// env var, defaulting to 30 seconds.
func schedulerPollInterval() time.Duration {
	if val := os.Getenv("CAMPAIGN_SCHEDULE_POLL_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return 30 * time.Second
}

// SchedulerWorker polls the database for scheduled campaigns and executes them.
type SchedulerWorker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// lock prevents duplicate execution across instances (nil in memory mode).
	lock synch_contract.DistributedLock[string]
	// syncFactory is used to create distributed CampaignChannel primitives (nil in memory mode).
	syncFactory *synch.Factory
}

// Package-level lock and factory, set from src/synch/main.go when SYNC_BACKEND=redis.
var (
	schedulerLock    synch_contract.DistributedLock[string]
	schedulerFactory *synch.Factory
)

// poolGetOrCreate and poolRelease are injected from serve.go to avoid an
// import cycle between campaign/worker and campaign/handler.
var (
	poolGetOrCreate func(uuid.UUID) *campaign_model.CampaignChannel
	poolRelease     func(uuid.UUID)
)

// SetChannelPool wires the global campaign channel pool functions used by the
// scheduler to obtain and release a CampaignChannel for each execution.
// Call this from serve.go after both packages have been initialised.
func SetChannelPool(
	getOrCreate func(uuid.UUID) *campaign_model.CampaignChannel,
	release func(uuid.UUID),
) {
	poolGetOrCreate = getOrCreate
	poolRelease = release
}

// SetSchedulerLock sets the distributed lock used by new SchedulerWorker instances.
// Called from src/synch/main.go when SYNC_BACKEND=redis.
func SetSchedulerLock(l synch_contract.DistributedLock[string]) {
	schedulerLock = l
}

// SetSchedulerFactory provides the sync factory so the worker can create
// distributed CampaignChannels when SYNC_BACKEND=redis.
func SetSchedulerFactory(f *synch.Factory) {
	schedulerFactory = f
}

// NewSchedulerWorker creates a new SchedulerWorker, picking up the package-level
// lock and factory set during initialisation.
func NewSchedulerWorker() *SchedulerWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SchedulerWorker{
		ctx:         ctx,
		cancel:      cancel,
		lock:        schedulerLock,
		syncFactory: schedulerFactory,
	}
}

// Start resets any campaigns stuck in "running" state (restart recovery) and
// begins the polling loop.
func (w *SchedulerWorker) Start() {
	w.recoverRunningCampaigns()

	w.wg.Add(1)
	go w.run()
	pterm.DefaultLogger.Info("Campaign scheduler worker started")
}

// Stop gracefully stops the scheduler worker.
func (w *SchedulerWorker) Stop() {
	pterm.DefaultLogger.Info("Stopping campaign scheduler worker...")
	w.cancel()
	w.wg.Wait()
	pterm.DefaultLogger.Info("Campaign scheduler worker stopped")
}

// recoverRunningCampaigns resets campaigns left in "running" state by a crashed
// instance back to "scheduled" so they are re-picked up.
func (w *SchedulerWorker) recoverRunningCampaigns() {
	result := database.DB.Model(&campaign_entity.Campaign{}).
		Where("status = ?", "running").
		Update("status", "scheduled")
	if result.Error != nil {
		pterm.DefaultLogger.Error("Campaign scheduler: failed to recover running campaigns: " + result.Error.Error())
		return
	}
	if result.RowsAffected > 0 {
		pterm.DefaultLogger.Warn(
			"Campaign scheduler: reset " + itoa(result.RowsAffected) + " running campaign(s) to scheduled (restart recovery)",
		)
	}
}

// run is the main polling loop.
func (w *SchedulerWorker) run() {
	defer w.wg.Done()

	interval := schedulerPollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processDueCampaigns()
		}
	}
}

// processDueCampaigns fetches campaigns that are due and processes them.
func (w *SchedulerWorker) processDueCampaigns() {
	var campaigns []campaign_entity.Campaign
	if err := database.DB.
		Where("status = ? AND scheduled_at <= ?", "scheduled", time.Now().UTC()).
		Limit(SchedulerBatchSize).
		Find(&campaigns).Error; err != nil {
		pterm.DefaultLogger.Error("Campaign scheduler: failed to fetch due campaigns: " + err.Error())
		return
	}

	if len(campaigns) == 0 {
		return
	}

	g, ctx := errgroup.WithContext(w.ctx)
	g.SetLimit(SchedulerPoolSize)

	for i := range campaigns {
		c := &campaigns[i]
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				w.processCampaign(c)
				return nil
			}
		})
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		pterm.DefaultLogger.Error("Campaign scheduler: error processing campaigns: " + err.Error())
	}
}

// processCampaign executes a single scheduled campaign.
func (w *SchedulerWorker) processCampaign(campaign *campaign_entity.Campaign) {
	campaignID := campaign.ID

	// Acquire distributed lock (Redis mode) to prevent duplicate execution.
	lockKey := "campaign_schedule:" + campaignID.String()
	if w.lock != nil {
		acquired, err := w.lock.TryLock(lockKey)
		if err != nil {
			pterm.DefaultLogger.Error("Campaign scheduler: lock error for " + campaignID.String() + ": " + err.Error())
			return
		}
		if !acquired {
			return // Another instance is already processing this campaign.
		}
		defer w.lock.Unlock(lockKey) //nolint:errcheck
	}

	// Atomically transition status: scheduled → running.
	// If 0 rows affected, another instance already claimed it.
	result := database.DB.Model(&campaign_entity.Campaign{}).
		Where("id = ? AND status = ?", campaignID, "scheduled").
		Update("status", "running")
	if result.Error != nil {
		pterm.DefaultLogger.Error("Campaign scheduler: failed to mark campaign running: " + result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		return // Already claimed by another instance.
	}

	pterm.DefaultLogger.Info("Campaign scheduler: starting campaign " + campaignID.String())

	// Get/create a pool channel so WebSocket clients can subscribe and receive
	// real-time progress updates from this scheduler-triggered execution.
	var channel *campaign_model.CampaignChannel
	if poolGetOrCreate != nil {
		channel = poolGetOrCreate(campaignID)
		defer poolRelease(campaignID)
	} else {
		// Fallback for tests or misconfigured startup: headless channel without pool.
		ch := campaign_model.CreateCampaignChannel(nil)
		channel = ch
	}

	// Execute the campaign using the same service as the WebSocket handler.
	_, err := campaign_service.SendWhatsAppCampaign(
		campaignID,
		*channel,
		func(data *campaign_model.CampaignResults) {
			channel.BroadcastProgress(*data)
		},
	)

	if err != nil {
		pterm.DefaultLogger.Error("Campaign scheduler: campaign " + campaignID.String() + " failed: " + err.Error())
		updateCampaignStatus(campaignID, "failed")
		return
	}

	pterm.DefaultLogger.Info("Campaign scheduler: campaign " + campaignID.String() + " completed")
	updateCampaignStatus(campaignID, "completed")
}

func updateCampaignStatus(campaignID uuid.UUID, status string) {
	if err := database.DB.Model(&campaign_entity.Campaign{}).
		Where("id = ?", campaignID).
		Update("status", status).Error; err != nil {
		pterm.DefaultLogger.Error("Campaign scheduler: failed to set status=" + status + " for " + campaignID.String() + ": " + err.Error())
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
