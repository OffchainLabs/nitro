// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
)

// defaultArrival is the arrival timestamp shared by the test txs. Most tests care only about priority order, not
// arrival time, so they reuse it directly; the tie-break test offsets it where arrival order matters.
var defaultArrival = time.Unix(0, 0)

// pgaMempoolTestEnv bundles the scaffolding shared by the mempool tests: the mempool under test and an id counter
// that hands each mock tx a distinct id as it is created.
type pgaMempoolTestEnv struct {
	mempool *Mempool[mockTx]
	currId  int
}

const testRoundsPerBlock = 2

// newPgaMempoolTestEnv builds an env whose mempool keys priorities against the given basefee.
func newPgaMempoolTestEnv(baseFee int64) *pgaMempoolTestEnv {
	return &pgaMempoolTestEnv{
		mempool: NewMempool[mockTx](testRoundsPerBlock, big.NewInt(baseFee)),
	}
}

// makePgaTestItem builds a mockTx like publishTransactionToQueue builds a txQueueItem, assigning it the next id from
// the env, and returns its result channel so tests can assert what the mempool sent back.
func (env *pgaMempoolTestEnv) makePgaTestItem(ctx context.Context, fee priorityFeeFunc, firstAppearance time.Time) (mockTx, chan error) {
	id := env.currId
	env.currId++
	resultChan := make(chan error, 1)
	item := mockTx{
		id:              id,
		fee:             fee,
		ctx:             ctx,
		firstAppearance: firstAppearance,
		resultChan:      resultChan,
		returnedResult:  &atomic.Bool{},
	}
	return item, resultChan
}

func expectNoResult(t *testing.T, resultChan chan error) {
	t.Helper()
	select {
	case err, ok := <-resultChan:
		t.Fatalf("expected no result, got err=%v (open=%v)", err, ok)
	default:
	}
}

func expectResult(t *testing.T, resultChan chan error, want error) {
	t.Helper()
	select {
	case err := <-resultChan:
		if !errors.Is(err, want) {
			t.Fatalf("result error = %v, want errors.Is(%v)", err, want)
		}
	default:
		t.Fatalf("expected result %v, got none", want)
	}
}

// mustPop pops the next valid transaction, failing the test if the mempool has none left.
func mustPop(t *testing.T, m *Mempool[mockTx]) mockTx {
	t.Helper()
	entry, ok := m.Pop()
	if !ok {
		t.Fatal("Pop returned ok=false, want a transaction")
	}
	return entry.tx
}

// mustPopEntry pops the next valid entry, failing if the mempool is empty. Tests use it when they need the entry's
// priority or boost, or want to re-insert it.
func mustPopEntry(t *testing.T, m *Mempool[mockTx]) PrioritizedTx[mockTx] {
	t.Helper()
	entry, ok := m.Pop()
	if !ok {
		t.Fatal("Pop returned ok=false, want a transaction")
	}
	return entry
}

func TestPgaMempoolPopsByPriority(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	tipBound, r1 := env.makePgaTestItem(context.Background(), constFee(5), defaultArrival)
	capBound, r2 := env.makePgaTestItem(context.Background(), constFee(20), defaultArrival)
	zeroTip, r3 := env.makePgaTestItem(context.Background(), constFee(0), defaultArrival)
	env.mempool.Push(tipBound)
	env.mempool.Push(capBound)
	env.mempool.Push(zeroTip)

	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3", env.mempool.PriorityQueueLen())
	}
	for _, r := range []chan error{r1, r2, r3} {
		expectNoResult(t, r)
	}
	// pop order by priority: capBound (20), tipBound (5), zeroTip (0)
	for _, want := range []int{capBound.id, tipBound.id, zeroTip.id} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("popped %d, want %d", got.id, want)
		}
	}
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after draining, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolMultipleRoundsPerBlock(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	// Round 1 arrival.
	round1, _ := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	env.mempool.Push(round1)

	// Round 2 of the same block: a higher-tip arrival joins the queue after the round boundary.
	env.mempool.ApplyRoundBoost()
	round2, _ := env.makePgaTestItem(context.Background(), constFee(50), defaultArrival)
	env.mempool.Push(round2)

	// Highest priority pops first, across rounds.
	if got := mustPop(t, env.mempool); got.id != round2.id {
		t.Fatalf("first pop = %d, want round2", got.id)
	}
	if got := mustPop(t, env.mempool); got.id != round1.id {
		t.Fatalf("second pop = %d, want round1", got.id)
	}
}

