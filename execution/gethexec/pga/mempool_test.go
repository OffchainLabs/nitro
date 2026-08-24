// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"math/big"
	"slices"
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

// makePgaTestItem builds a mockTx with the next id from the env.
func (env *pgaMempoolTestEnv) makePgaTestItem(fee priorityFeeFunc) mockTx {
	id := env.currId
	env.currId++
	return mockTx{
		PGAState:        &PGAState{},
		id:              id,
		fee:             fee,
		expired:         new(bool),
		firstAppearance: defaultArrival,
	}
}

// mustPop peeks and removes the next valid transaction, failing the test if the mempool has none left.
func mustPop(t *testing.T, m *Mempool[mockTx]) mockTx {
	t.Helper()
	item, ok := m.ValidateAndPeek()
	if !ok {
		t.Fatal("ValidateAndPeek returned ok=false, want a transaction")
	}
	popped, ok := m.Pop()
	if !ok || popped.id != item.id {
		t.Fatalf("Pop = (id %d, ok %v), want the peeked tx (id %d)", popped.id, ok, item.id)
	}
	return item
}

func TestPgaMempoolPopsByPriority(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	tipBound := env.makePgaTestItem(constFee(5))
	capBound := env.makePgaTestItem(constFee(20))
	zeroTip := env.makePgaTestItem(constFee(0))
	env.mempool.Push(tipBound)
	env.mempool.Push(capBound)
	env.mempool.Push(zeroTip)

	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3", env.mempool.PriorityQueueLen())
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
	round1 := env.makePgaTestItem(constFee(10))
	env.mempool.Push(round1)

	// Round 2 of the same block: a higher-tip arrival joins the queue after the round boundary.
	env.mempool.ApplyRoundBoost()
	round2 := env.makePgaTestItem(constFee(50))
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

	low := env.makePgaTestItem(droppedFee())   // fee cap below base
	atBase := env.makePgaTestItem(constFee(0)) // fee cap == base, priority 0
	env.mempool.Push(low)
	env.mempool.Push(atBase)

	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if got := mustPop(t, env.mempool); got.id != atBase.id {
		t.Fatalf("survivor = %d, want atBase", got.id)
	}
}

func TestPgaMempoolNextBlockRekeysAgainstNewBaseFee(t *testing.T) {
	env := newPgaMempoolTestEnv(10)

	// A is tip-bound (constant), B is cap-bound; their order flips with the
	// basefee: B outranks A at base 10 and falls below it at base 55.
	itemA := env.makePgaTestItem(constFee(10))
	itemB := env.makePgaTestItem(func(baseFee *big.Int) (uint64, bool) {
		if baseFee.Cmp(big.NewInt(10)) == 0 {
			return 50, true
		}
		return 5, true
	})
	idA, idB := itemA.id, itemB.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)

	if top := mustPop(t, env.mempool); top.id != idB { // base 10: A=10, B=50 -> B first
		t.Fatalf("at base 10: top = %d, want B", top.id)
	}

	// The next block builds a fresh mempool against the higher basefee and re-keys the requeued txs.
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(55))
	next.Push(itemA)
	next.Push(itemB)

	if got := mustPop(t, next); got.id != idA { // base 55: A=10, B=5 -> A first
		t.Fatalf("at base 55: top = %d, want A", got.id)
	}
	if got := mustPop(t, next); got.id != idB {
		t.Fatalf("at base 55: second = %d, want B", got.id)
	}
}

func TestPgaMempoolNextBlockDropsRequeuedFeeCapTooLow(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA := env.makePgaTestItem(constFee(10))
	// B is valid at the first basefee but its fee cap falls below the second.
	itemB := env.makePgaTestItem(func(baseFee *big.Int) (uint64, bool) {
		if baseFee.Cmp(big.NewInt(60)) >= 0 {
			return 0, false
		}
		return 10, true
	})
	idA := itemA.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)
	if env.mempool.PriorityQueueLen() != 2 { // both valid at base 40
		t.Fatalf("len = %d, want 2", env.mempool.PriorityQueueLen())
	}

	// The next block re-keys the requeued txs against base 60: A stays, B drops.
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(60))
	for {
		item, ok := env.mempool.ValidateAndPeek()
		if !ok {
			break
		}
		env.mempool.Pop()
		next.Push(item)
	}

	if next.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", next.PriorityQueueLen())
	}
	if got := mustPop(t, next); got.id != idA {
		t.Fatalf("survivor = %d, want A", got.id)
	}
}

