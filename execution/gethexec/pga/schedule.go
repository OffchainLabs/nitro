// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"errors"
	"time"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// Schedule tracks which PGA round of the block is active and decides when each execute phase begins.
type Schedule struct {
	roundsPerBlock uint64
	roundLength    time.Duration
	activeRound    uint64
	deadline       time.Time
}

// NewSchedule returns a Schedule at round 1 of a block whose first execute phase begins now. It assumes roundsPerBlock
// is greater than 0; with roundsPerBlock of 0 every round reports as the last round.
func NewSchedule(roundsPerBlock uint64, roundLength time.Duration) *Schedule {
	return &Schedule{
		roundsPerBlock: roundsPerBlock,
		roundLength:    roundLength,
		activeRound:    1,
		deadline:       time.Now().Add(roundLength),
	}
}

func (s *Schedule) Round() uint64 {
	return s.activeRound
}

func (s *Schedule) IsLastRound() bool {
	return s.activeRound >= s.roundsPerBlock
}

func (s *Schedule) Deadline() time.Time {
	return s.deadline
}

func (s *Schedule) RoundIsOver() bool {
	return time.Now().After(s.deadline)
}

func (s *Schedule) ElapsedInterval() time.Duration {
	return arbmath.SaturatingCast[time.Duration](s.activeRound) * s.roundLength
}

// ErrNoMoreRounds is returned by WaitAndAdvanceRound when the schedule is already on the last round of the block, so
// there is no next round to advance to.
var ErrNoMoreRounds = errors.New("pga: no rounds remaining in the block")

// WaitAndAdvanceRound moves to the next round, waiting until the current round's deadline when called before it and
// returning immediately when called after it. It returns ErrNoMoreRounds without waiting when the schedule is already
// on the last round. If ctx is cancelled before the deadline, WaitAndAdvanceRound returns its error without advancing
// so the caller can stop building the block; otherwise it returns nil.
func (s *Schedule) WaitAndAdvanceRound(ctx context.Context) error {
	if s.IsLastRound() {
		return ErrNoMoreRounds
	}
	now := time.Now()
	boundary := s.deadline
	if now.Before(boundary) {
		timer := time.NewTimer(boundary.Sub(now))
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		boundary = now
	}
	s.deadline = boundary.Add(s.roundLength)
	s.activeRound++
	return nil
}