func TestPgaMempoolPushRejectsFeeCapBelowBaseFee(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	low, lowResult := env.makePgaTestItem(context.Background(), failFee(errFeeCapTooLow), defaultArrival) // fee cap below base
	atBase, atBaseResult := env.makePgaTestItem(context.Background(), constFee(0), defaultArrival)        // fee cap == base, priority 0
	env.mempool.Push(low)
	env.mempool.Push(atBase)

	expectResult(t, lowResult, errFeeCapTooLow)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	expectNoResult(t, atBaseResult)
	if got := mustPop(t, env.mempool); got.id != atBase.id {
		t.Fatalf("survivor = %d, want atBase", got.id)
	}
}

func TestPgaMempoolNextBlockRekeysAgainstNewBaseFee(t *testing.T) {
	env := newPgaMempoolTestEnv(10)

	// A is tip-bound (constant), B is cap-bound; their order flips with the
	// basefee: B outranks A at base 10 and falls below it at base 55.
	itemA, resultA := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	itemB, resultB := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(10)) == 0 {
			return 50, nil
		}
		return 5, nil
	}, defaultArrival)
	idA, idB := itemA.id, itemB.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)

	if top := mustPopEntry(t, env.mempool); top.tx.id != idB { // base 10: A=10, B=50 -> B first
		t.Fatalf("at base 10: top = %d, want B", top.tx.id)
	}

	// The next block builds a fresh mempool against the higher basefee and re-keys the requeued txs.
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(55))
	next.Push(itemA)
	next.Push(itemB)

	expectNoResult(t, resultA)
	expectNoResult(t, resultB)
	if got := mustPop(t, next); got.id != idA { // base 55: A=10, B=5 -> A first
		t.Fatalf("at base 55: top = %d, want A", got.id)
	}
	if got := mustPop(t, next); got.id != idB {
		t.Fatalf("at base 55: second = %d, want B", got.id)
	}
}

func TestPgaMempoolNextBlockDropsRequeuedFeeCapTooLow(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA, resultA := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	// B is valid at the first basefee but its fee cap falls below the second.
	itemB, resultB := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(60)) >= 0 {
			return 0, errFeeCapTooLow
		}
		return 10, nil
	}, defaultArrival)
	idA := itemA.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)
	if env.mempool.PriorityQueueLen() != 2 { // both valid at base 40
		t.Fatalf("len = %d, want 2", env.mempool.PriorityQueueLen())
	}

	// The next block re-keys the requeued txs against base 60: A stays, B drops.
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(60))
	for {
		entry, ok := env.mempool.Pop()
		if !ok {
			break
		}
		tx := entry.Tx()
		tx.boost = entry.Boost()
		next.Push(tx)
	}

	expectResult(t, resultB, errFeeCapTooLow)
	expectNoResult(t, resultA)
	if next.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", next.PriorityQueueLen())
	}
	if got := mustPop(t, next); got.id != idA {
		t.Fatalf("survivor = %d, want A", got.id)
	}
}

