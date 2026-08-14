// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math"
	"math/big"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// testPGARoundLength is the round length newTestPGATxOrderer uses: 250ms block / 2 rounds.
const testPGARoundLength = 125 * time.Millisecond

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

// makeExpiredPGAQueueItem builds a queue item whose submission context is already cancelled,
// returning its result channel.
func makeExpiredPGAQueueItem(t *testing.T, nonce uint64, gasTipCap int64) (txQueueItem, chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasFeeCap: big.NewInt(testBaseFee + 1000),
		GasTipCap: big.NewInt(gasTipCap),
		Gas:       21000,
	})
	resultChan := make(chan error, 1)
	return newRegularTxQueueItem(ctx, tx, nil, resultChan, false, 0), resultChan
}

func newTestPGATxOrderer(seq txOrdererSequencer) *pgaTxOrderer {
	return NewPGATxOrderer(context.Background(), seq, 2, testPGARoundLength, big.NewInt(testBaseFee))
}

// newTestPGATxOrdererWithRounds builds an orderer for a 300ms block split into the given number
// of rounds.
func newTestPGATxOrdererWithRounds(seq txOrdererSequencer, rounds uint) *pgaTxOrderer {
	// Test round counts are tiny; the conversion cannot overflow.
	// #nosec G115
	return NewPGATxOrderer(context.Background(), seq, rounds, 300*time.Millisecond/time.Duration(rounds), big.NewInt(testBaseFee))
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
	if o.StartBlock(nil) {
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

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	for _, wantNonce := range []uint64{1, 2, 0} { // by tip: 20, 10, 5
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
		}
	}
}

// A requeued tx re-enters the next block's auction with its accumulated boost folded into the
// priority and carried on the yielded item.
func TestPGATxOrdererStartBlockRestoresBoost(t *testing.T) {
	boosted := makePGAQueueItem(t, 0, 10)
	boosted.AddBoost(25) // accumulated in a previous block
	plain := makePGAQueueItem(t, 1, 20)
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{boosted, plain}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	// boosted outranks plain: tip 10 + boost 25 > tip 20.
	first, ok := o.NextQueueItem(nil, math.MaxInt)
	if !ok || first.tx.Nonce() != 0 {
		t.Fatalf("first yield = (nonce %d, %v), want the boosted tx", first.tx.Nonce(), ok)
	}
	if first.GetPriority() != 35 {
		t.Fatalf("yielded priority = %d, want 35 (tip 10 + boost 25)", first.GetPriority())
	}
}

// A popped tx too big for the remaining block space ends the block, even when a smaller tx
// that would fit is still queued: both leave through TakeRemaining in priority order.
func TestPGATxOrdererOversizedTxEndsBlock(t *testing.T) {
	oversized := makePGAQueueItem(t, 0, 20)
	oversized.txSize = 11
	fits := makePGAQueueItem(t, 1, 5)
	fits.txSize = 10
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{oversized, fits}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if item, ok := o.NextQueueItem(nil, 10); ok {
		t.Fatalf("NextQueueItem yielded nonce %d, want end of block on the oversized tx", item.tx.Nonce())
	}
	if got := queueItemNonces(o.TakeRemaining()); !slices.Equal(got, []uint64{0, 1}) {
		t.Fatalf("TakeRemaining nonces = %v, want [0 1] (oversized tx re-queued at its priority)", got)
	}
}

// The oversized push-back keeps the tx's accumulated boost, so it re-enters the next block's
// auction where it left off.
func TestPGATxOrdererOversizedTxKeepsBoost(t *testing.T) {
	oversized := makePGAQueueItem(t, 0, 10)
	oversized.AddBoost(25) // accumulated in a previous block
	oversized.txSize = 11
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{oversized}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if _, ok := o.NextQueueItem(nil, 10); ok {
		t.Fatal("NextQueueItem yielded, want end of block on the oversized tx")
	}
	remaining := o.TakeRemaining()
	if len(remaining) != 1 || remaining[0].tx.Nonce() != 0 {
		t.Fatalf("TakeRemaining returned %d items, want just the oversized tx", len(remaining))
	}
	if remaining[0].GetPriority() != 35 {
		t.Fatalf("re-queued priority = %d, want 35 (tip 10 + boost 25 preserved)", remaining[0].GetPriority())
	}
}

