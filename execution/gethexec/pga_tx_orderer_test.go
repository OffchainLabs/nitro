// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// testPGARoundLength is the round length pgaTestConfigFetcher yields: 250ms block / 2 rounds.
const testPGARoundLength = 125 * time.Millisecond

func pgaTestConfigFetcher() *SequencerConfig {
	c := DefaultSequencerConfig
	c.MaxBlockSpeed = 250 * time.Millisecond
	c.ExperimentalPGA.RoundsPerBlock = 2
	return &c
}

// makePGAQueueItem builds a queue item whose PGA priority is gasTipCap: the fee cap sits far
// enough above the test basefee that the tip is never cap-bound.
func makePGAQueueItem(t *testing.T, nonce uint64, gasTipCap int64) txQueueItem {
	t.Helper()
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasFeeCap: big.NewInt(testBaseFee + 1000),
		GasTipCap: big.NewInt(gasTipCap),
		Gas:       21000,
	})
	return newRegularTxQueueItem(context.Background(), tx, nil, make(chan error, 1), false, 0)
}

func newTestPGATxOrderer(seq txOrdererSequencer) *PGATxOrderer {
	return NewPGATxOrderer(context.Background(), seq, pgaTestConfigFetcher, big.NewInt(testBaseFee))
}

// TakeRemaining runs in a deferred cleanup that can fire before StartBlock arms the mempool; it
// must report nothing instead of panicking.
func TestPGATxOrdererTakeRemainingBeforeStartBlock(t *testing.T) {
	o := newTestPGATxOrderer(&stubOrdererSequencer{})
	if remaining := o.TakeRemaining(); len(remaining) != 0 {
		t.Fatalf("TakeRemaining before StartBlock returned %d items, want 0", len(remaining))
	}
}

func TestPGATxOrdererStartBlockEmpty(t *testing.T) {
	o := newTestPGATxOrderer(&stubOrdererSequencer{})
	if o.StartBlock() {
		t.Fatal("StartBlock on empty = true, want false")
	}
}

func TestPGATxOrdererYieldsByPriority(t *testing.T) {
	items := []txQueueItem{
		makePGAQueueItem(t, 0, 5),
		makePGAQueueItem(t, 1, 20),
		makePGAQueueItem(t, 2, 10),
	}
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: items})

	if !o.StartBlock() {
		t.Fatal("StartBlock = false, want true")
	}
	for _, wantNonce := range []uint64{1, 2, 0} { // by tip: 20, 10, 5
		item, ok := o.NextQueueItem()
		if !ok || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
		}
	}
}

// A requeued tx re-enters the next block's auction with its accumulated boost folded into the
// priority and carried on the yielded item.
func TestPGATxOrdererStartBlockRestoresBoost(t *testing.T) {
	boosted := makePGAQueueItem(t, 0, 10)
	boosted.pgaBoost = 25 // as left by TakeRemaining in a previous block
	plain := makePGAQueueItem(t, 1, 20)
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{boosted, plain}})

	if !o.StartBlock() {
		t.Fatal("StartBlock = false, want true")
	}
	// boosted outranks plain: tip 10 + boost 25 > tip 20.
	first, ok := o.NextQueueItem()
	if !ok || first.tx.Nonce() != 0 {
		t.Fatalf("first yield = (nonce %d, %v), want the boosted tx", first.tx.Nonce(), ok)
	}
	if first.pgaBoost != 25 {
		t.Fatalf("yielded boost = %d, want 25 preserved", first.pgaBoost)
	}
}

// A mid-block round that expires advances to the next round: the still-queued txs earn the
// anti-starvation boost and keep flowing.
func TestPGATxOrdererRoundExpiryAdvancesAndBoosts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		included := makePGAQueueItem(t, 0, 100)
		leftover := makePGAQueueItem(t, 1, 0)
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{included, leftover}})

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem()
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion()

		// Round 1 expires with the leftover still queued: NextQueueItem advances to round 2 and
		// yields it, boosted by includedPriority / (2 * roundsPerBlock) = 100 / 4.
		time.Sleep(testPGARoundLength + time.Millisecond)
		item, ok = o.NextQueueItem()
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the boosted leftover", item.tx.Nonce(), ok)
		}
		if item.pgaBoost != 25 {
			t.Fatalf("leftover boost = %d, want 25", item.pgaBoost)
		}
	})
}

// When the last round expires the block ends, and the never-yielded txs keep their accumulated
// boost for the next block.
func TestPGATxOrdererLastRoundExpiryEndsBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		included := makePGAQueueItem(t, 0, 100)
		low := makePGAQueueItem(t, 1, 0)
		mid := makePGAQueueItem(t, 2, 5)
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{included, low, mid}})

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion()

		// Round 1 expires: advance to round 2, boosting both leftovers by 100 / 4 = 25.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 2 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the mid-tip tx", item.tx.Nonce(), ok)
		}

		// Round 2 is the last: when it expires the block is over, even with a tx still queued.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if _, ok := o.NextQueueItem(); ok {
			t.Fatal("NextQueueItem yielded after the last round expired, want end of block")
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %d items, want just the low-tip tx", len(remaining))
		}
		if remaining[0].pgaBoost != 25 {
			t.Fatalf("leftover boost = %d, want 25", remaining[0].pgaBoost)
		}
	})
}

// When the queue empties mid-round, NextQueueItem waits out the round boundary and drains the
// txs that arrived in the meantime; after the last round it reports the end of the block.
func TestPGATxOrdererAdvancesRoundWhenQueueEmpties(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := newTestPGATxOrderer(seq)

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// A new tx arrives mid-round; the empty queue makes NextQueueItem wait for the round
		// boundary and drain it.
		seq.items = []txQueueItem{makePGAQueueItem(t, 1, 10)}
		start := time.Now()
		item, ok := o.NextQueueItem()
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("second yield = (nonce %d, %v), want the round-2 tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != testPGARoundLength {
			t.Fatalf("NextQueueItem waited %v, want the round boundary at %v", waited, testPGARoundLength)
		}

		// Round 2 is the last: once the queue empties again the block is over, immediately.
		start = time.Now()
		if _, ok := o.NextQueueItem(); ok {
			t.Fatal("NextQueueItem yielded past the last round, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v on the last round, want immediate return", waited)
		}
	})
}
