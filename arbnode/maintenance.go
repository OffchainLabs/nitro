// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MaintenanceRunner struct {
	stopwaiter.StopWaiter

	exec            execution.ExecutionClient
	config          MaintenanceConfigFetcher
	seqCoordinator  *SeqCoordinator
	lastMaintenance atomic.Int64

	// lock is used to ensures that at any given time, only single node is on
	// maintenance mode.
	lock *redislock.Simple
}

type MaintenanceConfig struct {
	TimeOfDay   string              `koanf:"time-of-day" reload:"hot"`
	Lock        redislock.SimpleCfg `koanf:"lock" reload:"hot"`
	Triggerable bool                `koanf:"triggerable" reload:"hot"`

	// Generated: the minutes since start of UTC day to run maintenance at
	minutesAfterMidnight int
	enabled              bool
}

// Returns true if successful
func (c *MaintenanceConfig) parseMaintenanceRunTime() bool {
	if c.TimeOfDay == "" {
		return true
	}
	parts := strings.Split(c.TimeOfDay, ":")
	if len(parts) != 2 {
		return false
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours >= 24 {
		return false
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes >= 60 {
		return false
	}
	c.enabled = true
	c.minutesAfterMidnight = hours*60 + minutes
	return true
}

func (c *MaintenanceConfig) Validate() error {
	if !c.parseMaintenanceRunTime() {
		return fmt.Errorf("expected maintenance run time to be in 24-hour HH:MM format but got \"%v\"", c.TimeOfDay)
	}
	return nil
}

func MaintenanceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".time-of-day", DefaultMaintenanceConfig.TimeOfDay, "UTC 24-hour time of day to run maintenance at (e.g. 15:00)")
	f.Bool(prefix+".triggerable", DefaultMaintenanceConfig.Triggerable, "maintenance is triggerable via rpc")
	redislock.AddConfigOptions(prefix+".lock", f)
}

var DefaultMaintenanceConfig = MaintenanceConfig{
	TimeOfDay:   "",
	Lock:        redislock.DefaultCfg,
	Triggerable: false,

	minutesAfterMidnight: 0,
}

type MaintenanceConfigFetcher func() *MaintenanceConfig

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, exec execution.ExecutionClient) (*MaintenanceRunner, error) {
	cfg := config()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	res := &MaintenanceRunner{
		exec:           exec,
		config:         config,
		seqCoordinator: seqCoordinator,
	}

	// node restart is considered "maintenance"
	res.lastMaintenance.Store(time.Now().UnixMilli())
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
	mr.CallIteratively(mr.maybeRunScheduledMaintenance)
}

func wentPastTimeOfDay(before time.Time, after time.Time, timeOfDay int) bool {
	if !after.After(before) {
		return false
	}
	if after.Sub(before) >= time.Hour*24 {
		return true
	}
	prevMinutes := before.Hour()*60 + before.Minute()
	newMinutes := after.Hour()*60 + after.Minute()
	if newMinutes < prevMinutes {
		newMinutes += 60 * 24
	}
	maintenanceRunTimeMinutes := timeOfDay
	if maintenanceRunTimeMinutes < prevMinutes {
		maintenanceRunTimeMinutes += 60 * 24
	}
	return prevMinutes < maintenanceRunTimeMinutes && newMinutes >= maintenanceRunTimeMinutes
}

// bool if running currently, if false - time of last time it was running
func (mr *MaintenanceRunner) getPrevMaintenance() (bool, time.Time) {
	milli := mr.lastMaintenance.Load()
	if milli == 0 {
		return true, time.Time{}
	}
	return false, time.UnixMilli(milli)
}

// bool if running currently, if false - duration since last time it was running
func (mr *MaintenanceRunner) TimeSinceLastMaintenance() (bool, time.Duration) {
	running, maintTime := mr.getPrevMaintenance()
	if running {
		return true, 0
	}
	return false, time.Since(maintTime)
}

func (mr *MaintenanceRunner) setMaintenanceDone() {
	milli := time.Now().UnixMilli()
	prev := mr.lastMaintenance.Swap(milli)
	if prev != 0 {
		log.Error("maintenance executed in parallel", "current", time.UnixMilli(milli), "prev", time.UnixMilli(prev))
	}
}

func (mr *MaintenanceRunner) setMaintenanceStart() error {
	prev := mr.lastMaintenance.Swap(0)
	if prev == 0 {
		return errors.New("already running")
	}
	return nil
}

func (mr *MaintenanceRunner) maybeRunScheduledMaintenance(ctx context.Context) time.Duration {
	config := mr.config()
	if !config.enabled {
		return time.Minute
	}

	now := time.Now().UTC()

	inMaintenance, lastMaintenance := mr.getPrevMaintenance()
	if inMaintenance {
		return time.Minute
	}

	if !wentPastTimeOfDay(lastMaintenance, now, config.minutesAfterMidnight) {
		return time.Minute
	}

	shouldTriggerMaintenance, err := mr.exec.ShouldTriggerMaintenance().Await(mr.GetContext())
	if err != nil {
		log.Error("error checking if maintenance should be triggered", "err", err)
		return time.Minute
	}
	if !shouldTriggerMaintenance {
		log.Debug("skipping maintenance, not triggered")
		return time.Minute
	}

	err = mr.attemptMaintenance(ctx)
	if err != nil {
		log.Error("scheduled maintenance error", "err", err)
	}

	return time.Minute
}

func (mr *MaintenanceRunner) Trigger() error {
	if !mr.config().Triggerable {
		return errors.New("maintenance not configured to be triggerable")
	}
	if running, _ := mr.getPrevMaintenance(); running {
		return nil
	}
	// maintenance takes a long time, run on a separate thread
	mr.LaunchThread(func(ctx context.Context) {
		err := mr.attemptMaintenance(ctx)
		if err != nil {
			log.Warn("triggered maintenance returned error", "err", err)
		}
	})
	return nil
}

func (mr *MaintenanceRunner) attemptMaintenance(ctx context.Context) error {
	if mr.seqCoordinator == nil {
		return mr.runMaintenance()
	}

	release := make(chan struct{})
	if !mr.lock.AttemptLockAndPeriodicallyRefreshIt(ctx, release) {
		return errors.New("did not catch maintenance lock")
	}
	defer func() {
		release <- struct{}{}
	}()

	res := errors.New("maintenance failed to hand-off chosen one")

	log.Info("Attempting avoiding lockout and handing off", "targetTime", mr.config().TimeOfDay)
	// Avoid lockout for the sequencer and try to handoff.
	if mr.seqCoordinator.AvoidLockout(ctx) && mr.seqCoordinator.TryToHandoffChosenOne(ctx) {
		res = mr.runMaintenance()
	}
	defer mr.seqCoordinator.SeekLockout(ctx) // needs called even if c.Zombify returns false
	return res
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

func (mr *MaintenanceRunner) runMaintenance() error {
	err := mr.setMaintenanceStart()
	if err != nil {
		return err
	}
	defer mr.setMaintenanceDone()

	log.Info("Triggering maintenance")
	_, err = mr.exec.TriggerMaintenance().Await(mr.GetContext())
	if err != nil {
		return err
	}

	mr.waitMaintenanceToComplete(mr.GetContext())
	return nil
}