// A tx that exactly fills the remaining block space is yielded: only a strictly bigger tx ends
// the block.
func TestPGATxOrdererExactFitTxYielded(t *testing.T) {
	item := makePGAQueueItem(t, 0, 10)
	item.txSize = 10
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{item}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	if got, ok := o.NextQueueItem(nil, 10); !ok || got.tx.Nonce() != 0 {
		t.Fatalf("yield = (nonce %d, %v), want the exact-fit tx", got.tx.Nonce(), ok)
	}
}

// A mid-block round that expires advances to the next round: the still-queued txs earn the
// anti-starvation boost and keep flowing.
func TestPGATxOrdererRoundExpiryAdvancesAndBoosts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		included := makePGAQueueItem(t, 0, 100)
		leftover := makePGAQueueItem(t, 1, 0)
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{included, leftover}})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		// Round 1 expires with the leftover still queued: NextQueueItem advances to round 2 and
		// yields it, boosted by includedPriority / (2 * roundsPerBlock) = 100 / 4.
		time.Sleep(testPGARoundLength + time.Millisecond)
		item, ok = o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the boosted leftover", item.tx.Nonce(), ok)
		}
		if item.GetPriority() != 25 {
			t.Fatalf("leftover priority = %d, want 25 (tip 0 + boost 25)", item.GetPriority())
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

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		// Round 1 expires: advance to round 2, boosting both leftovers by 100 / 4 = 25.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 2 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the mid-tip tx", item.tx.Nonce(), ok)
		}

		// Round 2 is the last: when it expires the block is over, even with a tx still queued.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded after the last round expired, want end of block")
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %d items, want just the low-tip tx", len(remaining))
		}
		if remaining[0].GetPriority() != 25 {
			t.Fatalf("leftover priority = %d, want 25 (tip 0 + boost 25)", remaining[0].GetPriority())
		}
	})
}

// When the queue empties mid-round, NextQueueItem waits out the round boundary and drains the
// txs that arrived in the meantime; after the last round it reports the end of the block.
func TestPGATxOrdererAdvancesRoundWhenQueueEmpties(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := newTestPGATxOrderer(seq)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// A new tx arrives mid-round; the empty queue makes NextQueueItem wait for the round
		// boundary and drain it.
		seq.items = []txQueueItem{makePGAQueueItem(t, 1, 10)}
		start := time.Now()
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("second yield = (nonce %d, %v), want the round-2 tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != testPGARoundLength {
			t.Fatalf("NextQueueItem waited %v, want the round boundary at %v", waited, testPGARoundLength)
		}

		// Round 2 is the last: once the queue empties again the block is over, immediately.
		start = time.Now()
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded past the last round, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v on the last round, want immediate return", waited)
		}
	})
}

// A cancelled sequencer context aborts the round wait and ends the block immediately.
func TestPGATxOrdererCtxCancelEndsBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := NewPGATxOrderer(ctx, seq, 2, testPGARoundLength, big.NewInt(testBaseFee))

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// The empty queue would normally wait out round 1; cancellation aborts the wait instead.
		cancel()
		start := time.Now()
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded after context cancellation, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v after cancellation, want immediate return", waited)
		}
	})
}

// An idle round does not end the block: NextQueueItem keeps waiting out round boundaries and
// picks up txs arriving in any later round.
func TestPGATxOrdererWaitsThroughIdleRounds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const roundLength = 100 * time.Millisecond // 300ms block / 3 rounds
		seq := &stubOrdererSequencer{
			items:   []txQueueItem{makePGAQueueItem(t, 0, 10)},
			batches: [][]txQueueItem{nil, {makePGAQueueItem(t, 1, 10)}}, // round 2 idle, round 3 delivers
		}
		o := newTestPGATxOrdererWithRounds(seq, 3)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// Round 2's drain is empty; the orderer waits through it and yields round 3's arrival.
		start := time.Now()
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("yield = (nonce %d, %v), want the round-3 tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != 2*roundLength {
			t.Fatalf("NextQueueItem waited %v, want two round boundaries at %v", waited, 2*roundLength)
		}
	})
}

