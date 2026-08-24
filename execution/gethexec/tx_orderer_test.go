// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// stubOrdererSequencer feeds drainValidatedTxs pre-set results: one batch per call. Like the real
// drain, it returns at most maxQueueItems items, keeping the excess for the next call. It records
// the bound passed to each call in drainLimits.
type stubOrdererSequencer struct {
	batches     [][]txQueueItem
	drainLimits []int
}

func newStubOrdererSequencer(items ...txQueueItem) *stubOrdererSequencer {
	return &stubOrdererSequencer{
		batches: [][]txQueueItem{
			items,
		},
	}
}

func newStubOrdererSequencerWithBatches(batches ...[]txQueueItem) *stubOrdererSequencer {
	return &stubOrdererSequencer{
		batches: batches,
	}
}

func (s *stubOrdererSequencer) drainValidatedTxs(statedb *state.StateDB, baseFee *big.Int, maxQueueItems int) []txQueueItem {
	s.drainLimits = append(s.drainLimits, maxQueueItems)
	if maxQueueItems <= 0 || len(s.batches) == 0 {
		return nil
	}
	drainedTxs := s.batches[0]
	s.batches = s.batches[1:]
	if len(drainedTxs) <= maxQueueItems {
		return drainedTxs
	}
	remainingTxs := drainedTxs[maxQueueItems:]
	drainedTxs = drainedTxs[:maxQueueItems]
	if len(s.batches) == 0 {
		s.batches = append(s.batches, remainingTxs)
	} else {
		s.batches[0] = append(remainingTxs, s.batches[0]...)
	}
	return drainedTxs
}

// newTestFIFOTxOrderer builds a FIFO orderer over the given canned items with no drain bound.
func newTestFIFOTxOrderer(t *testing.T, items ...txQueueItem) *fifoTxOrderer {
	t.Helper()
	return newFIFOTxOrderer(newStubOrdererSequencer(items...), txOrdererConfig{maxBlockTxCandidates: math.MaxInt}, 0)
}

func queueItemNonces(items []txQueueItem) []uint64 {
	out := make([]uint64, len(items))
	for i, it := range items {
		out[i] = it.tx.Nonce()
	}
	return out
}

func TestFIFOTxOrdererStartBlockEmpty(t *testing.T) {
	o := newTestFIFOTxOrderer(t)
	if o.StartBlock(nil) {
		t.Fatal("StartBlock on empty = true, want false")
	}
	if _, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64); reason == fetchedTx {
		t.Fatal("NextQueueItem yielded from an empty block")
	}
	if remaining := o.TakeRemaining(); len(remaining) != 0 {
		t.Fatalf("TakeRemaining returned %d items, want 0", len(remaining))
	}
}

// TestFIFOTxOrdererBlockLifecycle covers one block: StartBlock arms the drained candidates,
// NextQueueItem yields them in order, and TakeRemaining returns the never-yielded tail,
// leaving the orderer empty.
func TestFIFOTxOrdererBlockLifecycle(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(4) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		items = append(items, item)
	}
	o := newTestFIFOTxOrderer(t, items...)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	first, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64)
	if reason != fetchedTx || first.tx.Nonce() != 0 {
		t.Fatalf("first yield = (nonce %d, %v), want nonce 0", first.tx.Nonce(), reason)
	}
	second, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64)
	if reason != fetchedTx || second.tx.Nonce() != 1 {
		t.Fatalf("second yield = (nonce %d, %v), want nonce 1", second.tx.Nonce(), reason)
	}

	remaining := o.TakeRemaining()
	if got := queueItemNonces(remaining); !slices.Equal(got, []uint64{2, 3}) {
		t.Fatalf("TakeRemaining nonces = %v, want [2 3] (the never-yielded tail)", got)
	}
	if _, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64); reason == fetchedTx {
		t.Fatal("NextQueueItem yielded after TakeRemaining")
	}
	if leftover := o.TakeRemaining(); len(leftover) != 0 {
		t.Fatalf("second TakeRemaining returned %d items, want 0", len(leftover))
	}
}