func TestPgaMempoolPeekSkipsMultipleInvalidInOneCall(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	// Two expired entries outrank the valid one, so a single Peek must skip both.
	expired1 := env.makePgaTestItem(constFee(30))
	*expired1.expired = true
	expired2 := env.makePgaTestItem(constFee(20))
	*expired2.expired = true
	good := env.makePgaTestItem(constFee(10))
	env.mempool.Push(expired1)
	env.mempool.Push(expired2)
	env.mempool.Push(good)

	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3 (lazy validation queues all three)", env.mempool.PriorityQueueLen())
	}

	// One Peek walks past both expired entries and returns the valid one.
	if got := mustPop(t, env.mempool); got.id != good.id {
		t.Fatalf("pop = %d, want good", got.id)
	}
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after draining, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPushBackKeepsPriority(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA := env.makePgaTestItem(constFee(50))
	itemB := env.makePgaTestItem(constFee(10))
	itemC := env.makePgaTestItem(constFee(30))
	idA, idB, idC := itemA.id, itemB.id, itemC.id
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)
	env.mempool.Push(itemC)

	top := mustPop(t, env.mempool)
	if top.id != idA {
		t.Fatalf("first pop = %d, want A", top.id)
	}
	env.mempool.Push(top) // push back; re-keyed to the same priority, so it pops first again

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

func TestPgaMempoolPeekReturnsFalseWhenEmpty(t *testing.T) {
	env := newPgaMempoolTestEnv(40)
	if _, ok := env.mempool.ValidateAndPeek(); ok {
		t.Fatal("Peek on an empty mempool returned ok=true, want false")
	}
}

func TestPgaMempoolPeekDropsAllInvalidThenReturnsFalse(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	bad := env.makePgaTestItem(constFee(5))
	*bad.expired = true
	env.mempool.Push(bad)

	// Lazy validation: the expired tx is queued and only dropped when Peek reaches it.
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if _, ok := env.mempool.ValidateAndPeek(); ok {
		t.Fatal("Peek returned ok=true, want false (the only tx is invalid)")
	}
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after Peek discarded the expired tx, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPopPeekedRemovesPeekedEvenIfExpired(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA := env.makePgaTestItem(constFee(20))
	itemB := env.makePgaTestItem(constFee(10))
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)

	peeked, ok := env.mempool.ValidateAndPeek()
	if !ok || peeked.id != itemA.id {
		t.Fatalf("peek = (id %d, ok %v), want A", peeked.id, ok)
	}
	// A expires between Peek and PopPeeked, as a queue timeout can. PopPeeked must not revalidate: it removes the
	// peeked tx, never the next valid one.
	*peeked.expired = true
	env.mempool.Pop()

	if got := mustPop(t, env.mempool); got.id != itemB.id {
		t.Fatalf("survivor = %d, want B", got.id)
	}
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after draining, want 0", env.mempool.PriorityQueueLen())
	}
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
	included := env.makePgaTestItem(constFee(100))
	remaining := env.makePgaTestItem(constFee(0))
	env.mempool.Push(included)
	env.mempool.Push(remaining)

	top := mustPop(t, env.mempool)
	if top.id != included.id || top.GetPriority() != 100 {
		t.Fatalf("first pop = (id %d, prio %d), want included with prio 100", top.id, top.GetPriority())
	}
	env.mempool.RecordIncludedTx(top.GetPriority())

	// Next round boosts the queue by lastIncludedPriority / (2K) = 100 / 4 = 25, lifting remaining from 0 to 25.
	env.mempool.ApplyRoundBoost()
	got := mustPop(t, env.mempool)
	if got.id != remaining.id || got.GetPriority() != 25 {
		t.Fatalf("boosted pop = (id %d, prio %d), want remaining with prio 25", got.id, got.GetPriority())
	}
}

func TestPgaMempoolBoostFoldedIntoNextBlock(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	included := env.makePgaTestItem(constFee(100))
	// remaining's fee depends on the basefee: 0 at base 40, 7 at base 50. The next block must recompute the fee and
	// re-add the accumulated boost.
	remaining := env.makePgaTestItem(func(baseFee *big.Int) (uint64, bool) {
		if baseFee.Cmp(big.NewInt(50)) == 0 {
			return 7, true
		}
		return 0, true
	})
	env.mempool.Push(included)
	env.mempool.Push(remaining)

	top := mustPop(t, env.mempool)
	if top.id != included.id {
		t.Fatalf("first pop = %d, want included", top.id)
	}
	env.mempool.RecordIncludedTx(top.GetPriority()) // 100
	env.mempool.ApplyRoundBoost()                   // boost remaining by 100 / 4 = 25

	// The next block's mempool re-keys against base 50: remaining = fee(50) + boost = 7 + 25 = 32, proving the boost
	// folds into the recomputed priority.
	leftover := mustPop(t, env.mempool)
	next := NewMempool[mockTx](testRoundsPerBlock, big.NewInt(50))
	next.Push(leftover)
	got := mustPop(t, next)
	if got.id != remaining.id || got.GetPriority() != 32 {
		t.Fatalf("after re-key pop = (id %d, prio %d), want remaining with prio 32", got.id, got.GetPriority())
	}
}