// With no arrivals at all, the block still runs out its full schedule and ends via the last
// round rather than the first empty one.
func TestPGATxOrdererEndsBlockAfterIdleRounds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const roundLength = 100 * time.Millisecond // 300ms block / 3 rounds
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := newTestPGATxOrdererWithRounds(seq, 3)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		start := time.Now()
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded with nothing queued, want end of block")
		}
		if waited := time.Since(start); waited != 2*roundLength {
			t.Fatalf("NextQueueItem waited %v, want the remaining rounds at %v", waited, 2*roundLength)
		}
	})
}

// Boost derives only from included txs: a yielded-but-never-included tx contributes nothing.
func TestPGATxOrdererNoBoostWithoutInclusion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		items := []txQueueItem{
			makePGAQueueItem(t, 0, 100),
			makePGAQueueItem(t, 1, 0),
		}
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: items})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		// The high-tip tx is yielded but OnTxInclusion is never called (as if the hooks failed it).
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}

		time.Sleep(testPGARoundLength + time.Millisecond)
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the leftover", item.tx.Nonce(), ok)
		}
		if item.GetPriority() != 0 {
			t.Fatalf("leftover priority = %d, want 0 (no boost without an inclusion)", item.GetPriority())
		}
	})
}

// When every drained tx is rejected at push (fee cap below basefee), StartBlock reports no work
// and each tx gets its error.
func TestPGATxOrdererStartBlockAllFeeCapRejected(t *testing.T) {
	first, firstResult := makeTestQueueItem(t, 0, testBaseFee-1)
	second, secondResult := makeTestQueueItem(t, 1, testBaseFee-1)
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{first, second}})

	if o.StartBlock(nil) {
		t.Fatal("StartBlock = true with only rejected txs, want false")
	}
	for i, resultChan := range []chan error{firstResult, secondResult} {
		select {
		case err := <-resultChan:
			if !errors.Is(err, core.ErrFeeCapTooLow) {
				t.Fatalf("tx %d result = %v, want fee-cap-too-low", i, err)
			}
		default:
			t.Fatalf("tx %d got no result, want fee-cap-too-low", i)
		}
	}
}

// A tx included in the last round still boosts the leftovers before the block ends, so they
// carry it into the next block's auction.
func TestPGATxOrdererLastRoundInclusionBoostsLeftovers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		items := []txQueueItem{
			makePGAQueueItem(t, 0, 100),
			makePGAQueueItem(t, 1, 0),
			makePGAQueueItem(t, 2, 5),
		}
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: items})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		// Round 1 expires: both leftovers gain 100 / 4 = 25; the mid tip is yielded and included
		// at priority 5 + 25 = 30.
		time.Sleep(testPGARoundLength + time.Millisecond)
		item, ok = o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 2 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the mid-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		// The last round expires: the block ends, but not before the round-2 inclusion boosts the
		// leftover by 30 / 4 = 7 on top of its earlier 25.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded after the last round expired, want end of block")
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %v, want just the low-tip tx", queueItemNonces(remaining))
		}
		if remaining[0].GetPriority() != 32 {
			t.Fatalf("leftover priority = %d, want 25 + 7 = 32 (tip 0)", remaining[0].GetPriority())
		}
	})
}

// A single-round block never waits: once its round expires the block is over, and the leftover
// boost uses the 2*roundsPerBlock divisor.
func TestPGATxOrdererSingleRoundPerBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const roundLength = 300 * time.Millisecond // 300ms block / 1 round
		items := []txQueueItem{
			makePGAQueueItem(t, 0, 100),
			makePGAQueueItem(t, 1, 0),
		}
		seq := &stubOrdererSequencer{items: items}
		o := newTestPGATxOrdererWithRounds(seq, 1)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		// The only round is the last: expiry ends the block immediately, with the leftover
		// boosted by 100 / 2.
		time.Sleep(roundLength + time.Millisecond)
		start := time.Now()
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded after the only round expired, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v on a single-round block, want immediate return", waited)
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %v, want just the low-tip tx", queueItemNonces(remaining))
		}
		if remaining[0].GetPriority() != 50 {
			t.Fatalf("leftover priority = %d, want 100 / 2 = 50 (tip 0)", remaining[0].GetPriority())
		}
	})
}

