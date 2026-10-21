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
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

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
		{"max size sampling exponent", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMaxExponent = MaxFIFOSizeSamplingExponent
		}, false},
		{"size sampling exponent above max", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMaxExponent = MaxFIFOSizeSamplingExponent + 1
		}, true},
		{"size sampling min tx size at zero", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMinTxSize = 0
		}, false},
		{"size sampling min tx size at max tx data size", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMinTxSize = c.MaxTxDataSize
		}, false},
		{"negative size sampling min tx size", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMinTxSize = -1
		}, true},
		{"size sampling min tx size above max tx data size", func(c *SequencerConfig) {
			c.FIFO.SizeSamplingMinTxSize = c.MaxTxDataSize + 1
		}, true},
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

func TestEndSequencingTracksDataLimitedBlockStreak(t *testing.T) {
	engine := &ExecutionEngine{}
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	endBlock := func(status ordererStatus, txErr, errWhileSequencing error) {
		seq.pendingQueueItemsResults = &pendingQueueItemsResults{
			hooks: &FullSequencingHooks{
				ordererStatus: status,
				sequencedTxs:  []sequencedTx{{queueItem: txQueueItem{resultChan: make(chan error, 1), returnedResult: &atomic.Bool{}}, err: txErr}},
			},
		}
		seq.EndSequencing(context.Background(), errWhileSequencing)
	}
	endDataLimitedBlocks := func(count uint) {
		for range count {
			endBlock(blockSizeLimitReached, nil, nil)
		}
		if seq.dataLimitedBlockStreak != count {
			t.Fatalf("streak after %d data-limited blocks = %d, want %d", count, seq.dataLimitedBlockStreak, count)
		}
	}

	endDataLimitedBlocks(2)
	endBlock(exhaustedQueue, nil, nil)
	if seq.dataLimitedBlockStreak != 0 {
		t.Fatalf("streak after a non-data-limited block = %d, want 0", seq.dataLimitedBlockStreak)
	}

	endDataLimitedBlocks(2)
	endBlock(blockSizeLimitReached, errors.New("tx failed"), nil)
	if seq.dataLimitedBlockStreak != 0 {
		t.Fatalf("streak after a block with no successful tx = %d, want 0", seq.dataLimitedBlockStreak)
	}

	endDataLimitedBlocks(2)
	endBlock(blockSizeLimitReached, nil, execution.ErrRetrySequencer)
	if seq.dataLimitedBlockStreak != 0 {
		t.Fatalf("streak after a block the consensus side rejected = %d, want 0", seq.dataLimitedBlockStreak)
	}

	endDataLimitedBlocks(2)
	endBlock(blockSizeLimitReached, nil, errors.New("write failed"))
	if seq.dataLimitedBlockStreak != 0 {
		t.Fatalf("streak after a block that failed to commit = %d, want 0", seq.dataLimitedBlockStreak)
	}

	endDataLimitedBlocks(2)
	seq.pendingQueueItemsResults = nil
	seq.EndSequencing(context.Background(), nil)
	if seq.dataLimitedBlockStreak != 2 {
		t.Fatalf("streak after EndSequencing with no pending results = %d, want unchanged 2", seq.dataLimitedBlockStreak)
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
		latestHeader:         engine.bc.CurrentBlock(),
		baseFee:              big.NewInt(testBaseFee),
		maxBlockTxCandidates: math.MaxInt,
		maxBlockSpeed:        DefaultSequencerConfig.MaxBlockSpeed,
	}
	orderer := newFIFOTxOrderer(newStubOrdererSequencer(item), ordererConfig, DefaultFIFOConfig, DefaultSequencerConfig.PollInterval, 0)

	statedb, err := engine.bc.StateAt(ordererConfig.latestHeader)
	if err != nil {
		t.Fatal(err)
	}
	sequencedMsg, _ := seq.createBlockWithTxOrderer(context.Background(), orderer, statedb)

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
		latestHeader:         engine.bc.CurrentBlock(),
		baseFee:              big.NewInt(testBaseFee),
		maxBlockTxCandidates: math.MaxInt,
		maxBlockSpeed:        DefaultSequencerConfig.MaxBlockSpeed,
	}
	orderer := newFIFOTxOrderer(newStubOrdererSequencer(item), ordererConfig, DefaultFIFOConfig, DefaultSequencerConfig.PollInterval, 0)

	statedb, err := engine.bc.StateAt(ordererConfig.latestHeader)
	if err != nil {
		t.Fatal(err)
	}
	sequencedMsg, throttle := seq.createBlockWithTxOrderer(context.Background(), panicAfterArmOrderer{orderer}, statedb)

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

