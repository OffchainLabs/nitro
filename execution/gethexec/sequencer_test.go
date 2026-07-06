// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
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

// TestEndSequencingDelayedCommitOutcome verifies that the delayed-message pop
// is keyed on the commit outcome reported to EndSequencing, not on the staged
// result: a failed durable write leaves the message queued for retry, a
// successful one pops it.
func TestEndSequencingDelayedCommitOutcome(t *testing.T) {
	newSequencerWithPendingDelayedCommit := func(t *testing.T) *Sequencer {
		engine := &ExecutionEngine{}
		engine.delayedMsgs.Push(&delayedMsg{msgIdx: 7})
		configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
		seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		seq.pendingDelayedMsgCommit = true
		return seq
	}

	t.Run("failed commit leaves message queued", func(t *testing.T) {
		seq := newSequencerWithPendingDelayedCommit(t)
		seq.EndSequencing(context.Background(), errors.New("durable write failed"))
		if seq.pendingDelayedMsgCommit {
			t.Error("pendingDelayedMsgCommit should be cleared")
		}
		if seq.execEngine.delayedMsgs.Len() != 1 {
			t.Error("delayed message should stay queued for retry after a failed commit")
		}
	})

	t.Run("successful commit pops message", func(t *testing.T) {
		seq := newSequencerWithPendingDelayedCommit(t)
		seq.EndSequencing(context.Background(), nil)
		if seq.pendingDelayedMsgCommit {
			t.Error("pendingDelayedMsgCommit should be cleared")
		}
		if seq.execEngine.delayedMsgs.Len() != 0 {
			t.Error("delayed message should be popped after a successful commit")
		}
	})
}

func TestSequencerDoesntBlockWithoutTransactions(t *testing.T) {
	engine := &ExecutionEngine{}
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan time.Duration, 1)
	go func() {
		_, wait := seq.StartSequencing(context.Background())
		done <- wait
	}()

	select {
	case wait := <-done:
		if wait == 0 {
			t.Fatal("expected non-zero next sequence time")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StartSequencing blocked without transactions")
	}
}