// Entries dropped as expired do not end the block: the orderer treats the emptied queue like an
// idle round and keeps waiting for arrivals.
func TestPGATxOrdererExpiredEntriesDontEndBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expired, expiredResult := makeExpiredPGAQueueItem(t, 0, 10)
		seq := &stubOrdererSequencer{
			items:   []txQueueItem{expired},
			batches: [][]txQueueItem{{makePGAQueueItem(t, 1, 10)}},
		}
		o := newTestPGATxOrderer(seq)

		// The expired entry still counts as work: it is only dropped lazily at Pop.
		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}

		// Pop drops the expired entry; the orderer waits out round 1 and yields round 2's arrival.
		start := time.Now()
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("yield = (nonce %d, %v), want the round-2 tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != testPGARoundLength {
			t.Fatalf("NextQueueItem waited %v, want the round boundary at %v", waited, testPGARoundLength)
		}
		select {
		case err := <-expiredResult:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expired tx result = %v, want context.Canceled", err)
			}
		default:
			t.Fatal("expired tx got no result, want context.Canceled")
		}
	})
}

// A tx revived from the nonce-failure cache joins the auction mid-round: it is yielded without
// waiting for a round boundary, entering fresh with no boost.
func TestPGATxOrdererNonceGapResolvedJoinsCurrentRound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 100)}})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion(item)

		o.OnNonceGapResolved(makePGAQueueItem(t, 1, 10))

		start := time.Now()
		item, ok = o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("yield = (nonce %d, %v), want the revived tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v for the revived tx, want a mid-round yield", waited)
		}
		if item.GetPriority() != 10 {
			t.Fatalf("revived priority = %d, want the bare tip 10 (re-enters fresh, no boost)", item.GetPriority())
		}
	})
}

// A revived tx competes in the current round by fee, slotting between the queued leftovers.
func TestPGATxOrdererNonceGapResolvedCompetesByFee(t *testing.T) {
	items := []txQueueItem{
		makePGAQueueItem(t, 0, 20),
		makePGAQueueItem(t, 1, 5),
	}
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: items})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	o.OnNonceGapResolved(makePGAQueueItem(t, 2, 10))

	for _, wantNonce := range []uint64{0, 2, 1} { // by tip: 20, 10, 5
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != wantNonce {
			t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
		}
	}
}

// A revived tx that is never yielded ends the block in TakeRemaining, re-queued like any other
// leftover.
func TestPGATxOrdererNonceGapResolvedLeftoverInTakeRemaining(t *testing.T) {
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	o.OnNonceGapResolved(makePGAQueueItem(t, 1, 5))

	remaining := o.TakeRemaining()
	if got := queueItemNonces(remaining); !slices.Equal(got, []uint64{0, 1}) {
		t.Fatalf("TakeRemaining nonces = %v, want [0 1]", got)
	}
}

// A revived tx whose fee cap sits below the block basefee is rejected at push, reporting the
// fee-cap error to its submitter.
func TestPGATxOrdererNonceGapResolvedFeeCapRejected(t *testing.T) {
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}})

	if !o.StartBlock(nil) {
		t.Fatal("StartBlock = false, want true")
	}
	revived, revivedResult := makeTestQueueItem(t, 1, testBaseFee-1)
	o.OnNonceGapResolved(revived)

	select {
	case err := <-revivedResult:
		if !errors.Is(err, core.ErrFeeCapTooLow) {
			t.Fatalf("revived tx result = %v, want fee-cap-too-low", err)
		}
	default:
		t.Fatal("revived tx got no result, want fee-cap-too-low")
	}
}

