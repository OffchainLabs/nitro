package timeboost

import (
	"time"
)

type roundTicker struct {
	c               chan time.Time
	done            chan bool
	roundTimingInfo RoundTimingInfo
}

func newRoundTicker(roundTimingInfo RoundTimingInfo) *roundTicker {
	return &roundTicker{
		c:               make(chan time.Time, 1),
		done:            make(chan bool),
		roundTimingInfo: roundTimingInfo,
	}
}

func (t *roundTicker) tickAtAuctionClose() {
	t.start(t.roundTimingInfo.AuctionClosing)
}

func (t *roundTicker) tickAtReserveSubmissionDeadline() {
	t.start(t.roundTimingInfo.AuctionClosing + t.roundTimingInfo.ReserveSubmission)
}

func (t *roundTicker) start(timeBeforeRoundStart time.Duration) {
	for {
		nextTick := t.roundTimingInfo.TimeTilNextRound() - timeBeforeRoundStart
		if nextTick < 0 {
			nextTick += t.roundTimingInfo.Round
		}

		// Use NewTimer instead of time.After to allow cancellation and avoid leaking timers
		timer := time.NewTimer(nextTick)
		select {
		case <-timer.C:
			t.c <- time.Now()
		case <-t.done:
			if !timer.Stop() {
				<-timer.C
			}
			close(t.c)
			return
		}
	}
}
