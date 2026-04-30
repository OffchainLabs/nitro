// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"context"
	"time"
)

type roundTicker struct {
	c               chan time.Time
	roundTimingInfo RoundTimingInfo
}

func newRoundTicker(roundTimingInfo RoundTimingInfo) *roundTicker {
	return &roundTicker{
		c:               make(chan time.Time, 1),
		roundTimingInfo: roundTimingInfo,
	}
}

func (t *roundTicker) tickAtAuctionClose(ctx context.Context) {
	t.start(ctx, t.roundTimingInfo.AuctionClosing)
}

func (t *roundTicker) tickAtReserveSubmissionDeadline(ctx context.Context) {
	t.start(ctx, t.roundTimingInfo.AuctionClosing+t.roundTimingInfo.ReserveSubmission)
}

// start ticks t.c at the specified offset before each round start. The channel is
// intentionally not closed on shutdown to avoid send-on-closed-channel panics.
func (t *roundTicker) start(ctx context.Context, timeBeforeRoundStart time.Duration) {
	for {
		nextTick := t.roundTimingInfo.TimeTilNextRound() - timeBeforeRoundStart
		if nextTick < 0 {
			nextTick += t.roundTimingInfo.Round
		}

		timer := time.NewTimer(nextTick)
		select {
		case <-timer.C:
			select {
			case t.c <- time.Now():
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}
