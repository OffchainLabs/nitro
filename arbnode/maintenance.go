// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"time"

	flag "github.com/spf13/pflag"

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
	RunInterval time.Duration       `koanf:"run-interval" reload:"hot"`
	Lock        redislock.SimpleCfg `koanf:"lock" reload:"hot"`
}

func MaintenanceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".run-interval", DefaultMaintenanceConfig.RunInterval, "how often to run maintenance")
	redislock.AddConfigOptions(prefix+".lock", f)
}

var DefaultMaintenanceConfig = MaintenanceConfig{
	RunInterval: time.Minute,
	Lock:        redislock.DefaultCfg,
}

type MaintenanceConfigFetcher func() *MaintenanceConfig

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, exec execution.ExecutionClient) *MaintenanceRunner {
	return &MaintenanceRunner{
		exec:           exec,
		config:         config,
		seqCoordinator: seqCoordinator,
	}
}

func (mr *MaintenanceRunner) Start(ctxIn context.Context) {
	mr.StopWaiter.Start(ctxIn, mr)
	mr.CallIteratively(mr.maybeRunMaintenance)
}

func (mr *MaintenanceRunner) maybeRunMaintenance(ctx context.Context) time.Duration {
	config := mr.config()

	shouldTriggerMaintenance, err := mr.exec.ShouldTriggerMaintenance().Await(mr.GetContext())
	if err != nil {
		log.Error("error checking if maintenance should be triggered", "err", err)
		return config.RunInterval
	}
	if !shouldTriggerMaintenance {
		log.Debug("skipping maintenance, not triggered")
		return config.RunInterval
	}

	// If seqCoordinator is nil there is no need to coordinate maintenance running with other sequecers.
	if mr.seqCoordinator == nil {
		mr.runMaintenance()
		return config.RunInterval
	}

	release := make(chan struct{})
	if !mr.lock.AttemptLockAndPeriodicallyRefreshIt(ctx, release) {
		log.Warn("maintenance lock not acquired, skipping maintenance")
		return config.RunInterval
	}
	defer func() {
		release <- struct{}{}
	}()

	log.Info("Attempting avoiding lockout and handing off")
	if mr.seqCoordinator.AvoidLockout(ctx) && mr.seqCoordinator.TryToHandoffChosenOne(ctx) {
		mr.runMaintenance()
	} else {
		log.Error("maintenance failed to hand-off chosen one")
	}
	defer mr.seqCoordinator.SeekLockout(ctx)

	return config.RunInterval
}

func (mr *MaintenanceRunner) waitMaintenanceToComplete(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
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
