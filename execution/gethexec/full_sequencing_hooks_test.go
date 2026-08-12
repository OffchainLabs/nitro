// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/transactionfeed"
)

// recordingBroadcaster records the feed messages broadcast through TxAccepted.
type recordingBroadcaster struct {
	msgs []*transactionfeed.TransactionFeedMessage
}

func (r *recordingBroadcaster) BroadcastTransaction(msg *transactionfeed.TransactionFeedMessage) {
	r.msgs = append(r.msgs, msg)
}

// TestFullSequencingHooksTxAcceptedReportsPGARound covers the feed's round attribution
func TestFullSequencingHooksTxAcceptedReportsPGARound(t *testing.T) {
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.SetPGARound(2)
	feed := &recordingBroadcaster{}
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item}}, math.MaxInt, nil, feed)

	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, EffectiveGasPrice: big.NewInt(testBaseFee)}

	internalTx := types.NewTx(&types.ArbitrumInternalTx{ChainId: big.NewInt(1)})
	hooks.TxAccepted(header, internalTx, receipt)

	tx, _, err := hooks.NextTxToSequence(nil)
	if err != nil || tx == nil {
		t.Fatalf("NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxSucceeded()
	hooks.TxAccepted(header, tx, receipt)

	if len(feed.msgs) != 2 {
		t.Fatalf("broadcast %d messages, want 2", len(feed.msgs))
	}
	if got := feed.msgs[0].PGARound; got != 0 {
		t.Errorf("internal tx pga_round = %d, want 0", got)
	}
	if got := feed.msgs[1].PGARound; got != 2 {
		t.Errorf("user tx pga_round = %d, want 2", got)
	}
}

// spyTxFetcher wraps fixedTxFetcher and records the queue items reported through OnTxInclusion.
type spyTxFetcher struct {
	fixedTxFetcher
	inclusions []txQueueItem
}

func (s *spyTxFetcher) OnTxInclusion(queueItem txQueueItem) {
	s.inclusions = append(s.inclusions, queueItem)
}

// TestFullSequencingHooksNotifyFetcherOnSuccessOnly covers the boost feedback wiring: only
// TxSucceeded notifies the fetcher of an inclusion; a failed tx must not earn PGA boost credit.
func TestFullSequencingHooksNotifyFetcherOnSuccessOnly(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	fetcher := &spyTxFetcher{fixedTxFetcher: fixedTxFetcher{items: []txQueueItem{item0, item1}}}
	hooks := MakeSequencingHooks(fetcher, math.MaxInt, nil, nil)

	if tx, _, err := hooks.NextTxToSequence(nil); err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxSucceeded()
	if len(fetcher.inclusions) != 1 || fetcher.inclusions[0].tx.Nonce() != 0 {
		t.Fatalf("inclusions after TxSucceeded = %v, want just the nonce-0 tx", queueItemNonces(fetcher.inclusions))
	}

	if tx, _, err := hooks.NextTxToSequence(nil); err != nil || tx == nil {
		t.Fatalf("second NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxFailed(errors.New("intrinsic gas too low"))
	if len(fetcher.inclusions) != 1 {
		t.Fatalf("inclusions after TxFailed = %d, want still 1", len(fetcher.inclusions))
	}
}

// TestFullSequencingHooksTxResultLifecycle covers the per-tx result bookkeeping: a pulled tx
// starts as txNotFinalized, TxSucceeded clears the marker, TxFailed sets the real error, and a
// tx the block processor never reports keeps the marker in SequencedTxes.
func TestFullSequencingHooksTxResultLifecycle(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	item2, _ := makeTestQueueItem(t, 2, testBaseFee)
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1, item2}}, math.MaxInt, nil, nil)

	pull := func() {
		t.Helper()
		tx, _, err := hooks.NextTxToSequence(nil)
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
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1}}, math.MaxInt, nil, nil)

	rollback := errors.New("group rolled back")
	if tx, _, err := hooks.NextTxToSequence(nil); err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	hooks.TxSucceeded()
	hooks.TxFailed(rollback)

	failure := errors.New("intrinsic gas too low")
	if tx, _, err := hooks.NextTxToSequence(nil); err != nil || tx == nil {
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
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{failed, fits}}, 25, nil, nil)

	tx, _, err := hooks.NextTxToSequence(nil)
	if err != nil || tx.Nonce() != 0 {
		t.Fatalf("first NextTxToSequence = (%v, %v), want nonce 0", tx, err)
	}
	hooks.TxFailed(errors.New("intrinsic gas too low"))

	// The failed tx's 20 bytes must not count against the 25-byte budget.
	tx, _, err = hooks.NextTxToSequence(nil)
	if err != nil || tx == nil || tx.Nonce() != 1 {
		t.Fatalf("second NextTxToSequence = (%v, %v), want nonce 1", tx, err)
	}
}

// TestFullSequencingHooksFailsOnUnreportedResult covers the fail-closed check: pulling the
// next tx before the block processor reported the previous one's result returns an error.
func TestFullSequencingHooksFailsOnUnreportedResult(t *testing.T) {
	item0, _ := makeTestQueueItem(t, 0, testBaseFee)
	item1, _ := makeTestQueueItem(t, 1, testBaseFee)
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: []txQueueItem{item0, item1}}, math.MaxInt, nil, nil)

	tx, _, err := hooks.NextTxToSequence(nil)
	if err != nil || tx == nil {
		t.Fatalf("first NextTxToSequence = (%v, %v), want a tx", tx, err)
	}
	// The pulled tx's result is never reported.
	tx, _, err = hooks.NextTxToSequence(nil)
	if err == nil {
		t.Fatalf("second NextTxToSequence = (%v, nil), want an error", tx)
	}
}
