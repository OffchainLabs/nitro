// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

// Regularly runs db compaction if configured
type MaintenanceRunner struct {
	stopwaiter.StopWaiter

	config          MaintenanceConfigFetcher
	seqCoordinator  *SeqCoordinator
	dbs             []ethdb.Database
	lastMaintenance time.Time

	// lock is used to ensures that at any given time, only single node is on
	// maintenance mode.
	lock *redislock.Simple
}

type MaintenanceConfig struct {
	TimeOfDay string              `koanf:"time-of-day" reload:"hot"`
	Lock      redislock.SimpleCfg `koanf:"lock" reload:"hot"`

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
	f.String(prefix+".time-of-day", DefaultMaintenanceConfig.TimeOfDay, "UTC 24-hour time of day to run maintenance (currently only db compaction) at (e.g. 15:00)")
	redislock.AddConfigOptions(prefix+".lock", f)
}

var DefaultMaintenanceConfig = MaintenanceConfig{
	TimeOfDay: "",
	Lock:      redislock.DefaultCfg,

	minutesAfterMidnight: 0,
}

type MaintenanceConfigFetcher func() *MaintenanceConfig

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, dbs []ethdb.Database) (*MaintenanceRunner, error) {
	cfg := config()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	res := &MaintenanceRunner{
		config:          config,
		seqCoordinator:  seqCoordinator,
		dbs:             dbs,
		lastMaintenance: time.Now().UTC(),
	}

	if seqCoordinator != nil {
		c := func() *redislock.SimpleCfg { return &cfg.Lock }
		r := func() bool { return true } // always ready to lock
		rl, err := redislock.NewSimple(seqCoordinator.Client, c, r)
		if err != nil {
			return nil, fmt.Errorf("creating new simple redis lock: %w", err)
		}
		res.lock = rl
	}
	return res, nil
}

func (mr *MaintenanceRunner) Start(ctxIn context.Context) {
	mr.StopWaiter.Start(ctxIn, mr)
	mr.CallIteratively(mr.maybeRunMaintenance)
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

func (mr *MaintenanceRunner) maybeRunMaintenance(ctx context.Context) time.Duration {
	config := mr.config()
	if !config.enabled {
		return time.Minute
	}

	now := time.Now().UTC()

	if !wentPastTimeOfDay(mr.lastMaintenance, now, config.minutesAfterMidnight) {
		return time.Minute
	}

	if mr.seqCoordinator == nil {
		mr.lastMaintenance = now
		mr.runMaintenance()
		return time.Minute
	}

	if !mr.lock.AttemptLock(ctx) {
		return time.Minute
	}
	defer mr.lock.Release(ctx)

	log.Info("Attempting avoiding lockout and handing off", "targetTime", config.TimeOfDay)
	// Avoid lockout for the sequencer and try to handoff.
	if mr.seqCoordinator.AvoidLockout(ctx) && mr.seqCoordinator.TryToHandoffChosenOne(ctx) {
		mr.lastMaintenance = now
		mr.runMaintenance()
	}
	defer mr.seqCoordinator.SeekLockout(ctx) // needs called even if c.Zombify returns false

	return time.Minute
}

func (mr *MaintenanceRunner) runMaintenance() {
	log.Info("Compacting databases (this may take a while...)")
	results := make(chan error, len(mr.dbs))
	for _, db := range mr.dbs {
		db := db
		go func() {
			results <- db.Compact(nil, nil)
		}()
	}
	for range mr.dbs {
		if err := <-results; err != nil {
			log.Warn("Failed to compact database", "err", err)
		}
	}
	log.Info("Done compacting databases")
}