// Candidates too big for the remaining block space don't end the block: NextQueueItem skips
// them and yields the next tx that fits; a tx exactly filling the space still fits.
func TestFIFOTxOrdererSkipsOversizedTxs(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(5) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		item.txSize = 10
		items = append(items, item)
	}
	items[1].txSize = 11 // oversized for the block space below
	items[2].txSize = 11
	o := newTestFIFOTxOrderer(t, items...)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	// Nonce 0 fills the space exactly; nonces 1 and 2 are skipped as oversized.
	for _, wantNonce := range []uint64{0, 3} {
		item, reason := o.NextQueueItem(nil, 10, math.MaxUint64)
		if reason != fetchedTx || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), reason, wantNonce)
		}
	}
	// TakeRemaining returns the never-yielded tail followed by the skipped oversized txs.
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{4, 1, 2}) {
		t.Fatalf("TakeRemaining nonces = %v, want [4 1 2]", got)
	}
}

// With only oversized candidates left, NextQueueItem reports the size limit and TakeRemaining
// recovers the skipped txs exactly once.
func TestFIFOTxOrdererAllOversizedExhausts(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(2) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		item.txSize = 100
		items = append(items, item)
	}
	o := newTestFIFOTxOrderer(t, items...)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if _, reason := o.NextQueueItem(nil, 99, math.MaxUint64); reason != blockSizeLimitReached {
		t.Fatalf("finish reason = %d, want blockSizeLimitReached (skips are not exhaustion)", reason)
	}
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{0, 1}) {
		t.Fatalf("TakeRemaining nonces = %v, want [0 1]", got)
	}
	if leftover := o.TakeRemaining(); len(leftover) != 0 {
		t.Fatalf("second TakeRemaining returned %d items, want 0", len(leftover))
	}
}

// Running out of block gas ends the block: the gas limit is the stop reason, and the
// unsequenced txs leave through TakeRemaining exactly once.
func TestFIFOTxOrdererGasLimitEndsBlock(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(2) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		items = append(items, item)
	}
	o := newTestFIFOTxOrderer(t, items...)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if _, reason := o.NextQueueItem(nil, math.MaxInt, params.TxGas-1); reason != blockGasLimitReached {
		t.Fatalf("finish reason = %d, want blockGasLimitReached with no gas left", reason)
	}
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{0, 1}) {
		t.Fatalf("TakeRemaining nonces = %v, want [0 1] exactly once", got)
	}
}

func TestFIFOTxOrdererPassesDrainBound(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(10) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		items = append(items, item)
	}
	seq := newStubOrdererSequencer(items...)
	o := newFIFOTxOrderer(seq, txOrdererConfig{maxBlockTxCandidates: 7}, 0)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if !slices.Equal(seq.drainLimits, []int{7}) {
		t.Fatalf("drain bounds = %v, want [7]", seq.drainLimits)
	}
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{0, 1, 2, 3, 4, 5, 6}) {
		t.Fatalf("candidates = %v, want the first 7 pending txs", got)
	}
}

func TestFIFOTxOrdererBlockInterval(t *testing.T) {
	blockInterval := 200 * time.Millisecond
	pollInterval := 100 * time.Millisecond
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	config := txOrdererConfig{maxBlockTxCandidates: math.MaxInt, maxBlockSpeed: blockInterval}
	o := newFIFOTxOrderer(newStubOrdererSequencer(item), config, pollInterval)
	if got := o.BlockInterval(); got != pollInterval {
		t.Fatalf("BlockInterval = %v, want %v", got, pollInterval)
	}
	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if got := o.BlockInterval(); got != blockInterval {
		t.Fatalf("BlockInterval = %v, want %v", got, blockInterval)
	}
}

// A revived nonce-gap tx joins the back of the block's candidates, so it can follow its
// predecessor into the same block; if never yielded it leaves through TakeRemaining.
func TestFIFOTxOrdererNonceGapResolvedJoinsCandidates(t *testing.T) {
	drained, _ := makeTestQueueItem(t, 0, testBaseFee)
	o := newTestFIFOTxOrderer(t, drained)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	revived, _ := makeTestQueueItem(t, 1, testBaseFee)
	o.OnNonceGapResolved(revived)
	trailing, _ := makeTestQueueItem(t, 2, testBaseFee)
	o.OnNonceGapResolved(trailing)

	for _, wantNonce := range []uint64{0, 1} {
		if item, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64); reason != fetchedTx || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), reason, wantNonce)
		}
	}
	// The block ends with the second revived tx never yielded: it leaves via TakeRemaining.
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{2}) {
		t.Fatalf("TakeRemaining nonces = %v, want [2]", got)
	}
	if _, reason := o.NextQueueItem(nil, math.MaxInt, math.MaxUint64); reason == fetchedTx {
		t.Fatal("NextQueueItem yielded after TakeRemaining")
	}
}