func TestPgaMempoolPopSkipsMultipleInvalidInOneCall(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	// Two expired entries outrank the valid one, so a single Pop must skip both.
	expired1, expired1Result := env.makePgaTestItem(canceledCtx, constFee(30), defaultArrival)
	expired2, expired2Result := env.makePgaTestItem(canceledCtx, constFee(20), defaultArrival)
	good, goodResult := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	env.mempool.Push(expired1)
	env.mempool.Push(expired2)
	env.mempool.Push(good)

	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3 (lazy validation queues all three)", env.mempool.PriorityQueueLen())
	}

	// One Pop walks past both expired entries and returns the valid one.
	if got := mustPop(t, env.mempool); got.id != good.id {
		t.Fatalf("pop = %d, want good", got.id)
	}
	expectResult(t, expired1Result, context.Canceled)
	expectResult(t, expired2Result, context.Canceled)
	expectNoResult(t, goodResult)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after draining, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPushBackKeepsPriority(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA, _ := env.makePgaTestItem(context.Background(), constFee(50), defaultArrival)
	itemB, _ := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	itemC, _ := env.makePgaTestItem(context.Background(), constFee(30), defaultArrival)
	idA, idB, idC := itemA.id, itemB.id, itemC.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)
	env.mempool.Push(itemC)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != idA {
		t.Fatalf("first pop = %d, want A", top.tx.id)
	}
	pushedBack := top.Tx()
	pushedBack.boost = top.Boost()
	env.mempool.Push(pushedBack) // push back; re-keyed to the same priority, so it pops first again

	if again := mustPop(t, env.mempool); again.id != idA {
		t.Fatalf("after push-back = %d, want A", again.id)
	}
	for _, want := range []int{idC, idB} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("remaining pop = %d, want %d", got.id, want)
		}
	}
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("queue not empty after draining: len %d", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPopReturnsFalseWhenEmpty(t *testing.T) {
	env := newPgaMempoolTestEnv(40)
	if _, ok := env.mempool.Pop(); ok {
		t.Fatal("Pop on an empty mempool returned ok=true, want false")
	}
}

func TestPgaMempoolPopDropsAllInvalidThenReturnsFalse(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	bad, badResult := env.makePgaTestItem(canceledCtx, constFee(5), defaultArrival)
	env.mempool.Push(bad)

	// Lazy validation: the expired tx is queued and only dropped when Pop reaches it.
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if _, ok := env.mempool.Pop(); ok {
		t.Fatal("Pop returned ok=true, want false (the only tx is invalid)")
	}
	expectResult(t, badResult, context.Canceled)
}

func TestPgaMempoolEmptyOps(t *testing.T) {
	env := newPgaMempoolTestEnv(40)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("fresh len = %d, want 0", env.mempool.PriorityQueueLen())
	}
	// ApplyRoundBoost on an empty mempool must be a no-op, not a panic.
	env.mempool.ApplyRoundBoost()
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after no-op ApplyRoundBoost, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolBoostAccumulation(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	// included pops and is recorded; remaining stays queued and should be boosted at the next round boundary.
	included, _ := env.makePgaTestItem(context.Background(), constFee(100), defaultArrival)
	remaining, _ := env.makePgaTestItem(context.Background(), constFee(0), defaultArrival)
	env.mempool.Push(included)
	env.mempool.Push(remaining)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != included.id || top.cachedPriority != 100 {
		t.Fatalf("first pop = (id %d, prio %d), want included with prio 100", top.tx.id, top.cachedPriority)
	}
	env.mempool.RecordIncludedTx(top.cachedPriority)

	// Next round boosts the queue by lastIncludedPriority / (2K) = 100 / 4 = 25, lifting remaining from 0 to 25.
	env.mempool.ApplyRoundBoost()
	got := mustPopEntry(t, env.mempool)
	if got.tx.id != remaining.id || got.cachedPriority != 25 {
		t.Fatalf("boosted pop = (id %d, prio %d), want remaining with prio 25", got.tx.id, got.cachedPriority)
	}
	if got.boost != 25 {
		t.Fatalf("remaining boost = %d, want 25", got.boost)
	}
}

func TestPgaMempoolBoostFoldedIntoNextBlock(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	included, _ := env.makePgaTestItem(context.Background(), constFee(100), defaultArrival)
	// remaining's fee depends on the basefee: 0 at base 40, 7 at base 50. The next block must recompute the fee and
	// re-add the accumulated boost.
	remaining, _ := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(50)) == 0 {
			return 7, nil
		}
		return 0, nil
	}, defaultArrival)
	env.mempool.Push(included)
	env.mempool.Push(remaining)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != included.id {
		t.Fatalf("first pop = %d, want included", top.tx.id)
	}
	env.mempool.RecordIncludedTx(top.cachedPriority) // 100
	env.mempool.ApplyRoundBoost()                    // boost remaining by 100 / 4 = 25

	// The next block's mempool re-keys against base 50: remaining = fee(50) + boost = 7 + 25 = 32, proving the boost
	// folds into the recomputed priority.
	leftover := mustPopEntry(t, env.mempool)
	requeued := leftover.Tx()
	requeued.boost = leftover.Boost()
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(50))
	next.Push(requeued)
	got := mustPopEntry(t, next)
	if got.tx.id != remaining.id || got.cachedPriority != 32 {
		t.Fatalf("after re-key pop = (id %d, prio %d), want remaining with prio 32", got.tx.id, got.cachedPriority)
	}
}