// A later-round drain whose txs are all rejected reports each error and keeps the block open for
// the rounds that remain.
func TestPGATxOrdererMidBlockRejectedDrainKeepsWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const roundLength = 100 * time.Millisecond // 300ms block / 3 rounds
		rejected, rejectedResult := makeTestQueueItem(t, 1, testBaseFee-1)
		seq := &stubOrdererSequencer{
			items:   []txQueueItem{makePGAQueueItem(t, 0, 10)},
			batches: [][]txQueueItem{{rejected}, {makePGAQueueItem(t, 2, 10)}},
		}
		o := newTestPGATxOrdererWithRounds(seq, 3)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// Round 2's only arrival is rejected at push; the orderer reports it and waits for round 3.
		start := time.Now()
		item, ok := o.NextQueueItem(nil, math.MaxInt)
		if !ok || item.tx.Nonce() != 2 {
			t.Fatalf("yield = (nonce %d, %v), want the round-3 tx", item.tx.Nonce(), ok)
		}
		if waited := time.Since(start); waited != 2*roundLength {
			t.Fatalf("NextQueueItem waited %v, want two round boundaries at %v", waited, 2*roundLength)
		}
		select {
		case err := <-rejectedResult:
			if !errors.Is(err, core.ErrFeeCapTooLow) {
				t.Fatalf("rejected tx result = %v, want fee-cap-too-low", err)
			}
		default:
			t.Fatal("rejected tx got no result, want fee-cap-too-low")
		}
	})
}

func TestPGATxOrdererBlockIntervalAllRounds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := newTestPGATxOrdererWithRounds(seq, 3)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded with nothing queued, want end of block")
		}
		if got := o.BlockInterval(); got != 300*time.Millisecond {
			t.Fatalf("BlockInterval = %v, want the full block time 300ms", got)
		}
	})
}

func TestPGATxOrdererBlockIntervalPartialBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const roundLength = 60 * time.Millisecond // 300ms block / 5 rounds
		oversized := makePGAQueueItem(t, 2, 10)
		oversized.txSize = 11
		seq := &stubOrdererSequencer{
			items:   []txQueueItem{makePGAQueueItem(t, 0, 10)},
			batches: [][]txQueueItem{{makePGAQueueItem(t, 1, 10)}, {oversized}},
		}
		o := newTestPGATxOrdererWithRounds(seq, 5)

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		// Rounds 1 and 2 each yield their drained tx; round 3's arrival doesn't fit and ends the block.
		for _, wantNonce := range []uint64{0, 1} {
			if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != wantNonce {
				t.Fatalf("yield = (nonce %d, %v), want nonce %d", item.tx.Nonce(), ok, wantNonce)
			}
		}
		if item, ok := o.NextQueueItem(nil, 10); ok {
			t.Fatalf("NextQueueItem yielded nonce %d, want end of block on the oversized tx", item.tx.Nonce())
		}
		if got := o.BlockInterval(); got != 3*roundLength {
			t.Fatalf("BlockInterval = %v, want three rounds at %v", got, 3*roundLength)
		}
	})
}

func TestPGATxOrdererBlockIntervalPartialFirstRound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		oversized := makePGAQueueItem(t, 0, 10)
		oversized.txSize = 11
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{oversized}})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if _, ok := o.NextQueueItem(nil, 10); ok {
			t.Fatal("NextQueueItem yielded, want end of block on the oversized tx")
		}
		if got := o.BlockInterval(); got != testPGARoundLength {
			t.Fatalf("BlockInterval = %v, want one round at %v", got, testPGARoundLength)
		}
	})
}

func TestPGATxOrdererBlockIntervalOverrun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}})

		if !o.StartBlock(nil) {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(nil, math.MaxInt); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		time.Sleep(500 * time.Millisecond)
		start := time.Now()
		if _, ok := o.NextQueueItem(nil, math.MaxInt); ok {
			t.Fatal("NextQueueItem yielded past the schedule, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v after the overrun, want immediate return", waited)
		}
		if got := o.BlockInterval(); got != 250*time.Millisecond {
			t.Fatalf("BlockInterval = %v, want MaxBlockSpeed (250ms) despite the overrun", got)
		}
	})
}