// TestDelayedSequencingPanicThrottlesDelayedTurns drives a real panic through
// a delayed turn: the engine has no chain, so sequencing the head delayed
// message panics on the nil blockchain.
func TestDelayedSequencingPanicThrottlesDelayedTurns(t *testing.T) {
	ctx := context.Background()
	engine := &ExecutionEngine{}
	engine.delayedMsgs.Push(&delayedMsg{msgIdx: 7})
	configFetcher := func() *SequencerConfig { c := DefaultSequencerConfig; return &c }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seq.Activate()

	before := time.Now()
	sequencedMsg, wait := seq.StartSequencing(ctx)
	after := time.Now()
	if sequencedMsg != nil {
		t.Error("panicking delayed turn produced a sequenced message")
	}
	if wait != 0 {
		t.Errorf("wait = %v, want 0 for a taken turn", wait)
	}
	if seq.pendingDelayedMsgCommit {
		t.Error("pendingDelayedMsgCommit set after a panicking delayed turn")
	}
	if seq.sequencingState.lastTurn != delayedSequencingTurn {
		t.Errorf("lastTurn = %v, want delayedSequencingTurn", seq.sequencingState.lastTurn)
	}
	throttledUntil := seq.sequencingState.delayedSequencingThrottledUntil
	if throttledUntil.Before(before.Add(delayedSequencingPanicThrottle)) || throttledUntil.After(after.Add(delayedSequencingPanicThrottle)) {
		t.Errorf("delayedSequencingThrottledUntil = %v, want within [%v, %v]", throttledUntil, before.Add(delayedSequencingPanicThrottle), after.Add(delayedSequencingPanicThrottle))
	}
	if !engine.createBlocksMutex.TryLock() {
		t.Fatal("createBlocksMutex still locked after the panic")
	}
	engine.createBlocksMutex.Unlock()
	if !engine.delayedMsgsMutex.TryLock() {
		t.Fatal("delayedMsgsMutex still locked after the panic")
	}
	engine.delayedMsgsMutex.Unlock()
	seq.EndSequencing(ctx, nil)
	if engine.delayedMsgs.Len() != 1 {
		t.Error("delayed message should stay queued after a panicking turn")
	}

	sequencedMsg, wait = seq.StartSequencing(ctx)
	if sequencedMsg != nil {
		t.Error("throttled delayed turn produced a sequenced message")
	}
	if wait <= DefaultSequencerConfig.MaxBlockSpeed {
		t.Errorf("wait = %v, want longer than MaxBlockSpeed %v while the delayed turn is throttled after a panic", wait, DefaultSequencerConfig.MaxBlockSpeed)
	}
	if seq.sequencingState.lastTurn != delayedSequencingTurn {
		t.Errorf("lastTurn = %v, want delayedSequencingTurn (no turn taken while throttled)", seq.sequencingState.lastTurn)
	}
	seq.EndSequencing(ctx, nil)
	if engine.delayedMsgs.Len() != 1 {
		t.Error("delayed message should stay queued while throttled")
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

func waitForQueuedTxCount(t *testing.T, seq *Sequencer, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for seq.QueuedTxCount() != want {
		if time.Now().After(deadline) {
			t.Fatalf("QueuedTxCount() = %d, want %d", seq.QueuedTxCount(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func awaitResult(t *testing.T, resultChan chan error, label string) error {
	t.Helper()
	select {
	case res := <-resultChan:
		return res
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not return", label)
		return nil
	}
}

func TestPublishTransactionDedupsInFlightNonces(t *testing.T) {
	ctx := context.Background()
	engine := newTestRecorderEngine(t, 0)
	config := DefaultSequencerConfig
	configFetcher := func() *SequencerConfig { return &config }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := types.LatestSigner(engine.bc.Config())
	sign := func(tipCap, feeCap int64) *types.Transaction {
		return types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   engine.bc.Config().ChainID,
			Nonce:     0,
			GasTipCap: big.NewInt(tipCap),
			GasFeeCap: big.NewInt(feeCap),
			Gas:       21000,
		})
	}

	first := sign(10, 100)
	firstResult := make(chan error, 1)
	go func() { firstResult <- seq.PublishTransaction(ctx, first, nil) }()
	waitForQueuedTxCount(t, seq, 1)

	if err := seq.PublishTransaction(ctx, first, nil); !errors.Is(err, txpool.ErrAlreadyKnown) {
		t.Errorf("resubmitting the same tx = %v, want ErrAlreadyKnown", err)
	}
	bumped := sign(11, 110)
	seq.orderingStrategy.Store(orderingStrategyFIFO)
	if err := seq.PublishTransaction(ctx, bumped, nil); !errors.Is(err, ErrNonceInFlight) {
		t.Errorf("same nonce under FIFO = %v, want ErrNonceInFlight", err)
	}

	seq.orderingStrategy.Store(orderingStrategyPGA)
	if err := seq.PublishTransaction(ctx, sign(10, 109), nil); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Errorf("same nonce without fee bump under PGA = %v, want ErrReplaceUnderpriced", err)
	}
	bumpedResult := make(chan error, 1)
	go func() { bumpedResult <- seq.PublishTransaction(ctx, bumped, nil) }()
	waitForQueuedTxCount(t, seq, 2)
	if err := seq.PublishTransaction(ctx, bumped, nil); !errors.Is(err, txpool.ErrAlreadyKnown) {
		t.Errorf("resubmitting the replacement = %v, want ErrAlreadyKnown", err)
	}
	if n := seq.inFlightTxs.len(); n != 1 {
		t.Errorf("inFlightTxs.len() = %d, want 1 while both calls are in flight", n)
	}

	drain := func(want *types.Transaction) {
		t.Helper()
		item := <-seq.txQueue
		if item.tx.Hash() != want.Hash() {
			t.Fatalf("drained item is %s, want %s", item.tx.Hash(), want.Hash())
		}
		item.returnResult(nil)
	}
	drain(first)
	if res := awaitResult(t, firstResult, "first submission"); res != nil {
		t.Errorf("first submission = %v, want nil", res)
	}
	if n := seq.inFlightTxs.len(); n != 1 {
		t.Errorf("inFlightTxs.len() = %d, want 1 while the replacement is in flight", n)
	}
	if err := seq.PublishTransaction(ctx, bumped, nil); !errors.Is(err, txpool.ErrAlreadyKnown) {
		t.Errorf("resubmitting the replacement after the replaced tx returned = %v, want ErrAlreadyKnown", err)
	}
	drain(bumped)
	if res := awaitResult(t, bumpedResult, "replacement submission"); res != nil {
		t.Errorf("replacement submission = %v, want nil", res)
	}
	if n := seq.inFlightTxs.len(); n != 0 {
		t.Errorf("inFlightTxs.len() = %d, want 0 after both calls returned", n)
	}
}

func TestPublishTransactionReleasesNonceAfterCancel(t *testing.T) {
	engine := newTestRecorderEngine(t, 0)
	config := DefaultSequencerConfig
	configFetcher := func() *SequencerConfig { return &config }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	tx := types.MustSignNewTx(key, types.LatestSigner(engine.bc.Config()), &types.DynamicFeeTx{
		ChainID:   engine.bc.Config().ChainID,
		Nonce:     0,
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(100),
		Gas:       21000,
	})

	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() { firstResult <- seq.PublishTransaction(ctx, tx, nil) }()
	waitForQueuedTxCount(t, seq, 1)
	cancel()
	if res := awaitResult(t, firstResult, "canceled submission"); !errors.Is(res, context.Canceled) {
		t.Errorf("canceled submission = %v, want context.Canceled", res)
	}
	if n := seq.inFlightTxs.len(); n != 0 {
		t.Errorf("inFlightTxs.len() = %d, want 0 after the submission returned", n)
	}
	stale := <-seq.txQueue
	if stale.ctx.Err() == nil {
		t.Error("the stale queue item ctx is not canceled")
	}

	secondResult := make(chan error, 1)
	go func() { secondResult <- seq.PublishTransaction(context.Background(), tx, nil) }()
	waitForQueuedTxCount(t, seq, 1)
	item := <-seq.txQueue
	item.returnResult(nil)
	if res := awaitResult(t, secondResult, "resubmission"); res != nil {
		t.Errorf("resubmission = %v, want nil", res)
	}
}

func TestPublishTransactionWithoutRejectNoopReplacements(t *testing.T) {
	ctx := context.Background()
	engine := newTestRecorderEngine(t, 0)
	config := DefaultSequencerConfig
	config.RejectNoopReplacements = false
	configFetcher := func() *SequencerConfig { return &config }
	seq, err := NewSequencer(engine, nil, configFetcher, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	tx := types.MustSignNewTx(key, types.LatestSigner(engine.bc.Config()), &types.DynamicFeeTx{
		ChainID:   engine.bc.Config().ChainID,
		Nonce:     0,
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(100),
		Gas:       21000,
	})

	results := make(chan error, 2)
	for range 2 {
		go func() { results <- seq.PublishTransaction(ctx, tx, nil) }()
	}
	waitForQueuedTxCount(t, seq, 2)
	if n := seq.inFlightTxs.len(); n != 0 {
		t.Errorf("inFlightTxs.len() = %d, want 0 with noop replacement rejection disabled", n)
	}
	for range 2 {
		item := <-seq.txQueue
		item.returnResult(nil)
	}
	for range 2 {
		if res := awaitResult(t, results, "submission"); res != nil {
			t.Errorf("submission = %v, want nil", res)
		}
	}
}
