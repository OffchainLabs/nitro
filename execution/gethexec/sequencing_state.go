// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import "time"

type sequencingTurn int

const (
	noSequencingTurn sequencingTurn = iota
	regularSequencingTurn
	delayedSequencingTurn
)

// sequencingState lets delayed messages drain unthrottled while limiting regular
// tx block creation to at most one per MaxBlockSpeed, preventing either from
// starving the other.
type sequencingState struct {
	lastTurn sequencingTurn
	// regularSequencingThrottledUntil is the earliest time a regular tx block
	// may be created again. It is set MaxBlockSpeed into the future after a block
	// is made (rate-limiting block production) or on a transient error. When a
	// regular turn instead finds the queue empty, it is set only the idle poll
	// interval into the future, matching the wait decideSequencingTurn uses when
	// there is no pending work.
	regularSequencingThrottledUntil time.Time
	// delayedSequencingThrottledUntil is the earliest time a delayed message
	// may be sequenced again. It is set only when a delayed turn fails to produce
	// a block (the head message is halted on a filtered tx), so a stuck delayed
	// message is retried at most once per MaxBlockSpeed instead of every iteration.
	delayedSequencingThrottledUntil time.Time
}

func decideSequencingTurn(state sequencingState, hasRegular, hasDelayed bool, now time.Time, pollInterval time.Duration) (sequencingTurn, time.Duration) {
	canSequenceRegular := hasRegular && !now.Before(state.regularSequencingThrottledUntil)
	canSequenceDelayed := hasDelayed && !now.Before(state.delayedSequencingThrottledUntil)
	switch {
	case canSequenceRegular && canSequenceDelayed:
		// Both runnable: alternate so neither starves the other.
		if state.lastTurn == regularSequencingTurn {
			return delayedSequencingTurn, 0
		}
		return regularSequencingTurn, 0
	case canSequenceRegular:
		return regularSequencingTurn, 0
	case canSequenceDelayed:
		return delayedSequencingTurn, 0
	case !hasRegular && !hasDelayed:
		return noSequencingTurn, pollInterval
	default:
		// Work is pending but throttled; wait for the soonest throttle to lift.
		return noSequencingTurn, state.soonestThrottleWait(hasRegular, hasDelayed, now)
	}
}

func (state sequencingState) soonestThrottleWait(hasRegular, hasDelayed bool, now time.Time) time.Duration {
	var wakeAt time.Time
	if hasRegular {
		wakeAt = state.regularSequencingThrottledUntil
	}
	if hasDelayed && (wakeAt.IsZero() || state.delayedSequencingThrottledUntil.Before(wakeAt)) {
		wakeAt = state.delayedSequencingThrottledUntil
	}
	return wakeAt.Sub(now)
}

func sequencingStateAfterRegularSequencing(state sequencingState, throttleRegularSequencingFor time.Duration, now time.Time) sequencingState {
	state.lastTurn = regularSequencingTurn
	if throttleRegularSequencingFor > 0 {
		state.regularSequencingThrottledUntil = now.Add(throttleRegularSequencingFor)
	} else {
		// Retry promptly (e.g. items were present but no block was produced
		// because all txs failed this round).
		state.regularSequencingThrottledUntil = time.Time{}
	}
	return state
}

func sequencingStateAfterDelayedSequencing(state sequencingState, producedMsg bool, now time.Time, maxBlockSpeed time.Duration) sequencingState {
	state.lastTurn = delayedSequencingTurn
	if !producedMsg {
		// The head delayed message is present but not sequenceable (halted on a
		// filtered tx); retry it at most once per MaxBlockSpeed.
		state.delayedSequencingThrottledUntil = now.Add(maxBlockSpeed)
	}
	return state
}
