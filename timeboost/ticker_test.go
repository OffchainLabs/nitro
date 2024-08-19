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
	nextMinute := now.Add(waitTime)
	<-time.After(waitTime)

	timeIntoRound, isClosed := auctionClosed(nextMinute, roundDuration, auctionClosingDuration)

	// We should not have closed the round yet, and the time into the round should be less than a second.
	require.False(t, isClosed)
	require.True(t, timeIntoRound < time.Second)

	// Wait right before auction closure (before the 45 second mark).
	now = time.Now()
	waitTime = (roundDuration - auctionClosingDuration) - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
	secondBeforeClosing := waitTime - time.Second
	<-time.After(secondBeforeClosing)

	timeIntoRound, isClosed = auctionClosed(nextMinute, roundDuration, auctionClosingDuration)
	require.False(t, isClosed)
	require.True(t, timeIntoRound < (roundDuration-auctionClosingDuration))

	// Wait a second more and the auction should be closed.
	<-time.After(time.Second)
	timeIntoRound, isClosed = auctionClosed(nextMinute, roundDuration, auctionClosingDuration)
	require.True(t, isClosed)
	require.True(t, timeIntoRound >= (roundDuration-auctionClosingDuration))
}
