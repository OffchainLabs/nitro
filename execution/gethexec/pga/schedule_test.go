// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"errors"
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

func TestScheduleRoundIsOver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewSchedule(2, testRoundLength)
		if s.RoundIsOver() {
			t.Error("round reported over at its start")
		}
		// RoundIsOver is a strict after: exactly at the deadline the round is still on.
		time.Sleep(testRoundLength)
		if s.RoundIsOver() {
			t.Error("round reported over exactly at its deadline")
		}
		time.Sleep(time.Nanosecond)
		if !s.RoundIsOver() {
			t.Error("round not reported over past its deadline")
		}
		// Advancing to the next round renews the deadline.
		if err := s.WaitAndAdvanceRound(context.Background()); err != nil {
			t.Fatalf("WaitAndAdvanceRound returned %v, want nil", err)
		}
		if s.RoundIsOver() {
			t.Error("new round reported over at its start")
		}
	})
}

func TestScheduleWaitAndAdvanceRoundWaitsForBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		s := NewSchedule(2, testRoundLength)
		deadline := s.Deadline()

		// Round 1 has not reached its deadline, so WaitAndAdvanceRound sleeps to it.
		if err := s.WaitAndAdvanceRound(context.Background()); err != nil {
			t.Fatalf("WaitAndAdvanceRound returned %v, want nil at the boundary", err)
		}

		if got := time.Since(start); got != testRoundLength {
			t.Errorf("WaitAndAdvanceRound slept %v, want %v", got, testRoundLength)
		}
		if got := s.Round(); got != 2 {
			t.Errorf("round = %d, want 2", got)
		}
		if got, want := s.Deadline(), deadline.Add(testRoundLength); !got.Equal(want) {
			t.Errorf("round 2 deadline = %v, want %v", got, want)
		}
	})
}

func TestScheduleWaitAndAdvanceRoundReturnsImmediatelyAfterOverrun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewSchedule(2, testRoundLength)

		// Run past the deadline before advancing.
		time.Sleep(testRoundLength + 50*time.Millisecond)
		overran := time.Now()

		if err := s.WaitAndAdvanceRound(context.Background()); err != nil {
			t.Fatalf("WaitAndAdvanceRound returned %v, want nil after overrun", err)
		}

		if got := time.Now(); !got.Equal(overran) {
			t.Errorf("WaitAndAdvanceRound advanced the clock to %v, want immediate return at %v", got, overran)
		}
		if got := s.Round(); got != 2 {
			t.Errorf("round = %d, want 2", got)
		}
		if got, want := s.Deadline(), overran.Add(testRoundLength); !got.Equal(want) {
			t.Errorf("round 2 deadline = %v, want %v", got, want)
		}
	})
}

func TestScheduleWaitAndAdvanceRoundContextCancelInterruptsWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		start := time.Now()
		s := NewSchedule(2, testRoundLength)

		cancel()

		if err := s.WaitAndAdvanceRound(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("WaitAndAdvanceRound returned %v, want context.Canceled", err)
		}
		if got := time.Since(start); got != 0 {
			t.Errorf("WaitAndAdvanceRound waited %v, want immediate return", got)
		}
		if got := s.Round(); got != 1 {
			t.Errorf("round = %d, want 1 (unchanged) after interruption", got)
		}
	})
}

func TestScheduleWaitAndAdvanceRoundContextCancelMidWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		start := time.Now()
		s := NewSchedule(2, testRoundLength)

		// WaitAndAdvanceRound blocks on the round-1 deadline in its own goroutine.
		errc := make(chan error, 1)
		go func() {
			errc <- s.WaitAndAdvanceRound(ctx)
		}()

		// Cancel partway through the round, while the call is still waiting.
		const elapsed = testRoundLength / 2
		time.Sleep(elapsed)
		cancel()

		if err := <-errc; !errors.Is(err, context.Canceled) {
			t.Errorf("WaitAndAdvanceRound returned %v, want context.Canceled", err)
		}
		if got := time.Since(start); got != elapsed {
			t.Errorf("WaitAndAdvanceRound returned after %v, want %v (mid-wait cancellation)", got, elapsed)
		}
		if got := s.Round(); got != 1 {
			t.Errorf("round = %d, want 1 (unchanged) after interruption", got)
		}
	})
}

func TestScheduleWaitAndAdvanceRoundOnLastRoundReturnsError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		s := NewSchedule(1, testRoundLength)
		if !s.IsLastRound() {
			t.Fatal("round 1 of 1 should be the last round")
		}

		if err := s.WaitAndAdvanceRound(context.Background()); !errors.Is(err, ErrNoMoreRounds) {
			t.Errorf("WaitAndAdvanceRound on the last round returned %v, want ErrNoMoreRounds", err)
		}
		if got := time.Since(start); got != 0 {
			t.Errorf("WaitAndAdvanceRound waited %v on the last round, want immediate return", got)
		}
		if got := s.Round(); got != 1 {
			t.Errorf("round = %d, want 1 (unchanged) after WaitAndAdvanceRound on the last round", got)
		}
	})
}

func TestScheduleAdvancesThroughBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const rounds = 3
		blockStart := time.Now()
		s := NewSchedule(rounds, testRoundLength)
		for r := uint(1); r < rounds; r++ {
			if s.IsLastRound() {
				t.Fatalf("round %d should not be the last of %d", r, rounds)
			}
			if err := s.WaitAndAdvanceRound(context.Background()); err != nil {
				t.Fatalf("round %d WaitAndAdvanceRound returned %v, want nil", r, err)
			}
			// r is a small loop counter, so converting it to a duration cannot overflow.
			want := time.Duration(r) * testRoundLength // #nosec G115
			if got := time.Since(blockStart); got != want {
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
