// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_roundTicker_contextCancellationStopsTicker(t *testing.T) {
	t.Parallel()
	roundDuration := 100 * time.Millisecond
	info := RoundTimingInfo{
		Offset:         time.Now(),
		Round:          roundDuration,
		AuctionClosing: 30 * time.Millisecond,
	}
	ticker := newRoundTicker(info)
	ctx, cancel := context.WithCancel(context.Background())
	go ticker.tickAtAuctionClose(ctx)

	// Receive at least one tick to confirm the ticker is running.
	select {
	case <-ticker.c:
	case <-time.After(2 * time.Second):
		t.Fatal("expected at least one tick before cancel")
	}

	cancel()

	// After cancellation, no further ticks should arrive.
	select {
	case _, ok := <-ticker.c:
		// A buffered tick that was already in-flight is acceptable,
		// but the channel must NOT be closed (ok should be true if we get a value).
		if !ok {
			t.Fatal("ticker channel was closed; it should remain open after context cancellation")
		}
	case <-time.After(3 * roundDuration):
		// No tick arrived — expected.
	}

	// Verify the channel is still open (not closed) by attempting a non-blocking send.
	// If close(t.c) had been called, this would panic.
	select {
	case ticker.c <- time.Time{}:
		// drain it back out
		<-ticker.c
	default:
		// channel is full (buffered 1) — that's fine, still proves it's not closed
	}
}

func Test_durationIntoRound_subSecondPrecision(t *testing.T) {
	t.Parallel()
	offset := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	info := RoundTimingInfo{
		Offset:         offset,
		Round:          1500 * time.Millisecond,
		AuctionClosing: 500 * time.Millisecond,
	}

	// At 0ms into a round, should return 0
	require.Equal(t, time.Duration(0), info.durationIntoRound(offset))

	// At 750ms into a round
	require.Equal(t, 750*time.Millisecond, info.durationIntoRound(offset.Add(750*time.Millisecond)))

	// At 1500ms (exactly one round), should wrap to 0
	require.Equal(t, time.Duration(0), info.durationIntoRound(offset.Add(1500*time.Millisecond)))

	// At 1750ms (250ms into second round)
	require.Equal(t, 250*time.Millisecond, info.durationIntoRound(offset.Add(1750*time.Millisecond)))

	// Timestamp before the offset should return 0 (negative guard)
	require.Equal(t, time.Duration(0), info.durationIntoRound(offset.Add(-1*time.Second)))

	// Auction should be closed at 1000ms into the round (>= 1500-500)
	require.True(t, info.isAuctionRoundClosedAt(offset.Add(1000*time.Millisecond)))

	// Auction should be open at 999ms into the round (< 1500-500)
	require.False(t, info.isAuctionRoundClosedAt(offset.Add(999*time.Millisecond)))
}

func Test_auctionClosed(t *testing.T) {
	t.Parallel()
	roundTimingInfo := RoundTimingInfo{
		Offset:         time.Now(),
		Round:          time.Minute,
		AuctionClosing: time.Second * 15,
	}

	initialTimestamp := time.Now()

	// We should not have closed the round yet, and the time into the round should be less than a second.
	isClosed := roundTimingInfo.isAuctionRoundClosedAt(initialTimestamp)
	require.False(t, isClosed)

	// Wait right before auction closure (before the 45 second mark).
	timestamp := initialTimestamp.Add((roundTimingInfo.Round - roundTimingInfo.AuctionClosing) - time.Second)
	isClosed = roundTimingInfo.isAuctionRoundClosedAt(timestamp)
	require.False(t, isClosed)

	// Wait a second more and the auction should be closed.
	timestamp = initialTimestamp.Add(roundTimingInfo.Round - roundTimingInfo.AuctionClosing)
	isClosed = roundTimingInfo.isAuctionRoundClosedAt(timestamp)
	require.True(t, isClosed)

	// Future timestamp should also be closed, until we reach the new round
	for i := float64(0); i < roundTimingInfo.AuctionClosing.Seconds(); i++ {
		timestamp = initialTimestamp.Add((roundTimingInfo.Round - roundTimingInfo.AuctionClosing) + time.Second*time.Duration(i))
		isClosed = roundTimingInfo.isAuctionRoundClosedAt(timestamp)
		require.True(t, isClosed)
	}
	isClosed = roundTimingInfo.isAuctionRoundClosedAt(initialTimestamp.Add(roundTimingInfo.Round))
	require.False(t, isClosed)
}
