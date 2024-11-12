package timeboost

import (
	"time"

	"github.com/offchainlabs/nitro/util/arbmath"
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
		// Calculate the start of the next round
		startOfNextMinute := now.Truncate(t.roundDuration).Add(t.roundDuration)
		// Subtract AUCTION_CLOSING_SECONDS seconds to get the tick time
		nextTickTime := startOfNextMinute.Add(-t.auctionClosingDuration)
		// Ensure we are not setting a past tick time
		if nextTickTime.Before(now) {
			// If the calculated tick time is in the past, move to the next interval
			nextTickTime = nextTickTime.Add(t.roundDuration)
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
	return arbmath.SaturatingUCast[uint64](time.Since(initialRoundTimestamp) / roundDuration)
}

func isAuctionRoundClosed(
	timestamp time.Time,
	initialTimestamp time.Time,
	roundDuration time.Duration,
	auctionClosingDuration time.Duration,
) bool {
	if timestamp.Before(initialTimestamp) {
		return false
	}
	timeInRound := timeIntoRound(timestamp, initialTimestamp, roundDuration)
	return arbmath.SaturatingCast[time.Duration](timeInRound)*time.Second >= roundDuration-auctionClosingDuration
}

func timeIntoRound(
	timestamp time.Time,
	initialTimestamp time.Time,
	roundDuration time.Duration,
) uint64 {
	secondsSinceOffset := uint64(timestamp.Sub(initialTimestamp).Seconds())
	roundDurationSeconds := uint64(roundDuration.Seconds())
	return secondsSinceOffset % roundDurationSeconds
}
