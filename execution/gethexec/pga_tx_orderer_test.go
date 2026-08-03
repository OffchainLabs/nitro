// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/core"
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

func newTestPGATxOrderer(seq txOrdererSequencer) *PGATxOrderer {
	return NewPGATxOrderer(context.Background(), seq, pgaTestConfigFetcher, big.NewInt(testBaseFee))
}

// pgaConfigFetcherWithRounds yields a 300ms block split into the given number of rounds.
func pgaConfigFetcherWithRounds(rounds uint) SequencerConfigFetcher {
	return func() *SequencerConfig {
		c := DefaultSequencerConfig
		c.MaxBlockSpeed = 300 * time.Millisecond
		c.ExperimentalPGA.RoundsPerBlock = rounds
		return &c
	}
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

// A cancelled sequencer context aborts the round wait and ends the block immediately.
func TestPGATxOrdererCtxCancelEndsBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		seq := &stubOrdererSequencer{items: []txQueueItem{makePGAQueueItem(t, 0, 10)}}
		o := NewPGATxOrderer(ctx, seq, pgaTestConfigFetcher, big.NewInt(testBaseFee))

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// The empty queue would normally wait out round 1; cancellation aborts the wait instead.
		cancel()
		start := time.Now()
		if _, ok := o.NextQueueItem(); ok {
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
		o := NewPGATxOrderer(context.Background(), seq, pgaConfigFetcherWithRounds(3), big.NewInt(testBaseFee))

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// Round 2's drain is empty; the orderer waits through it and yields round 3's arrival.
		start := time.Now()
		item, ok := o.NextQueueItem()
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
		o := NewPGATxOrderer(context.Background(), seq, pgaConfigFetcherWithRounds(3), big.NewInt(testBaseFee))

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		start := time.Now()
		if _, ok := o.NextQueueItem(); ok {
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

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		// The high-tip tx is yielded but OnTxInclusion is never called (as if the hooks failed it).
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}

		time.Sleep(testPGARoundLength + time.Millisecond)
		item, ok := o.NextQueueItem()
		if !ok || item.tx.Nonce() != 1 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the leftover", item.tx.Nonce(), ok)
		}
		if item.pgaBoost != 0 {
			t.Fatalf("leftover boost = %d, want 0 without an inclusion", item.pgaBoost)
		}
	})
}

// When every drained tx is rejected at push (fee cap below basefee), StartBlock reports no work
// and each tx gets its error.
func TestPGATxOrdererStartBlockAllFeeCapRejected(t *testing.T) {
	first, firstResult := makeTestQueueItem(t, 0, testBaseFee-1)
	second, secondResult := makeTestQueueItem(t, 1, testBaseFee-1)
	o := newTestPGATxOrderer(&stubOrdererSequencer{items: []txQueueItem{first, second}})

	if o.StartBlock() {
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

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion()

		// Round 1 expires: both leftovers gain 100 / 4 = 25; the mid tip is yielded and included
		// at priority 5 + 25 = 30.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 2 {
			t.Fatalf("round-2 yield = (nonce %d, %v), want the mid-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion()

		// The last round expires: the block ends, but not before the round-2 inclusion boosts the
		// leftover by 30 / 4 = 7 on top of its earlier 25.
		time.Sleep(testPGARoundLength + time.Millisecond)
		if _, ok := o.NextQueueItem(); ok {
			t.Fatal("NextQueueItem yielded after the last round expired, want end of block")
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %v, want just the low-tip tx", queueItemNonces(remaining))
		}
		if remaining[0].pgaBoost != 32 {
			t.Fatalf("leftover boost = %d, want 25 + 7 = 32", remaining[0].pgaBoost)
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
		o := NewPGATxOrderer(context.Background(), seq, pgaConfigFetcherWithRounds(1), big.NewInt(testBaseFee))

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want the high-tip tx", item.tx.Nonce(), ok)
		}
		o.OnTxInclusion()

		// The only round is the last: expiry ends the block immediately, with the leftover
		// boosted by 100 / 2.
		time.Sleep(roundLength + time.Millisecond)
		start := time.Now()
		if _, ok := o.NextQueueItem(); ok {
			t.Fatal("NextQueueItem yielded after the only round expired, want end of block")
		}
		if waited := time.Since(start); waited != 0 {
			t.Fatalf("NextQueueItem waited %v on a single-round block, want immediate return", waited)
		}
		remaining := o.TakeRemaining()
		if len(remaining) != 1 || remaining[0].tx.Nonce() != 1 {
			t.Fatalf("TakeRemaining returned %v, want just the low-tip tx", queueItemNonces(remaining))
		}
		if remaining[0].pgaBoost != 50 {
			t.Fatalf("leftover boost = %d, want 100 / 2 = 50", remaining[0].pgaBoost)
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
		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}

		// Pop drops the expired entry; the orderer waits out round 1 and yields round 2's arrival.
		start := time.Now()
		item, ok := o.NextQueueItem()
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
		o := NewPGATxOrderer(context.Background(), seq, pgaConfigFetcherWithRounds(3), big.NewInt(testBaseFee))

		if !o.StartBlock() {
			t.Fatal("StartBlock = false, want true")
		}
		if item, ok := o.NextQueueItem(); !ok || item.tx.Nonce() != 0 {
			t.Fatalf("first yield = (nonce %d, %v), want nonce 0", item.tx.Nonce(), ok)
		}

		// Round 2's only arrival is rejected at push; the orderer reports it and waits for round 3.
		start := time.Now()
		item, ok := o.NextQueueItem()
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
