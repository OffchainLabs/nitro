package timeboost

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_auctionClosed(t *testing.T) {
	t.Parallel()
	roundDuration := time.Minute
	auctionClosingDuration := time.Second * 15
	now := time.Now()
	waitTime := roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
	initialTimestamp := now.Add(waitTime)

	// We should not have closed the round yet, and the time into the round should be less than a second.
	isClosed := isAuctionRoundClosed(initialTimestamp, initialTimestamp, roundDuration, auctionClosingDuration)
	require.False(t, isClosed)

	// Wait right before auction closure (before the 45 second mark).
	timestamp := initialTimestamp.Add((roundDuration - auctionClosingDuration) - time.Second)
	isClosed = isAuctionRoundClosed(timestamp, initialTimestamp, roundDuration, auctionClosingDuration)
	require.False(t, isClosed)

	// Wait a second more and the auction should be closed.
	timestamp = initialTimestamp.Add(roundDuration - auctionClosingDuration)
	isClosed = isAuctionRoundClosed(timestamp, initialTimestamp, roundDuration, auctionClosingDuration)
	require.True(t, isClosed)

	// Future timestamp should also be closed, until we reach the new round
	for i := float64(0); i < auctionClosingDuration.Seconds(); i++ {
		timestamp = initialTimestamp.Add((roundDuration - auctionClosingDuration) + time.Second*time.Duration(i))
		isClosed = isAuctionRoundClosed(timestamp, initialTimestamp, roundDuration, auctionClosingDuration)
		require.True(t, isClosed)
	}
	isClosed = isAuctionRoundClosed(initialTimestamp.Add(roundDuration), initialTimestamp, roundDuration, auctionClosingDuration)
	require.False(t, isClosed)
}
