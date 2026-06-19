// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"time"
)

// Schedule tracks which PGA round of the block is active and decides when each execute phase begins.
type Schedule struct {
	roundsPerBlock int
	roundLength    time.Duration
	activeRound    int
	deadline       time.Time
}

// NewSchedule returns a Schedule at round 1 of a block whose first execute phase begins now.
func NewSchedule(roundsPerBlock int, roundLength time.Duration) *Schedule {
	return &Schedule{
		roundsPerBlock: roundsPerBlock,
		roundLength:    roundLength,
		activeRound:    1,
		deadline:       time.Now().Add(roundLength),
	}
}

func (s *Schedule) Round() int {
	return s.activeRound
}

func (s *Schedule) IsLastRound() bool {
	return s.activeRound >= s.roundsPerBlock
}

func (s *Schedule) Deadline() time.Time {
	return s.deadline
}

// AdvanceRound moves to the next round, waiting until the current round's deadline when called before it and returning
// immediately when called after it. If ctx is cancelled before the deadline, AdvanceRound returns its error without
// advancing so the caller can stop building the block; otherwise it returns nil.
func (s *Schedule) AdvanceRound(ctx context.Context) error {
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
