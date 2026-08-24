// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func TestSequencerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*SequencerConfig)
		wantErr bool
	}{
		{"default config", func(c *SequencerConfig) {}, false},
		{"forced fifo", func(c *SequencerConfig) {
			c.PGA.DangerousForceFIFO = true
		}, false},
		{"timeboost enabled", func(c *SequencerConfig) {
			c.Timeboost.Enable = true
		}, false},
		{"forced fifo and timeboost enabled", func(c *SequencerConfig) {
			c.PGA.DangerousForceFIFO = true
			c.Timeboost.Enable = true
		}, false},
		{"zero value pga config", func(c *SequencerConfig) {
			c.PGA = PGAConfig{}
		}, true},
		{"zero rounds per block", func(c *SequencerConfig) {
			c.PGA.RoundsPerBlock = 0
		}, true},
		{"one round per block", func(c *SequencerConfig) {
			c.PGA.RoundsPerBlock = 1
		}, false},
		{"many rounds per block", func(c *SequencerConfig) {
			c.Enable = true
			c.MaxBlockSpeed = 250 * time.Millisecond
			c.PGA.RoundsPerBlock = 6
		}, false},
		{"round length below 5ms", func(c *SequencerConfig) {
			c.MaxBlockSpeed = 10 * time.Millisecond
			c.PGA.RoundsPerBlock = 6
		}, true},
		{"zero max block tx candidates", func(c *SequencerConfig) {
			c.MaxBlockTxCandidates = 0
		}, true},
		{"negative max block tx candidates", func(c *SequencerConfig) {
			c.MaxBlockTxCandidates = -1
		}, true},
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
	if got := pgaRoundLength(250*time.Millisecond, 2); got != 125*time.Millisecond {
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
		errWhileSeq error
	}{
		{"retry", execution.ErrRetrySequencer},
		{"success", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ExecutionEngine{}
			configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
			seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			seq.pendingQueueItemsResults = &pendingQueueItemsResults{hooks: &FullSequencingHooks{}}

			seq.EndSequencing(context.Background(), tt.errWhileSeq)

			if seq.pendingQueueItemsResults != nil {
				t.Error("pendingQueueItemsResults should be cleared by EndSequencing")
			}
		})
	}
}

