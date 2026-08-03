// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"slices"
	"testing"
)

// stubOrdererSequencer feeds drainValidatedTxs canned results: items first, then one batch per
// later call.
type stubOrdererSequencer struct {
	items   []txQueueItem
	batches [][]txQueueItem
}

func (s *stubOrdererSequencer) drainValidatedTxs() []txQueueItem {
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
	o := newFIFOTxOrderer(&stubOrdererSequencer{})
	if o.StartBlock() {
		t.Fatal("StartBlock on empty = true, want false")
	}
	if _, yielded := o.NextQueueItem(); yielded {
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
	o := newFIFOTxOrderer(&stubOrdererSequencer{items: items})

	if !o.StartBlock() {
		t.Fatal("StartBlock = false, want true")
	}
	first, ok := o.NextQueueItem()
	if !ok || first.tx.Nonce() != 0 {
		t.Fatalf("first yield = (nonce %d, %v), want nonce 0", first.tx.Nonce(), ok)
	}
	second, ok := o.NextQueueItem()
	if !ok || second.tx.Nonce() != 1 {
		t.Fatalf("second yield = (nonce %d, %v), want nonce 1", second.tx.Nonce(), ok)
	}

	remaining := o.TakeRemaining()
	if got := queueItemNonces(remaining); !slices.Equal(got, []uint64{2, 3}) {
		t.Fatalf("TakeRemaining nonces = %v, want [2 3] (the never-yielded tail)", got)
	}
	if _, yielded := o.NextQueueItem(); yielded {
		t.Fatal("NextQueueItem yielded after TakeRemaining")
	}
	if leftover := o.TakeRemaining(); len(leftover) != 0 {
		t.Fatalf("second TakeRemaining returned %d items, want 0", len(leftover))
	}
}
