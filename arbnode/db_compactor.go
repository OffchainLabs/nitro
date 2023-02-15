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
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

// Regularly runs db compaction on sequencers avoiding downtime
type DbCompactor struct {
	stopwaiter.StopWaiter

	config         DbCompactorConfigFetcher
	seqCoordinator *SeqCoordinator
	dbs            []ethdb.Database
	lastCheck      time.Time
}

type DbCompactorConfig struct {
	TimeOfDay string `koanf:"time-of-day" reload:"hot"`

	// Generated: the minutes since start of UTC day to compact at
	minutesAfterMidnight int
}

// Returns true if successful
func (c *DbCompactorConfig) parseDbCompactionTime() bool {
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
	c.minutesAfterMidnight = hours*60 + minutes
	return true
}

func (c *DbCompactorConfig) Validate() error {
	if !c.parseDbCompactionTime() {
		return fmt.Errorf("expected sequencer coordinator db compaction time to be in 24-hour HH:MM format but got \"%v\"", c.TimeOfDay)
	}
	return nil
}

func DbCompactorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".time-of-day", DefaultDbCompactorConfig.TimeOfDay, "UTC 24-hour time of day to run database compaction at (e.g. 15:00)")
}

var DefaultDbCompactorConfig = DbCompactorConfig{
	TimeOfDay: "",

	minutesAfterMidnight: 0,
}

type DbCompactorConfigFetcher func() *DbCompactorConfig

func NewDbCompactor(config DbCompactorConfigFetcher, seqCoordinator *SeqCoordinator, dbs []ethdb.Database) *DbCompactor {
	return &DbCompactor{
		config:         config,
		seqCoordinator: seqCoordinator,
		dbs:            dbs,
		lastCheck:      time.Now().UTC(),
	}
}

func (c *DbCompactor) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn, c)
	c.CallIteratively(c.maybeCompactDb)
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

func (c *DbCompactor) maybeCompactDb(ctx context.Context) time.Duration {
	config := c.config()
	if config.TimeOfDay == "" {
		return time.Minute
	}
	now := time.Now().UTC()
	if wentPastTimeOfDay(c.lastCheck, now, config.minutesAfterMidnight) {
		log.Info("attempting to release sequencer lockout to run database compaction", "targetTime", config.TimeOfDay)
		success := c.seqCoordinator.Zombify(ctx)
		defer c.seqCoordinator.Unzombify(ctx) // needs called even if c.Zombify returns false
		if success {
			// We've released liveliness, now wait for the handoff
			success = c.seqCoordinator.TryToHandoff(ctx)
			if success {
				c.compactDb()
			}
		}
	}
	c.lastCheck = now
	return time.Minute
}

func (c *DbCompactor) compactDb() {
	log.Info("compacting databases (this may take a while...)")
	results := make(chan error, len(c.dbs))
	for _, db := range c.dbs {
		db := db
		go func() {
			results <- db.Compact(nil, nil)
		}()
	}
	for range c.dbs {
		err := <-results
		if err != nil {
			log.Warn("failed to compact database", "err", err)
		}
	}
	log.Info("done compacting databases")
}
