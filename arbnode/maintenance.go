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
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

// Regularly runs db compaction if configured
type MaintenanceRunner struct {
	stopwaiter.StopWaiter

	exec           execution.FullExecutionClient
	config         MaintenanceConfigFetcher
	seqCoordinator *SeqCoordinator
	dbs            []ethdb.Database
	lastCheck      time.Time
}

type MaintenanceConfig struct {
	TimeOfDay string `koanf:"time-of-day" reload:"hot"`

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
}

var DefaultMaintenanceConfig = MaintenanceConfig{
	TimeOfDay: "",

	minutesAfterMidnight: 0,
}

type MaintenanceConfigFetcher func() *MaintenanceConfig

func NewMaintenanceRunner(config MaintenanceConfigFetcher, seqCoordinator *SeqCoordinator, dbs []ethdb.Database, exec execution.FullExecutionClient) (*MaintenanceRunner, error) {
	err := config().Validate()
	if err != nil {
		return nil, err
	}
	return &MaintenanceRunner{
		config:         config,
		exec:           exec,
		seqCoordinator: seqCoordinator,
		dbs:            dbs,
		lastCheck:      time.Now().UTC(),
	}, nil
}

func (c *MaintenanceRunner) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn, c)
	c.CallIteratively(c.maybeRunMaintenance)
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

func (c *MaintenanceRunner) maybeRunMaintenance(ctx context.Context) time.Duration {
	config := c.config()
	if !config.enabled {
		return time.Minute
	}
	now := time.Now().UTC()
	if wentPastTimeOfDay(c.lastCheck, now, config.minutesAfterMidnight) {
		log.Info("attempting to release sequencer lockout to run database compaction", "targetTime", config.TimeOfDay)
		if c.seqCoordinator == nil {
			c.runMaintenance()
		} else {
			// We want to switch sequencers before running maintenance
			success := c.seqCoordinator.AvoidLockout(ctx)
			defer c.seqCoordinator.SeekLockout(ctx) // needs called even if c.Zombify returns false
			if success {
				// We've unset the wants lockout key, now wait for the handoff
				success = c.seqCoordinator.TryToHandoffChosenOne(ctx)
				if success {
					c.runMaintenance()
				}
			}
		}
	}
	c.lastCheck = now
	return time.Minute
}

func (c *MaintenanceRunner) runMaintenance() {
	log.Info("compacting databases (this may take a while...)")
	results := make(chan error, len(c.dbs))
	expected := 0
	for _, db := range c.dbs {
		expected++
		db := db
		go func() {
			results <- db.Compact(nil, nil)
		}()
	}
	expected++
	go func() {
		results <- c.exec.Maintenance()
	}()
	for i := 0; i < expected; i++ {
		err := <-results
		if err != nil {
			log.Warn("maintenance error", "err", err)
		}
	}
	log.Info("done compacting databases")
}
