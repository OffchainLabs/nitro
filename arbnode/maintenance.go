// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// Regularly runs db compaction if configured
type MaintenanceRunner struct {
	stopwaiter.StopWaiter

	exec            execution.FullExecutionClient
	config          MaintenanceConfigFetcher
	seqCoordinator  *SeqCoordinator
	dbs             []ethdb.Database
	lastMaintenance atomic.Int64

	// lock is used to ensures that at any given time, only single node is on
	// maintenance mode.
	lock *redislock.Simple
}

type MaintenanceConfig struct {
	TimeOfDay   string              `koanf:"time-of-day" reload:"hot"`
	Lock        redislock.SimpleCfg `koanf:"lock" reload:"hot"`
	Triggerable bool                `koanf:"triggerable" reload:"hot"`

	// Generated: the minutes since start of UTC day to compact at
	minutesAfterMidnight int
	enabled              bool
}

// Returns true if successful
func (c *MaintenanceConfig) parseDbCompactionTime() bool {
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
	if !c.parseDbCompactionTime() {
		return fmt.Errorf("expected sequencer coordinator db compaction time to be in 24-hour HH:MM format but got \"%v\"", c.TimeOfDay)
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

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, dbs []ethdb.Database, exec execution.FullExecutionClient) (*MaintenanceRunner, error) {
	cfg := config()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	res := &MaintenanceRunner{
		exec:           exec,
		config:         config,
		seqCoordinator: seqCoordinator,
		dbs:            dbs,
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
	dbCompactionMinutes := timeOfDay
	if dbCompactionMinutes < prevMinutes {
		dbCompactionMinutes += 60 * 24
	}
	return prevMinutes < dbCompactionMinutes && newMinutes >= dbCompactionMinutes
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

	err := mr.attemptMaintenance(ctx)
	if err != nil {
		log.Warn("scheduled maintenance error", "err", err)
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

	if !mr.lock.AttemptLock(ctx) {
		return errors.New("did not catch maintenance lock")
	}
	defer mr.lock.Release(ctx)

	res := errors.New("maintenance failed to hand-off chosen one")

	log.Info("Attempting avoiding lockout and handing off", "targetTime", mr.config().TimeOfDay)
	// Avoid lockout for the sequencer and try to handoff.
	if mr.seqCoordinator.AvoidLockout(ctx) && mr.seqCoordinator.TryToHandoffChosenOne(ctx) {
		res = mr.runMaintenance()
	}
	defer mr.seqCoordinator.SeekLockout(ctx) // needs called even if c.Zombify returns false
	return res
}

func (mr *MaintenanceRunner) runMaintenance() error {
	err := mr.setMaintenanceStart()
	if err != nil {
		return err
	}
	defer mr.setMaintenanceDone()

	log.Info("Compacting databases and flushing triedb to disk (this may take a while...)")
	results := make(chan error, len(mr.dbs))
	expected := 0
	for _, db := range mr.dbs {
		expected++
		db := db
		go func() {
			results <- db.Compact(nil, nil)
		}()
	}
	expected++
	go func() {
		results <- mr.exec.Maintenance()
	}()
	for i := 0; i < expected; i++ {
		subErr := <-results
		if subErr != nil {
			err = errors.Join(err, subErr)
			log.Warn("maintenance error", "err", subErr)
		}
	}
	log.Info("Done compacting databases and flushing triedb to disk")
	return err
}
