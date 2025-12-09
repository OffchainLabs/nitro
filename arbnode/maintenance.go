// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MaintenanceRunner struct {
	stopwaiter.StopWaiter

	exec           execution.ExecutionClient
	config         MaintenanceConfigFetcher
	seqCoordinator *SeqCoordinator

	// lock is used to ensure that at any given time, only single node is on
	// maintenance mode.
	lock *redislock.Simple
}

type MaintenanceConfig struct {
	Enable        bool                `koanf:"enable" reload:"hot"`
	CheckInterval time.Duration       `koanf:"check-interval" reload:"hot"`
	Lock          redislock.SimpleCfg `koanf:"lock" reload:"hot"`
}

func MaintenanceConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMaintenanceConfig.Enable, "enable maintenance runner")
	f.Duration(prefix+".check-interval", DefaultMaintenanceConfig.CheckInterval, "how often to check if maintenance should be run")
	redislock.AddConfigOptions(prefix+".lock", f)
}

var DefaultMaintenanceConfig = MaintenanceConfig{
	Enable:        false,
	CheckInterval: time.Minute,
	Lock:          redislock.DefaultCfg,
}

type MaintenanceConfigFetcher func() *MaintenanceConfig

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, exec execution.ExecutionClient) (*MaintenanceRunner, error) {
	cfg := config()

	res := &MaintenanceRunner{
		exec:           exec,
		config:         config,
		seqCoordinator: seqCoordinator,
	}

	if seqCoordinator != nil {
		c := func() *redislock.SimpleCfg { return &cfg.Lock }
		r := func() bool { return true } // always ready to lock
		rl, err := redislock.NewSimple(seqCoordinator.RedisCoordinator().Client, c, r)
		if err != nil {
			return nil, fmt.Errorf("creating new simple redis lock: %w", err)
		}
		res.lock = rl
	}
	return res, nil
}

func (mr *MaintenanceRunner) Start(ctxIn context.Context) {
	mr.StopWaiter.Start(ctxIn, mr)
	mr.CallIteratively(mr.MaybeRunMaintenance)
}

// exported for testing
func (mr *MaintenanceRunner) MaybeRunMaintenance(ctx context.Context) time.Duration {
	config := mr.config()
	if !config.Enable {
		log.Debug("maintenance is disabled, skipping")
		return config.CheckInterval
	}

	shouldTriggerMaintenance, err := mr.exec.ShouldTriggerMaintenance().Await(mr.GetContext())
	if err != nil {
		log.Error("error checking if maintenance should be triggered", "err", err)
		return config.CheckInterval
	}
	if !shouldTriggerMaintenance {
		log.Debug("skipping maintenance, not triggered")
		return config.CheckInterval
	}

	// If seqCoordinator is nil there is no need to coordinate maintenance running with other sequecers.
	if mr.seqCoordinator == nil {
		mr.runMaintenance()
		return config.CheckInterval
	}

	release := make(chan struct{})
	if !mr.lock.AttemptLockAndPeriodicallyRefreshIt(ctx, release) {
		log.Warn("maintenance lock not acquired, skipping maintenance")
		return config.CheckInterval
	}
	defer func() {
		close(release)
	}()

	log.Info("Attempting avoiding lockout and handing off")
	if mr.seqCoordinator.AvoidLockout(ctx) && mr.seqCoordinator.TryToHandoffChosenOne(ctx) {
		log.Info("Avoided lockout and handed off chosen one")
		mr.runMaintenance()
	} else {
		log.Error("maintenance failed to hand-off chosen one")
	}
	mr.seqCoordinator.SeekLockout(ctx)

	return config.CheckInterval
}

func (mr *MaintenanceRunner) waitMaintenanceToComplete(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Warn("Maintenance wait interrupted", "err", ctx.Err())
			return
		default:
			select {
			case <-ctx.Done():
				log.Warn("Maintenance wait interrupted", "err", ctx.Err())
				return
			case <-ticker.C:
				maintenanceStatus, err := mr.exec.MaintenanceStatus().Await(ctx)
				if err != nil {
					log.Error("Error checking maintenance status", "err", err)
					continue
				}
				if maintenanceStatus.IsRunning {
					log.Debug("Maintenance is still running, waiting for completion")
				} else {
					log.Info("Execution is not running maintenance anymore, maintenance completed successfully")
					return
				}
			}
		}
	}
}

func (mr *MaintenanceRunner) runMaintenance() {
	log.Info("Triggering maintenance")
	_, err := mr.exec.TriggerMaintenance().Await(mr.GetContext())
	if err != nil {
		log.Error("Error triggering maintenance", "err", err)
		return
	}
	mr.waitMaintenanceToComplete(mr.GetContext())
}
