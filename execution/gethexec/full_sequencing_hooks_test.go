// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/core"
)

// spyTxOrderer counts OnTxInclusion calls; the hooks use no other txOrderer method.
type spyTxOrderer struct {
	inclusions int
}

func (s *spyTxOrderer) StartBlock() bool                   { return false }
func (s *spyTxOrderer) NextQueueItem() (txQueueItem, bool) { return txQueueItem{}, false }
func (s *spyTxOrderer) TakeRemaining() []txQueueItem       { return nil }
func (s *spyTxOrderer) OnTxInclusion()                     { s.inclusions++ }

// TestFullSequencingHooksNotifyOrdererOnSuccessOnly covers the boost feedback wiring: only
// TxSucceeded notifies the orderer of an inclusion; a failed tx must not earn PGA boost credit.
func TestFullSequencingHooksNotifyOrdererOnSuccessOnly(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	orderer := &spyTxOrderer{}
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1}}, math.MaxInt, nil, nil, orderer)

	if tx, _, err := hooks.NextTxToSequence(); err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxSucceeded()
	if orderer.inclusions != 1 {
		t.Fatalf("inclusions after TxSucceeded = %d, want 1", orderer.inclusions)
	}

	if tx, _, err := hooks.NextTxToSequence(); err != nil || tx == nil {
		t.Fatalf("second NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxFailed(errors.New("intrinsic gas too low"))
	if orderer.inclusions != 1 {
		t.Fatalf("inclusions after TxFailed = %d, want still 1", orderer.inclusions)
	}
}

// TestFullSequencingHooksTxResultLifecycle covers the per-tx result bookkeeping: a pulled tx
// starts as txNotFinalized, TxSucceeded clears the marker, TxFailed sets the real error, and a
// tx the block processor never reports keeps the marker in SequencedTxes.
func TestFullSequencingHooksTxResultLifecycle(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	item2, _ := makeTestQueueItem(t, 2, testBaseFee)
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1, item2}}, math.MaxInt, nil, nil, nil)

	pull := func() {
		t.Helper()
		tx, _, err := hooks.NextTxToSequence()
		if err != nil || tx == nil {
			t.Fatalf("NextTxToSequence = (%v, %v), want a tx", tx, err)
		}
	}

	failure := errors.New("intrinsic gas too low")
	pull()
	hooks.TxSucceeded()
	pull()
	hooks.TxFailed(failure)
	pull() // never finalized

	results := hooks.SequencedTxes()
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("succeeded tx has err %v, want nil", results[0].Err)
	}
	if !errors.Is(results[1].Err, failure) {
		t.Errorf("failed tx has err %v, want %v", results[1].Err, failure)
	}
	if !errors.Is(results[2].Err, txNotFinalized) {
		t.Errorf("unreported tx has err %v, want txNotFinalized", results[2].Err)
	}
}

// TestFullSequencingHooksLateFailureOverridesSuccess covers the double-report guard: a failure
// reported after a success replaces it (a group rollback fails already-succeeded txs), while a
// success never overrides a failure.
func TestFullSequencingHooksLateFailureOverridesSuccess(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1}}, math.MaxInt, nil, nil, nil)

	rollback := errors.New("group rolled back")
	if tx, _, err := hooks.NextTxToSequence(); err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxSucceeded()
	hooks.TxFailed(rollback)

	failure := errors.New("intrinsic gas too low")
	if tx, _, err := hooks.NextTxToSequence(); err != nil || tx == nil {
		t.Fatalf("second NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxFailed(failure)
	hooks.TxSucceeded()

	results := hooks.SequencedTxes()
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if !errors.Is(results[0].Err, rollback) {
		t.Errorf("rolled-back tx has err %v, want %v", results[0].Err, rollback)
	}
	if !errors.Is(results[1].Err, failure) {
		t.Errorf("failed tx has err %v, want %v (success must not override)", results[1].Err, failure)
	}
}

// TestFullSequencingHooksFailedTxDoesNotConsumeBudget covers the size accounting: only
// successful txs count against maxSequencedTxsSize, so a failed tx must not shrink the budget.
func TestFullSequencingHooksFailedTxDoesNotConsumeBudget(t *testing.T) {
	failed, _ := makeTestQueueItem(t, 0, testBaseFee)
	failed.txSize = 20
	fits, _ := makeTestQueueItem(t, 1, testBaseFee)
	fits.txSize = 20
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{failed, fits}}, 25, nil, nil, nil)

	tx, _, err := hooks.NextTxToSequence()
	if err != nil || tx.Nonce() != 0 {
		t.Fatalf("first NextTxToSequence = (%v, %v), want nonce 0", tx, err)
	}
	hooks.TxFailed(errors.New("intrinsic gas too low"))

	// The failed tx's 20 bytes must not count against the 25-byte budget.
	tx, _, err = hooks.NextTxToSequence()
	if err != nil || tx == nil || tx.Nonce() != 1 {
		t.Fatalf("second NextTxToSequence = (%v, %v), want nonce 1", tx, err)
	}
}

// TestFullSequencingHooksFailsOnUnreportedResult covers the fail-closed check: pulling the
// next tx before the block processor reported the previous one's result returns an error.
func TestFullSequencingHooksFailsOnUnreportedResult(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1}}, math.MaxInt, nil, nil, nil)

	tx, _, err := hooks.NextTxToSequence()
	if err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	// The pulled tx's result is never reported.
	tx, _, err = hooks.NextTxToSequence()
	if err == nil {
		t.Fatalf("second NextTxToSequence = (%v, nil), want an error", tx)
	}
}

// TestFullSequencingHooksSkipsOversizedTx covers the size budget: a tx that would exceed
// maxSequencedTxsSize is failed with ErrGasLimitReached and the next candidate is yielded.
func TestFullSequencingHooksSkipsOversizedTx(t *testing.T) {
	small, _ := makeTestQueueItem(t, 0, testBaseFee)
	small.txSize = 10
	big, _ := makeTestQueueItem(t, 1, testBaseFee)
	big.txSize = 100
	fits, _ := makeTestQueueItem(t, 2, testBaseFee)
	fits.txSize = 10
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{small, big, fits}}, 25, nil, nil, nil)

	tx, _, err := hooks.NextTxToSequence()
	if err != nil || tx.Nonce() != 0 {
		t.Fatalf("first NextTxToSequence = (%v, %v), want nonce 0", tx, err)
	}
	hooks.TxSucceeded()

	// The oversized tx is skipped; the one that fits is yielded next.
	tx, _, err = hooks.NextTxToSequence()
	if err != nil || tx.Nonce() != 2 {
		t.Fatalf("second NextTxToSequence = (%v, %v), want nonce 2", tx, err)
	}
	hooks.TxSucceeded()

	if !hooks.txSizeLimitReached {
		t.Error("txSizeLimitReached = false, want true")
	}
	results := hooks.SequencedTxes()
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if !errors.Is(results[1].Err, core.ErrGasLimitReached) {
		t.Errorf("oversized tx has err %v, want ErrGasLimitReached", results[1].Err)
	}
}
