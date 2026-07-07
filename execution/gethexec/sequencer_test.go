// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/execution"
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
		engine.waitingForFilteredTx = &FilteredTxWaitState{DelayedMsgIdx: 7}
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
		if seq.execEngine.waitingForFilteredTx == nil {
			t.Error("filtered-tx halt should not be considered resolved by a failed commit")
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
		if seq.execEngine.waitingForFilteredTx != nil {
			t.Error("filtered-tx halt should be resolved by a successful commit")
		}
	})
}

// EndSequencing must consume the staged queue-item results on every path;
// a leftover struct would be re-processed by a later no-op turn's
// EndSequencing(nil).
func TestEndSequencingClearsPendingQueueItemsResults(t *testing.T) {
	tests := []struct {
		name        string
		forwarder   *TxForwarder
		errWhileSeq error
	}{
		{"retry with forwarder", &TxForwarder{}, execution.ErrRetrySequencer},
		{"retry without forwarder", nil, execution.ErrRetrySequencer},
		{"success", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ExecutionEngine{}
			configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
			seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			seq.forwarder = tt.forwarder
			seq.pendingQueueItemsResults = &pendingQueueItemsResults{hooks: &FullSequencingHooks{}}

			seq.EndSequencing(context.Background(), tt.errWhileSeq)

			if seq.pendingQueueItemsResults != nil {
				t.Error("pendingQueueItemsResults should be cleared by EndSequencing")
			}
		})
	}
}

func TestCheckHealthChosenSequencerDeadline(t *testing.T) {
	past := time.Now().Add(-time.Second)
	future := time.Now().Add(time.Hour)
	tests := []struct {
		name        string
		forwarder   *TxForwarder
		isActive    bool
		activeUntil *time.Time
		wantErr     error
	}{
		// Zero-value TxForwarder has enabled=false, so its CheckHealth returns
		// ErrNoSequencer.
		{"forwarding delegates to forwarder", &TxForwarder{}, false, &past, ErrNoSequencer},
		{"paused is healthy even past deadline", nil, false, &past, nil},
		{"active without coordinator signal is healthy", nil, true, nil, nil},
		{"active within deadline is healthy", nil, true, &future, nil},
		{"active past deadline is not chosen", nil, true, &past, ErrNotChosenSequencer},
		{"active after release is not chosen", nil, true, &time.Time{}, ErrNotChosenSequencer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ExecutionEngine{}
			configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
			seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			seq.forwarder = tt.forwarder
			seq.isActive = tt.isActive
			if tt.activeUntil != nil {
				seq.SetActiveUntil(*tt.activeUntil)
			}

			if got := seq.CheckHealth(context.Background()); !errors.Is(got, tt.wantErr) {
				t.Errorf("CheckHealth() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestBackgroundForwarderExpiresNonceFailuresWhileInactive(t *testing.T) {
	engine := &ExecutionEngine{}
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	nonceErr := errors.New("nonce too high")
	resultChan := make(chan error, 1)
	seq.nonceFailures.LruCache.Add(
		addressAndNonce{nonce: 7},
		&nonceFailure{
			queueItem: txQueueItem{
				resultChan:     resultChan,
				returnedResult: &atomic.Bool{},
				ctx:            context.Background(),
			},
			nonceErr: nonceErr,
			expiry:   time.Now().Add(-time.Second),
		},
	)

	seq.backgroundForwarder(context.Background())

	select {
	case res := <-resultChan:
		if !errors.Is(res, nonceErr) {
			t.Errorf("parked tx got %v, want the original nonce error", res)
		}
	default:
		t.Error("expired nonce failure was not returned to the client")
	}
	if seq.nonceFailures.Len() != 0 {
		t.Errorf("nonceFailures.Len() = %d, want 0", seq.nonceFailures.Len())
	}
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