func TestEndSequencingRoutesStagedQueueItems(t *testing.T) {
	newSeqWithStagedItem := func(t *testing.T) (*Sequencer, chan error) {
		engine := &ExecutionEngine{}
		configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
		seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		resultChan := make(chan error, 1)
		seq.pendingQueueItemsResults = &pendingQueueItemsResults{
			hooks: &FullSequencingHooks{
				sequencedTxs: []sequencedTx{{queueItem: txQueueItem{resultChan: resultChan, returnedResult: &atomic.Bool{}}}},
			},
		}
		return seq, resultChan
	}

	t.Run("retry without forwarder re-queues the item", func(t *testing.T) {
		seq, resultChan := newSeqWithStagedItem(t)

		seq.EndSequencing(context.Background(), execution.ErrRetrySequencer)

		if seq.txRetryQueue.Len() != 1 {
			t.Fatalf("staged item should be re-queued for retry, txRetryQueue len = %d", seq.txRetryQueue.Len())
		}
		select {
		case err := <-resultChan:
			t.Errorf("item should be retried, not returned to submitter; got result %v", err)
		default:
		}
	})

	t.Run("retry with forwarder re-queues the item without forwarding", func(t *testing.T) {
		seq, resultChan := newSeqWithStagedItem(t)
		seq.forwarder = &TxForwarder{}

		seq.EndSequencing(context.Background(), execution.ErrRetrySequencer)

		if seq.txRetryQueue.Len() != 1 {
			t.Fatalf("staged item should be re-queued for the backgroundForwarder, txRetryQueue len = %d", seq.txRetryQueue.Len())
		}
		select {
		case err := <-resultChan:
			t.Errorf("item should be retried, not returned to submitter; got result %v", err)
		default:
		}
	})

	t.Run("generic error returns the item to its submitter", func(t *testing.T) {
		seq, resultChan := newSeqWithStagedItem(t)

		wantErr := errors.New("durable write failed")
		seq.EndSequencing(context.Background(), wantErr)

		if seq.txRetryQueue.Len() != 0 {
			t.Errorf("item should be returned to submitter, not re-queued; txRetryQueue len = %d", seq.txRetryQueue.Len())
		}
		select {
		case err := <-resultChan:
			if !errors.Is(err, wantErr) {
				t.Errorf("submitter got %v, want %v", err, wantErr)
			}
		default:
			t.Error("submitter never received a result")
		}
	})
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

// A block-creation turn that exits after the orderer is armed must leave the never-attempted
// txs in txRetryQueue via the deferred sweep, not fail them back to their submitters.
func TestCreateBlockRequeuesNeverAttemptedTxs(t *testing.T) {
	engine := newTestRecorderEngine(t, 0) // genesis-only chain so the pre-StartBlock state fetch succeeds
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	// A non-nil l1Reader with no known L1 block forces the early exit after StartBlock.
	seq, err := NewSequencer(engine, &headerreader.HeaderReader{}, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	item, resultChan := makeTestQueueItem(t, 0, testBaseFee)
	ordererConfig := txOrdererConfig{
		baseFee:              big.NewInt(testBaseFee),
		maxBlockTxCandidates: math.MaxInt,
		maxBlockSpeed:        DefaultSequencerConfig.MaxBlockSpeed,
	}
	orderer := newFIFOTxOrderer(newStubOrdererSequencer(item), ordererConfig, DefaultSequencerConfig.PollInterval)

	sequencedMsg, _ := seq.createBlockWithTxOrderer(context.Background(), orderer)

	if sequencedMsg != nil {
		t.Fatal("expected no block to be sequenced")
	}
	if seq.txRetryQueue.Len() != 1 {
		t.Fatalf("txRetryQueue.Len() = %d, want 1 (never-attempted tx re-queued)", seq.txRetryQueue.Len())
	}
	select {
	case res := <-resultChan:
		t.Errorf("never-attempted tx was resolved with %v; it should only be re-queued", res)
	default:
	}
}

// panicAfterArmOrderer arms the block's candidates normally, then panics: it stands in for a
// panic anywhere in block creation while the orderer still holds txs.
type panicAfterArmOrderer struct{ *fifoTxOrderer }

func (p panicAfterArmOrderer) StartBlock(statedb *state.StateDB) bool {
	p.fifoTxOrderer.StartBlock(statedb)
	panic("test-injected block creation panic")
}

// A panic during block creation must fail the orderer's remaining txs with an internal error
// rather than requeue them: a requeue could resurrect a tx that panics the sequencer in a loop.
func TestCreateBlockPanicFailsTxsInsteadOfRequeueing(t *testing.T) {
	engine := newTestRecorderEngine(t, 0)
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	item, resultChan := makeTestQueueItem(t, 0, testBaseFee)
	ordererConfig := txOrdererConfig{
		baseFee:              big.NewInt(testBaseFee),
		maxBlockTxCandidates: math.MaxInt,
		maxBlockSpeed:        DefaultSequencerConfig.MaxBlockSpeed,
	}
	orderer := newFIFOTxOrderer(newStubOrdererSequencer(item), ordererConfig, DefaultSequencerConfig.PollInterval)

	sequencedMsg, throttle := seq.createBlockWithTxOrderer(context.Background(), panicAfterArmOrderer{orderer})

	if sequencedMsg != nil {
		t.Fatal("expected no block to be sequenced")
	}
	if throttle != DefaultSequencerConfig.MaxBlockSpeed {
		t.Errorf("throttle = %v, want MaxBlockSpeed %v", throttle, DefaultSequencerConfig.MaxBlockSpeed)
	}
	select {
	case res := <-resultChan:
		if !errors.Is(res, sequencerInternalError) {
			t.Errorf("tx got %v, want sequencerInternalError", res)
		}
	default:
		t.Error("tx from the panicked block was never resolved")
	}
	if seq.txRetryQueue.Len() != 0 {
		t.Errorf("txRetryQueue.Len() = %d, want 0 (failed txs must not be requeued)", seq.txRetryQueue.Len())
	}
}

// While inactive, backgroundForwarder is the only path that moves queued txs to the forwarder.
// With the forwarder temporarily disabled (ErrNoSequencer), the drained items must be re-queued
// for retry, not resolved or dropped.
func TestBackgroundForwarderDrainsQueuesWhileInactive(t *testing.T) {
	engine := &ExecutionEngine{}
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seq.forwarder = &TxForwarder{}

	queued, queuedChan := makeTestQueueItem(t, 0, testBaseFee)
	seq.txQueue <- queued
	retried, retriedChan := makeTestQueueItem(t, 1, testBaseFee)
	seq.txRetryQueue.Push(retried)
	parked, parkedChan := makeTestQueueItem(t, 2, testBaseFee)
	seq.nonceFailures.cache.Add(
		addressAndNonce{nonce: 2},
		&nonceFailure{queueItem: parked, expiry: time.Now().Add(time.Hour)},
	)

	seq.backgroundForwarder(context.Background())

	if n := len(seq.txQueue); n != 0 {
		t.Errorf("len(txQueue) = %d, want 0", n)
	}
	if n := seq.nonceFailures.Len(); n != 0 {
		t.Errorf("nonceFailures.Len() = %d, want 0", n)
	}
	if n := seq.txRetryQueue.Len(); n != 3 {
		t.Errorf("txRetryQueue.Len() = %d, want 3", n)
	}
	for name, ch := range map[string]chan error{"queued": queuedChan, "retried": retriedChan, "parked": parkedChan} {
		select {
		case res := <-ch:
			t.Errorf("%s tx was resolved with %v; it should only be re-queued", name, res)
		default:
		}
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
	seq.nonceFailures.cache.Add(
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

// failQueuedItems runs when the shutdown forwarder fails to initialize; it must resolve every
// queued item so submitters fail fast instead of waiting out their abort deadlines.
func TestFailQueuedItemsResolvesAllQueues(t *testing.T) {
	engine := &ExecutionEngine{}
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	pendingChan := make(chan error, 1)
	tx := types.NewTx(&types.DynamicFeeTx{Gas: 21000})
	seq.txQueue <- newRegularTxQueueItem(context.Background(), tx, nil, pendingChan, false, 0)

	parkedChan := make(chan error, 1)
	seq.nonceFailures.cache.Add(
		addressAndNonce{nonce: 7},
		&nonceFailure{
			queueItem: txQueueItem{
				resultChan:     parkedChan,
				returnedResult: &atomic.Bool{},
				ctx:            context.Background(),
			},
		},
	)

	seq.failQueuedItems()

	for name, resultChan := range map[string]chan error{"pending tx": pendingChan, "parked nonce failure": parkedChan} {
		select {
		case res := <-resultChan:
			if !errors.Is(res, ErrNoSequencer) {
				t.Errorf("%s got %v, want ErrNoSequencer", name, res)
			}
		default:
			t.Errorf("%s was never resolved", name)
		}
	}
	if n := len(seq.txQueue); n != 0 {
		t.Errorf("len(txQueue) = %d, want 0", n)
	}
	if n := seq.nonceFailures.Len(); n != 0 {
		t.Errorf("nonceFailures.Len() = %d, want 0", n)
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
