package timeboost

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
