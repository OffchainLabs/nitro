package timeboost

import (
	"time"
)

type auctionCloseTicker struct {
	c                      chan time.Time
	done                   chan bool
	roundDuration          time.Duration
	auctionClosingDuration time.Duration
}

func newAuctionCloseTicker(roundDuration, auctionClosingDuration time.Duration) *auctionCloseTicker {
	return &auctionCloseTicker{
		c:                      make(chan time.Time, 1),
		done:                   make(chan bool),
		roundDuration:          roundDuration,
		auctionClosingDuration: auctionClosingDuration,
	}
}

func (t *auctionCloseTicker) start() {
	for {
		now := time.Now()
		// Calculate the start of the next minute
		startOfNextMinute := now.Truncate(time.Minute).Add(time.Minute)
		// Subtract 15 seconds to get the tick time
		nextTickTime := startOfNextMinute.Add(-15 * time.Second)
		// Ensure we are not setting a past tick time
		if nextTickTime.Before(now) {
			// If the calculated tick time is in the past, move to the next interval
			nextTickTime = nextTickTime.Add(time.Minute)
		}
		// Calculate how long to wait until the next tick
		waitTime := nextTickTime.Sub(now)

		select {
		case <-time.After(waitTime):
			t.c <- time.Now()
		case <-t.done:
			close(t.c)
			return
		}
	}
}

// CurrentRound returns the current round number.
func CurrentRound(initialRoundTimestamp time.Time, roundDuration time.Duration) uint64 {
	if roundDuration == 0 {
		return 0
	}
	return uint64(time.Since(initialRoundTimestamp) / roundDuration)
}

// auctionClosed returns the time into the current round and whether the auction for this round is closed.
func auctionClosed(initialRoundTimestamp time.Time, roundDuration time.Duration, auctionClosingDuration time.Duration) (time.Duration, bool) {
	if roundDuration == 0 {
		return 0, true
	}
	d := time.Since(initialRoundTimestamp) % roundDuration
	return d, d >= roundDuration-auctionClosingDuration
}