func TestPgaMempoolPushPreservesCarriedBoost(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	deferred, _ := env.makePgaTestItem(context.Background(), constFee(50), defaultArrival)
	remaining, _ := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	env.mempool.Push(deferred)
	env.mempool.Push(remaining)

	// Round 1 boost: lastIncludedPriority 40 -> delta 10 lifts both queued txs.
	env.mempool.RecordIncludedTx(40)
	env.mempool.ApplyRoundBoost()

	// deferred is popped but does not fit in the block, carrying its round-1 boost of 10. The popped entry is a
	// detached copy, so later boosts cannot touch it.
	popped := mustPopEntry(t, env.mempool)
	if popped.tx.id != deferred.id || popped.cachedPriority != 60 { // 50 + 10
		t.Fatalf("popped = (id %d, prio %d), want deferred with prio 60", popped.tx.id, popped.cachedPriority)
	}
	if popped.boost != 10 {
		t.Fatalf("deferred boost at pop = %d, want 10", popped.boost)
	}

	// Round 2 boost runs while deferred is out: lastIncludedPriority 20 -> delta 5 lifts only the still-queued tx.
	env.mempool.RecordIncludedTx(20)
	env.mempool.ApplyRoundBoost()
	if popped.boost != 10 {
		t.Fatalf("deferred boost while out = %d, want 10 (untouched while out of the queue)", popped.boost)
	}

	// Re-adding deferred carries its boost of 10 into the priority, not the round-2 boost it never received.
	readded := popped.Tx()
	readded.boost = popped.Boost()
	env.mempool.Push(readded)

	first := mustPopEntry(t, env.mempool)
	if first.tx.id != deferred.id || first.cachedPriority != 60 { // 50 + 10, unaffected by round 2
		t.Fatalf("re-added pop = (id %d, prio %d), want deferred with prio 60", first.tx.id, first.cachedPriority)
	}
	second := mustPopEntry(t, env.mempool)
	if second.tx.id != remaining.id || second.cachedPriority != 25 { // 10 + 10 + 5
		t.Fatalf("remaining pop = (id %d, prio %d), want remaining with prio 25", second.tx.id, second.cachedPriority)
	}
	if second.boost != 15 {
		t.Fatalf("remaining boost = %d, want 15 (round 1 + round 2)", second.boost)
	}
}

func TestPgaMempoolPushFoldsCarriedBoost(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	carried, _ := env.makePgaTestItem(context.Background(), constFee(15), defaultArrival)
	carried.boost = 20
	plain, _ := env.makePgaTestItem(context.Background(), constFee(30), defaultArrival)
	env.mempool.Push(carried)
	env.mempool.Push(plain)

	// carried outranks plain: 15 + 20 > 30.
	first := mustPopEntry(t, env.mempool)
	if first.tx.id != carried.id || first.cachedPriority != 35 || first.boost != 20 {
		t.Fatalf("first pop = (id %d, prio %d, boost %d), want carried with (35, 20)", first.tx.id, first.cachedPriority, first.boost)
	}
	if got := mustPop(t, env.mempool); got.id != plain.id {
		t.Fatalf("second pop = %d, want plain", got.id)
	}
}

func TestPgaMempoolPushBatchOrdersAndFoldsBoost(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	low, _ := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	high, _ := env.makePgaTestItem(context.Background(), constFee(30), defaultArrival)
	carried, _ := env.makePgaTestItem(context.Background(), constFee(15), defaultArrival)
	carried.boost = 20 // 15 + 20 = 35 outranks high's 30
	env.mempool.PushBatch([]mockTx{low, high, carried})

	first := mustPopEntry(t, env.mempool)
	if first.tx.id != carried.id || first.cachedPriority != 35 || first.boost != 20 {
		t.Fatalf("first pop = (id %d, prio %d, boost %d), want carried with (35, 20)", first.tx.id, first.cachedPriority, first.boost)
	}
	for _, want := range []int{high.id, low.id} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("pop = %d, want %d", got.id, want)
		}
	}
}

func TestPgaMempoolPushBatchDropsFeeCapTooLow(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	bad, badResult := env.makePgaTestItem(context.Background(), failFee(errFeeCapTooLow), defaultArrival)
	good, goodResult := env.makePgaTestItem(context.Background(), constFee(5), defaultArrival)
	env.mempool.PushBatch([]mockTx{bad, good})

	expectResult(t, badResult, errFeeCapTooLow)
	expectNoResult(t, goodResult)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if got := mustPop(t, env.mempool); got.id != good.id {
		t.Fatalf("survivor = %d, want good", got.id)
	}
}

func TestPgaMempoolPushBatchMergesWithQueue(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	queued, _ := env.makePgaTestItem(context.Background(), constFee(20), defaultArrival)
	env.mempool.Push(queued)
	env.mempool.PushBatch(nil)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d after empty batch, want 1", env.mempool.PriorityQueueLen())
	}

	batchLow, _ := env.makePgaTestItem(context.Background(), constFee(10), defaultArrival)
	batchHigh, _ := env.makePgaTestItem(context.Background(), constFee(30), defaultArrival)
	env.mempool.PushBatch([]mockTx{batchLow, batchHigh})
	for _, want := range []int{batchHigh.id, queued.id, batchLow.id} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("pop = %d, want %d", got.id, want)
		}
	}
}