func TestPgaMempoolPushPreservesCarriedBoost(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	deferred := env.makePgaTestItem(constFee(50))
	remaining := env.makePgaTestItem(constFee(10))
	env.mempool.Push(deferred)
	env.mempool.Push(remaining)

	// Round 1 boost: lastIncludedPriority 40 -> delta 10 lifts both queued txs.
	env.mempool.RecordIncludedTx(40)
	env.mempool.ApplyRoundBoost()

	// deferred is popped but does not fit in the block, carrying its round-1 boost of 10. Once out of the queue,
	// later boosts cannot touch it.
	popped := mustPop(t, env.mempool)
	if popped.id != deferred.id || popped.GetPriority() != 60 { // 50 + 10
		t.Fatalf("popped = (id %d, prio %d), want deferred with prio 60", popped.id, popped.GetPriority())
	}

	// Round 2 boost runs while deferred is out: lastIncludedPriority 20 -> delta 5 lifts only the still-queued tx.
	env.mempool.RecordIncludedTx(20)
	env.mempool.ApplyRoundBoost()
	if popped.GetPriority() != 60 {
		t.Fatalf("deferred priority while out = %d, want 60 (untouched while out of the queue)", popped.GetPriority())
	}

	// Re-adding deferred carries its boost of 10 into the priority, not the round-2 boost it never received.
	env.mempool.Push(popped)

	first := mustPop(t, env.mempool)
	if first.id != deferred.id || first.GetPriority() != 60 { // 50 + 10, unaffected by round 2
		t.Fatalf("re-added pop = (id %d, prio %d), want deferred with prio 60", first.id, first.GetPriority())
	}
	second := mustPop(t, env.mempool)
	if second.id != remaining.id || second.GetPriority() != 25 { // 10 + 10 + 5
		t.Fatalf("remaining pop = (id %d, prio %d), want remaining with prio 25", second.id, second.GetPriority())
	}
}

func TestPgaMempoolPushBatchOrdersAndFoldsBoost(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	low := env.makePgaTestItem(constFee(10))
	high := env.makePgaTestItem(constFee(30))
	carried := env.makePgaTestItem(constFee(15))
	carried.ApplyRoundBoundary(20) // 15 + 20 = 35 outranks high's 30
	env.mempool.PushBatch([]mockTx{low, high, carried})

	first := mustPop(t, env.mempool)
	if first.id != carried.id || first.GetPriority() != 35 {
		t.Fatalf("first pop = (id %d, prio %d), want carried with prio 35", first.id, first.GetPriority())
	}
	for _, want := range []int{high.id, low.id} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("pop = %d, want %d", got.id, want)
		}
	}
}

func TestPgaMempoolPushBatchDropsFeeCapTooLow(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	bad := env.makePgaTestItem(droppedFee())
	good := env.makePgaTestItem(constFee(5))
	env.mempool.PushBatch([]mockTx{bad, good})

	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if got := mustPop(t, env.mempool); got.id != good.id {
		t.Fatalf("survivor = %d, want good", got.id)
	}
}

func TestPgaMempoolPushBatchMergesWithQueue(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	queued := env.makePgaTestItem(constFee(20))
	env.mempool.Push(queued)
	env.mempool.PushBatch(nil)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d after empty batch, want 1", env.mempool.PriorityQueueLen())
	}

	batchLow := env.makePgaTestItem(constFee(10))
	batchHigh := env.makePgaTestItem(constFee(30))
	env.mempool.PushBatch([]mockTx{batchLow, batchHigh})
	for _, want := range []int{batchHigh.id, queued.id, batchLow.id} {
		if got := mustPop(t, env.mempool); got.id != want {
			t.Fatalf("pop = %d, want %d", got.id, want)
		}
	}
}

func TestPgaMempoolTakeRemaining(t *testing.T) {
	env := newPgaMempoolTestEnv(40)

	itemA := env.makePgaTestItem(constFee(30))
	itemB := env.makePgaTestItem(constFee(10))
	itemC := env.makePgaTestItem(constFee(20))
	env.mempool.Push(itemA)
	env.mempool.Push(itemB)
	env.mempool.Push(itemC)

	if top := mustPop(t, env.mempool); top.id != itemA.id {
		t.Fatalf("pop = %d, want A", top.id)
	}

	remaining := env.mempool.TakeRemaining()
	ids := make([]int, len(remaining))
	for i, item := range remaining {
		ids[i] = item.id
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []int{itemB.id, itemC.id}) {
		t.Fatalf("TakeRemaining ids = %v, want the two never-popped txs", ids)
	}
	remaining = env.mempool.TakeRemaining()
	if len(remaining) != 0 {
		t.Fatalf("TakeRemaining was not empty after the first call, want 0, got %d", len(remaining))
	}
}
