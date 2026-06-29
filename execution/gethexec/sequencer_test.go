// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"
)

func TestSequencerConfigValidatePGA(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*SequencerConfig)
		wantErr bool
	}{
		{"default config", func(c *SequencerConfig) {}, false},
		{"pga enabled", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
		}, false},
		{"timeboost enabled", func(c *SequencerConfig) {
			c.Timeboost.Enable = true
		}, false},
		{"pga and timeboost enabled", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.Timeboost.Enable = true
		}, true},
		{"zero value pga config", func(c *SequencerConfig) {
			c.ExperimentalPGA = PGAConfig{}
		}, true},
		{"pga enabled with zero rounds per block", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.ExperimentalPGA.RoundsPerBlock = 0
		}, true},
		{"pga enabled with one round per block", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.ExperimentalPGA.RoundsPerBlock = 1
		}, false},
		{"round length below minimum", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.MaxBlockSpeed = 250 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 6
		}, true},
		{"round length at minimum", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.MaxBlockSpeed = 250 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 5
		}, false},
		{"fast blocks with pga disabled", func(c *SequencerConfig) {
			c.MaxBlockSpeed = 10 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 1
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultSequencerConfig
			tt.modify(&c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestPGARoundLength(t *testing.T) {
	c := DefaultSequencerConfig
	c.MaxBlockSpeed = 250 * time.Millisecond
	c.ExperimentalPGA.RoundsPerBlock = 2
	if got := c.PGARoundLength(); got != 125*time.Millisecond {
		t.Errorf("expected round length 125ms, got %v", got)
	}
}

// now0 is an arbitrary non-zero reference time. The zero time.Time{} is far in
// its past, which models "regular not throttled".
var now0 = time.Unix(1_700_000_000, 0)

const (
	testMaxBlockSpeed = 250 * time.Millisecond
	testPollInterval  = 50 * time.Millisecond
)

func TestDecideSequencingTurn(t *testing.T) {
	tests := []struct {
		name           string
		state          sequencingState
		hasRegularTxs  bool
		hasDelayedMsgs bool
		wantTurn       sequencingTurn
		wantWait       time.Duration
	}{
		{
			name:     "nothing pending polls",
			wantTurn: noSequencingTurn,
			wantWait: testPollInterval,
		},
		{
			name:          "only regular, not throttled",
			hasRegularTxs: true,
			wantTurn:      regularSequencingTurn,
		},
		{
			name:          "only regular, throttled waits remaining",
			state:         sequencingState{regularSequencingThrottledUntil: now0.Add(100 * time.Millisecond)},
			hasRegularTxs: true,
			wantTurn:      noSequencingTurn,
			wantWait:      100 * time.Millisecond,
		},
		{
			name:           "only delayed",
			hasDelayedMsgs: true,
			wantTurn:       delayedSequencingTurn,
		},
		{
			name:           "both, last turn regular yields to delayed",
			state:          sequencingState{lastTurn: regularSequencingTurn},
			hasRegularTxs:  true,
			hasDelayedMsgs: true,
			wantTurn:       delayedSequencingTurn,
		},
		{
			name:           "both, last turn delayed and not throttled picks regular",
			state:          sequencingState{lastTurn: delayedSequencingTurn},
			hasRegularTxs:  true,
			hasDelayedMsgs: true,
			wantTurn:       regularSequencingTurn,
		},
		{
			name:           "both, last turn delayed but regular throttled keeps delayed draining",
			state:          sequencingState{lastTurn: delayedSequencingTurn, regularSequencingThrottledUntil: now0.Add(100 * time.Millisecond)},
			hasRegularTxs:  true,
			hasDelayedMsgs: true,
			wantTurn:       delayedSequencingTurn,
		},
		{
			name:           "only delayed, throttled waits remaining",
			state:          sequencingState{delayedSequencingThrottledUntil: now0.Add(100 * time.Millisecond)},
			hasDelayedMsgs: true,
			wantTurn:       noSequencingTurn,
			wantWait:       100 * time.Millisecond,
		},
		{
			name:           "both, delayed throttled, regular runs",
			state:          sequencingState{lastTurn: regularSequencingTurn, delayedSequencingThrottledUntil: now0.Add(100 * time.Millisecond)},
			hasRegularTxs:  true,
			hasDelayedMsgs: true,
			wantTurn:       regularSequencingTurn,
		},
		{
			name:           "both throttled waits for the soonest to lift",
			state:          sequencingState{regularSequencingThrottledUntil: now0.Add(200 * time.Millisecond), delayedSequencingThrottledUntil: now0.Add(100 * time.Millisecond)},
			hasRegularTxs:  true,
			hasDelayedMsgs: true,
			wantTurn:       noSequencingTurn,
			wantWait:       100 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn, wait := decideSequencingTurn(tt.state, tt.hasRegularTxs, tt.hasDelayedMsgs, now0, testPollInterval)
			if turn != tt.wantTurn {
				t.Errorf("turn = %v, want %v", turn, tt.wantTurn)
			}
			if wait != tt.wantWait {
				t.Errorf("wait = %v, want %v", wait, tt.wantWait)
			}
		})
	}
}

func TestStateAfterRegular(t *testing.T) {
	// Block made / transient error => throttle one MaxBlockSpeed out.
	got := sequencingStateAfterRegularSequencing(sequencingState{}, testMaxBlockSpeed, now0)
	if got.lastTurn != regularSequencingTurn {
		t.Errorf("lastTurn = %v, want regular", got.lastTurn)
	}
	if want := now0.Add(testMaxBlockSpeed); !got.regularSequencingThrottledUntil.Equal(want) {
		t.Errorf("regularSequencingThrottledUntil = %v, want %v", got.regularSequencingThrottledUntil, want)
	}

	// Empty queue => throttle only the idle poll interval out, matching the wait
	// decideSequencingTurn uses when there is no pending work (not MaxBlockSpeed).
	got = sequencingStateAfterRegularSequencing(sequencingState{}, testPollInterval, now0)
	if want := now0.Add(testPollInterval); !got.regularSequencingThrottledUntil.Equal(want) {
		t.Errorf("regularSequencingThrottledUntil = %v, want %v", got.regularSequencingThrottledUntil, want)
	}

	// Items present but no block produced => no throttle, retry promptly.
	got = sequencingStateAfterRegularSequencing(sequencingState{regularSequencingThrottledUntil: now0.Add(time.Hour)}, 0, now0)
	if !got.regularSequencingThrottledUntil.IsZero() {
		t.Errorf("regularSequencingThrottledUntil = %v, want zero", got.regularSequencingThrottledUntil)
	}
}

func TestStateAfterDelayed(t *testing.T) {
	// Produced a delayed block => drain unthrottled, no delayed throttle set.
	got := sequencingStateAfterDelayedSequencing(sequencingState{}, true, now0, testMaxBlockSpeed)
	if got.lastTurn != delayedSequencingTurn {
		t.Errorf("lastTurn = %v, want delayed", got.lastTurn)
	}
	if !got.delayedSequencingThrottledUntil.IsZero() {
		t.Errorf("delayedSequencingThrottledUntil = %v, want zero", got.delayedSequencingThrottledUntil)
	}

	// No block produced (filtered halt) => throttle delayed one MaxBlockSpeed out;
	// the regular throttle is untouched.
	got = sequencingStateAfterDelayedSequencing(sequencingState{regularSequencingThrottledUntil: now0.Add(time.Hour)}, false, now0, testMaxBlockSpeed)
	if want := now0.Add(testMaxBlockSpeed); !got.delayedSequencingThrottledUntil.Equal(want) {
		t.Errorf("delayedSequencingThrottledUntil = %v, want %v", got.delayedSequencingThrottledUntil, want)
	}
	if want := now0.Add(time.Hour); !got.regularSequencingThrottledUntil.Equal(want) {
		t.Errorf("regularSequencingThrottledUntil = %v, want %v (untouched)", got.regularSequencingThrottledUntil, want)
	}
}
