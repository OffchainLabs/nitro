// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"testing"
	"testing/synctest"
	"time"
)

const testRoundLength = 125 * time.Millisecond

func TestScheduleInitialState(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		now := time.Now()
		s := NewSchedule(2, testRoundLength)
		if got := s.Round(); got != 1 {
			t.Errorf("initial round = %d, want 1", got)
		}
		if s.IsLastRound() {
			t.Error("round 1 of 2 should not be the last round")
		}
		if got, want := s.Deadline(), now.Add(testRoundLength); !got.Equal(want) {
			t.Errorf("round 1 deadline = %v, want %v", got, want)
		}
	})
}

func TestScheduleAdvanceRoundWaitsForBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		s := NewSchedule(2, testRoundLength)
		deadline := s.Deadline()

		// Round 1 has not reached its deadline, so AdvanceRound sleeps to it.
		s.AdvanceRound()

		if got := time.Since(start); got != testRoundLength {
			t.Errorf("AdvanceRound slept %v, want %v", got, testRoundLength)
		}
		if got := s.Round(); got != 2 {
			t.Errorf("round = %d, want 2", got)
		}
		if got, want := s.Deadline(), deadline.Add(testRoundLength); !got.Equal(want) {
			t.Errorf("round 2 deadline = %v, want %v", got, want)
		}
	})
}

func TestScheduleAdvanceRoundReturnsImmediatelyAfterOverrun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewSchedule(2, testRoundLength)

		// Run past the deadline before advancing.
		time.Sleep(testRoundLength + 50*time.Millisecond)
		overran := time.Now()

		s.AdvanceRound()

		if got := time.Now(); !got.Equal(overran) {
			t.Errorf("AdvanceRound advanced the clock to %v, want immediate return at %v", got, overran)
		}
		if got := s.Round(); got != 2 {
			t.Errorf("round = %d, want 2", got)
		}
		if got, want := s.Deadline(), overran.Add(testRoundLength); !got.Equal(want) {
			t.Errorf("round 2 deadline = %v, want %v", got, want)
		}
	})
}

func TestScheduleAdvancesThroughBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const rounds = 3
		blockStart := time.Now()
		s := NewSchedule(rounds, testRoundLength)
		for r := 1; r < rounds; r++ {
			if s.IsLastRound() {
				t.Fatalf("round %d should not be the last of %d", r, rounds)
			}
			s.AdvanceRound()
			if got, want := time.Since(blockStart), time.Duration(r)*testRoundLength; got != want {
				t.Fatalf("round %d boundary at +%v, want +%v", r+1, got, want)
			}
			if gotRound := s.Round(); gotRound != r+1 {
				t.Fatalf("round = %d, want %d", gotRound, r+1)
			}
		}
		if !s.IsLastRound() {
			t.Errorf("round %d should be the last of %d", s.Round(), rounds)
		}
	})
}

func TestScheduleSingleRoundPerBlock(t *testing.T) {
	s := NewSchedule(1, testRoundLength)
	if !s.IsLastRound() {
		t.Error("round 1 of 1 should be the last round")
	}
}
