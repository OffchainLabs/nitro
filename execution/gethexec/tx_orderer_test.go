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
)

// stubOrdererSequencer feeds drainValidatedTxs canned results: items first, then one batch per
// later call.
type stubOrdererSequencer struct {
	items   []txQueueItem
	batches [][]txQueueItem
}

func (s *stubOrdererSequencer) drainValidatedTxs(statedb *state.StateDB, baseFee *big.Int) []txQueueItem {
	if s.items != nil {
		items := s.items
		s.items = nil
		return items
	}
	if len(s.batches) > 0 {
		batch := s.batches[0]
		s.batches = s.batches[1:]
		return batch
	}
	return nil
}

func queueItemNonces(items []txQueueItem) []uint64 {
	out := make([]uint64, len(items))
	for i, it := range items {
		out[i] = it.tx.Nonce()
	}
	return out
}

func TestFIFOTxOrdererStartBlockEmpty(t *testing.T) {
	o := newFIFOTxOrderer(&stubOrdererSequencer{}, 0, nil)
	if o.StartBlock(nil) {
		t.Fatal("StartBlock on empty = true, want false")
	}
	if _, yielded := o.NextQueueItem(nil, math.MaxInt); yielded {
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
	o := newFIFOTxOrderer(&stubOrdererSequencer{items: items}, 0, nil)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	first, ok := o.NextQueueItem(nil, math.MaxInt)
	if !ok || first.tx.Nonce() != 0 {
		t.Fatalf("first yield = (nonce %d, %v), want nonce 0", first.tx.Nonce(), ok)
	}
	second, ok := o.NextQueueItem(nil, math.MaxInt)
	if !ok || second.tx.Nonce() != 1 {
		t.Fatalf("second yield = (nonce %d, %v), want nonce 1", second.tx.Nonce(), ok)
	}

	remaining := o.TakeRemaining()
	if got := queueItemNonces(remaining); !slices.Equal(got, []uint64{2, 3}) {
		t.Fatalf("TakeRemaining nonces = %v, want [2 3] (the never-yielded tail)", got)
	}
	if _, yielded := o.NextQueueItem(nil, math.MaxInt); yielded {
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
	o := newFIFOTxOrderer(&stubOrdererSequencer{items: items}, 0, nil)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	// Nonce 0 fills the space exactly; nonces 1 and 2 are skipped as oversized.
	for _, wantNonce := range []uint64{0, 3} {
		item, ok := o.NextQueueItem(nil, 10)
		if !ok || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
		}
	}
	// TakeRemaining returns the never-yielded tail followed by the skipped oversized txs.
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{4, 1, 2}) {
		t.Fatalf("TakeRemaining nonces = %v, want [4 1 2]", got)
	}
}

// With only oversized candidates left, NextQueueItem reports exhaustion and TakeRemaining
// recovers the skipped txs exactly once.
func TestFIFOTxOrdererAllOversizedExhausts(t *testing.T) {
	var items []txQueueItem
	for nonce := range uint64(2) {
		item, _ := makeTestQueueItem(t, nonce, testBaseFee)
		item.txSize = 100
		items = append(items, item)
	}
	o := newFIFOTxOrderer(&stubOrdererSequencer{items: items}, 0, nil)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if item, ok := o.NextQueueItem(nil, 99); ok {
		t.Fatalf("NextQueueItem yielded nonce %d, want exhaustion with only oversized txs", item.tx.Nonce())
	}
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{0, 1}) {
		t.Fatalf("TakeRemaining nonces = %v, want [0 1]", got)
	}
	if leftover := o.TakeRemaining(); len(leftover) != 0 {
		t.Fatalf("second TakeRemaining returned %d items, want 0", len(leftover))
	}
}

func TestFIFOTxOrdererBlockInterval(t *testing.T) {
	blockInterval := 200 * time.Millisecond
	o := newFIFOTxOrderer(&stubOrdererSequencer{}, blockInterval, nil)
	if got := o.BlockInterval(); got != blockInterval {
		t.Fatalf("BlockInterval = %v, want %v", got, blockInterval)
	}
}

// A revived nonce-gap tx joins the back of the block's candidates, so it can follow its
// predecessor into the same block; if never yielded it leaves through TakeRemaining.
func TestFIFOTxOrdererNonceGapResolvedJoinsCandidates(t *testing.T) {
	drained, _ := makeTestQueueItem(t, 0, testBaseFee)
	o := newFIFOTxOrderer(&stubOrdererSequencer{items: []txQueueItem{drained}}, 0, nil)

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	revived, _ := makeTestQueueItem(t, 1, testBaseFee)
	o.OnNonceGapResolved(revived)
	trailing, _ := makeTestQueueItem(t, 2, testBaseFee)
	o.OnNonceGapResolved(trailing)

	for _, wantNonce := range []uint64{0, 1} {
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
		}
	}
	// The block ends with the second revived tx never yielded: it leaves via TakeRemaining.
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{2}) {
		t.Fatalf("TakeRemaining nonces = %v, want [2]", got)
	}
	if _, yielded := o.NextQueueItem(nil, math.MaxInt); yielded {
		t.Fatal("NextQueueItem yielded after TakeRemaining")
	}
}
