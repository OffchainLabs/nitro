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

	"github.com/ethereum/go-ethereum/core/txpool"
)

// defaultArrival is the arrival timestamp shared by the test txs. Most tests care only about priority order, not
// arrival time, so they reuse it directly; the tie-break test offsets it where arrival order matters.
var defaultArrival = time.Unix(0, 0)

// pgaMempoolTestEnv bundles the scaffolding shared by the mempool tests: the mempool under test, its waiting-list
// channel, and an id counter that hands each mock tx a distinct id as it is created.
type pgaMempoolTestEnv struct {
	mempool *Mempool[mockTx]
	ch      chan mockTx
	currId  int
}

const testChanCap = 100
const testRoundsPerBlock = 2

// newPgaMempoolTestEnv builds an env whose mempool reads from a waiting-list channel of capacity testChanCap.
func newPgaMempoolTestEnv() *pgaMempoolTestEnv {
	ch := make(chan mockTx, testChanCap)
	return &pgaMempoolTestEnv{
		mempool: NewMempool(ch, testRoundsPerBlock),
		ch:      ch,
	}
}

// makePgaTestItem builds a mockTx like publishTransactionToQueue builds a txQueueItem, assigning it the next id from
// the env, and returns its result channel so tests can assert what the mempool sent back.
func (env *pgaMempoolTestEnv) makePgaTestItem(ctx context.Context, fee priorityFeeFunc, size int, firstAppearance time.Time) (mockTx, chan error) {
	id := env.currId
	env.currId++
	resultChan := make(chan error, 1)
	item := mockTx{
		id:              id,
		fee:             fee,
		size:            size,
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

func TestPgaMempoolStartNewBlockDrainsWaitingList(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	tipBound, r1 := env.makePgaTestItem(context.Background(), constFee(5), 10, defaultArrival)
	capBound, r2 := env.makePgaTestItem(context.Background(), constFee(20), 10, defaultArrival)
	zeroTip, r3 := env.makePgaTestItem(context.Background(), constFee(0), 10, defaultArrival)
	env.ch <- tipBound
	env.ch <- capBound
	env.ch <- zeroTip

	env.mempool.StartNewBlock(base, 1000)

	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3", env.mempool.PriorityQueueLen())
	}
	if len(env.ch) != 0 {
		t.Fatalf("waiting list not drained: %d remaining", len(env.ch))
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

	// second drain on the empty channel must return, not block
	env.mempool.StartNewBlock(base, 1000)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after second StartNewBlock, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolAreThereTxsForNextRound(t *testing.T) {
	env := newPgaMempoolTestEnv()

	// Empty mempool and empty channel: nothing for the next round.
	if env.mempool.AreThereTxsForNextRound() {
		t.Fatal("empty mempool should report no txs for the next round")
	}

	// A waiting-list arrival counts, even before promotion.
	item, _ := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	env.ch <- item
	if !env.mempool.AreThereTxsForNextRound() {
		t.Fatal("a waiting-list tx should count for the next round")
	}

	// After promotion it's in the heap and the channel is empty; still counts.
	env.mempool.StartNewBlock(big.NewInt(40), 1000)
	if len(env.ch) != 0 {
		t.Fatalf("channel not drained: %d", len(env.ch))
	}
	if !env.mempool.AreThereTxsForNextRound() {
		t.Fatal("a queued tx should count for the next round")
	}

	// Drain the queue: nothing left.
	mustPop(t, env.mempool)
	if env.mempool.AreThereTxsForNextRound() {
		t.Fatal("fully drained mempool should report no txs for the next round")
	}
}

func TestPgaMempoolMultipleRoundsPerBlock(t *testing.T) {
	env := newPgaMempoolTestEnv()

	// Round 1: StartNewBlock records the basefee and maxsize and runs the block's first round, draining the arrival.
	round1, _ := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	env.ch <- round1
	env.mempool.StartNewBlock(big.NewInt(40), 1000)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("after round 1: len = %d, want 1", env.mempool.PriorityQueueLen())
	}

	// Round 2 of the same block: a higher-tip arrival is promoted using the
	// basefee and maxsize StartNewBlock recorded, without a new StartNewBlock.
	round2, _ := env.makePgaTestItem(context.Background(), constFee(50), 10, defaultArrival)
	env.ch <- round2
	env.mempool.StartNewPGARound()
	if env.mempool.PriorityQueueLen() != 2 {
		t.Fatalf("after round 2: len = %d, want 2", env.mempool.PriorityQueueLen())
	}

	// Highest priority pops first, across rounds.
	if got := mustPop(t, env.mempool); got.id != round2.id {
		t.Fatalf("first pop = %d, want round2", got.id)
	}
	if got := mustPop(t, env.mempool); got.id != round1.id {
		t.Fatalf("second pop = %d, want round1", got.id)
	}
}

func TestPgaMempoolStartNewBlockDrainsOnlySnapshot(t *testing.T) {
	env := newPgaMempoolTestEnv()
	// Fill the buffer to capacity so the snapshot at promotion start is exactly testChanCap.
	for range testChanCap {
		item, _ := env.makePgaTestItem(context.Background(), constFee(5), 10, defaultArrival)
		env.ch <- item
	}
	// A producer that keeps offering more: these arrive during promotion and
	// must wait for the next round rather than join the current one.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			item, _ := env.makePgaTestItem(context.Background(), constFee(5), 10, defaultArrival)
			select {
			case env.ch <- item:
			case <-stop:
				return
			}
		}
	}()

	env.mempool.StartNewBlock(big.NewInt(40), 1000)
	close(stop)
	<-done

	if env.mempool.PriorityQueueLen() != testChanCap {
		t.Fatalf("promoted %d, want exactly the snapshot of %d", env.mempool.PriorityQueueLen(), testChanCap)
	}
}

func TestPgaMempoolStartNewBlockRejectsFeeCapBelowBaseFee(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	low, lowResult := env.makePgaTestItem(context.Background(), failFee(errFeeCapTooLow), 10, defaultArrival) // fee cap below base
	atBase, atBaseResult := env.makePgaTestItem(context.Background(), constFee(0), 10, defaultArrival)        // fee cap == base, priority 0
	env.ch <- low
	env.ch <- atBase

	env.mempool.StartNewBlock(base, 1000)

	expectResult(t, lowResult, errFeeCapTooLow)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	expectNoResult(t, atBaseResult)
	if got := mustPop(t, env.mempool); got.id != atBase.id {
		t.Fatalf("survivor = %d, want atBase", got.id)
	}
}

func TestPgaMempoolStartNewBlockRecomputesAgainstNewBaseFee(t *testing.T) {
	env := newPgaMempoolTestEnv()

	// A is tip-bound (constant), B is cap-bound; their order flips with the
	// basefee: B outranks A at base 10 and falls below it at base 55.
	itemA, resultA := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	itemB, resultB := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(10)) == 0 {
			return 50, nil
		}
		return 5, nil
	}, 10, defaultArrival)
	idA, idB := itemA.id, itemB.id
	env.ch <- itemA
	env.ch <- itemB

	env.mempool.StartNewBlock(big.NewInt(10), 1000) // base 10: A=10, B=50 -> B first
	top := mustPopEntry(t, env.mempool)
	if top.tx.id != idB {
		t.Fatalf("before re-key: top = %d, want B", top.tx.id)
	}
	env.mempool.Push(top) // restore for the re-key

	// A second StartNewBlock against a higher basefee re-keys the queued txs.
	env.mempool.StartNewBlock(big.NewInt(55), 1000) // base 55: A=10, B=5 -> A first

	expectNoResult(t, resultA)
	expectNoResult(t, resultB)
	if got := mustPop(t, env.mempool); got.id != idA {
		t.Fatalf("after re-key: top = %d, want A", got.id)
	}
	if got := mustPop(t, env.mempool); got.id != idB {
		t.Fatalf("after re-key: second = %d, want B", got.id)
	}
}

func TestPgaMempoolStartNewBlockDropsRequeuedFeeCapTooLow(t *testing.T) {
	env := newPgaMempoolTestEnv()

	itemA, resultA := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	// B is valid at the first basefee but its fee cap falls below the second.
	itemB, resultB := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(60)) >= 0 {
			return 0, errFeeCapTooLow
		}
		return 10, nil
	}, 10, defaultArrival)
	idA := itemA.id
	env.ch <- itemA
	env.ch <- itemB
	env.mempool.StartNewBlock(big.NewInt(40), 1000) // both valid at base 40

	// A second StartNewBlock against a higher basefee drops the requeued tx whose fee
	// cap is now too low.
	env.mempool.StartNewBlock(big.NewInt(60), 1000) // base 60: A stays, B drops

	expectResult(t, resultB, errFeeCapTooLow)
	expectNoResult(t, resultA)
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if got := mustPop(t, env.mempool); got.id != idA {
		t.Fatalf("survivor = %d, want A", got.id)
	}
}

func TestPgaMempoolPopSkipsMultipleInvalidInOneCall(t *testing.T) {
	env := newPgaMempoolTestEnv()
	const maxSize = 100

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	// Two invalid entries (different drop reasons) outrank the valid one, so a single Pop must skip both.
	expired, expiredResult := env.makePgaTestItem(canceledCtx, constFee(30), 10, defaultArrival)
	oversized, oversizedResult := env.makePgaTestItem(context.Background(), constFee(20), maxSize+1, defaultArrival)
	good, goodResult := env.makePgaTestItem(context.Background(), constFee(10), maxSize, defaultArrival)
	env.ch <- expired
	env.ch <- oversized
	env.ch <- good

	env.mempool.StartNewBlock(big.NewInt(40), maxSize)
	if env.mempool.PriorityQueueLen() != 3 {
		t.Fatalf("len = %d, want 3 (lazy validation promotes all three)", env.mempool.PriorityQueueLen())
	}

	// One Pop walks past both invalid entries (expired, then oversized) and returns the valid one.
	if got := mustPop(t, env.mempool); got.id != good.id {
		t.Fatalf("pop = %d, want good", got.id)
	}
	expectResult(t, expiredResult, context.Canceled)
	expectResult(t, oversizedResult, txpool.ErrOversizedData)
	expectNoResult(t, goodResult)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after draining, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPopDropsOversizedAfterMaxSizeDecrease(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	// large outranks small, so Pop reaches it first once the max size shrinks below its size.
	small, smallResult := env.makePgaTestItem(context.Background(), constFee(10), 50, defaultArrival)  // 50 bytes
	large, largeResult := env.makePgaTestItem(context.Background(), constFee(20), 100, defaultArrival) // 100 bytes
	idSmall := small.id
	env.ch <- small
	env.ch <- large
	env.mempool.StartNewBlock(base, 100) // both fit under the initial max size of 100
	if env.mempool.PriorityQueueLen() != 2 {
		t.Fatalf("after first StartNewBlock: len = %d, want 2", env.mempool.PriorityQueueLen())
	}

	// A later block shrinks the hot-reloadable max transaction size. The previously-accepted tx is not swept on re-key;
	// it stays in the heap, and Pop drops it lazily by validating against the current (smaller) max size.
	env.mempool.StartNewBlock(base, 99) // max size now 99: large (100) no longer fits
	if got := mustPop(t, env.mempool); got.id != idSmall {
		t.Fatalf("survivor = %d, want small", got.id)
	}
	expectResult(t, largeResult, txpool.ErrOversizedData)
	expectNoResult(t, smallResult)
}

func TestPgaMempoolPushBackKeepsPriority(t *testing.T) {
	env := newPgaMempoolTestEnv()

	itemA, _ := env.makePgaTestItem(context.Background(), constFee(50), 10, defaultArrival)
	itemB, _ := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	itemC, _ := env.makePgaTestItem(context.Background(), constFee(30), 10, defaultArrival)
	idA, idB, idC := itemA.id, itemB.id, itemC.id
	env.ch <- itemA
	env.ch <- itemB
	env.ch <- itemC
	env.mempool.StartNewBlock(big.NewInt(40), 1000)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != idA {
		t.Fatalf("first pop = %d, want A", top.tx.id)
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

// TestPgaMempoolPushDropsOnFeeError covers Push's re-key failure branch: a
// push-back whose ComputePgaPriority errors is dropped with that error rather
// than queued. The basefee is constant within a block, so a re-key that
// succeeded on the way out cannot fail on push-back today; this pins the
// public API's behavior regardless.
func TestPgaMempoolPushDropsOnFeeError(t *testing.T) {
	env := newPgaMempoolTestEnv()
	env.mempool.StartNewBlock(big.NewInt(40), 1000) // sets the basefee Push re-keys against

	item, result := env.makePgaTestItem(context.Background(), failFee(errFeeCapTooLow), 10, defaultArrival)
	env.mempool.Push(PrioritizedTx[mockTx]{tx: item})

	expectResult(t, result, errFeeCapTooLow)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d, want 0 (a failed re-key should drop the tx, not queue it)", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolPopReturnsFalseWhenEmpty(t *testing.T) {
	env := newPgaMempoolTestEnv()
	if _, ok := env.mempool.Pop(); ok {
		t.Fatal("Pop on an empty mempool returned ok=true, want false")
	}
}

func TestPgaMempoolPopDropsAllInvalidThenReturnsFalse(t *testing.T) {
	env := newPgaMempoolTestEnv()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	bad, badResult := env.makePgaTestItem(canceledCtx, constFee(5), 10, defaultArrival)
	env.ch <- bad
	env.mempool.StartNewBlock(big.NewInt(40), 1000)

	// Lazy validation: the expired tx is promoted and only dropped when Pop reaches it.
	if env.mempool.PriorityQueueLen() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.PriorityQueueLen())
	}
	if _, ok := env.mempool.Pop(); ok {
		t.Fatal("Pop returned ok=true, want false (the only tx is invalid)")
	}
	expectResult(t, badResult, context.Canceled)
}

func TestPgaMempoolEmptyOps(t *testing.T) {
	env := newPgaMempoolTestEnv() // empty waiting list
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("fresh len = %d, want 0", env.mempool.PriorityQueueLen())
	}
	// StartNewBlock must not block on the empty channel and must not panic.
	env.mempool.StartNewBlock(big.NewInt(40), 1000)
	if env.mempool.PriorityQueueLen() != 0 {
		t.Fatalf("len = %d after no-op StartNewBlock, want 0", env.mempool.PriorityQueueLen())
	}
}

func TestPgaMempoolBoostAccumulation(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	// included pops and is recorded; remaining stays queued and should be boosted at the next round boundary.
	included, _ := env.makePgaTestItem(context.Background(), constFee(100), 10, defaultArrival)
	remaining, _ := env.makePgaTestItem(context.Background(), constFee(0), 10, defaultArrival)
	env.ch <- included
	env.ch <- remaining
	env.mempool.StartNewBlock(base, 1000)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != included.id || top.cachedPriority != 100 {
		t.Fatalf("first pop = (id %d, prio %d), want included with prio 100", top.tx.id, top.cachedPriority)
	}
	env.mempool.RecordIncludedTx(top.cachedPriority)

	// Next round boosts the queue by lastIncludedPriority / (2K) = 100 / 4 = 25, lifting remaining from 0 to 25.
	env.mempool.StartNewPGARound()
	got := mustPopEntry(t, env.mempool)
	if got.tx.id != remaining.id || got.cachedPriority != 25 {
		t.Fatalf("boosted pop = (id %d, prio %d), want remaining with prio 25", got.tx.id, got.cachedPriority)
	}
	if got.boost != 25 {
		t.Fatalf("remaining boost = %d, want 25", got.boost)
	}
}

func TestPgaMempoolBoostFoldedIntoNextBlock(t *testing.T) {
	env := newPgaMempoolTestEnv()

	included, _ := env.makePgaTestItem(context.Background(), constFee(100), 10, defaultArrival)
	// remaining's fee depends on the basefee: 0 at base 40, 7 at base 50. The next block must recompute the fee and
	// re-add the accumulated boost.
	remaining, _ := env.makePgaTestItem(context.Background(), func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(50)) == 0 {
			return 7, nil
		}
		return 0, nil
	}, 10, defaultArrival)
	env.ch <- included
	env.ch <- remaining
	env.mempool.StartNewBlock(big.NewInt(40), 1000)

	top := mustPopEntry(t, env.mempool)
	if top.tx.id != included.id {
		t.Fatalf("first pop = %d, want included", top.tx.id)
	}
	env.mempool.RecordIncludedTx(top.cachedPriority) // 100
	env.mempool.StartNewPGARound()                   // boost remaining by 100 / 4 = 25

	// A new block re-keys against base 50: remaining = fee(50) + boost = 7 + 25 = 32, proving the boost folds into the
	// recomputed priority.
	env.mempool.StartNewBlock(big.NewInt(50), 1000)
	got := mustPopEntry(t, env.mempool)
	if got.tx.id != remaining.id || got.cachedPriority != 32 {
		t.Fatalf("after re-key pop = (id %d, prio %d), want remaining with prio 32", got.tx.id, got.cachedPriority)
	}
}

func TestPgaMempoolBoostFreezeForCachedEntry(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	cached, _ := env.makePgaTestItem(context.Background(), constFee(50), 10, defaultArrival)
	remaining, _ := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	env.ch <- cached
	env.ch <- remaining
	env.mempool.StartNewBlock(base, 1000)

	// Round 1 boost: lastIncludedPriority 40 -> delta 10 lifts both queued txs.
	env.mempool.RecordIncludedTx(40)
	env.mempool.StartNewPGARound()

	// cached leaves the queue (as if moved to the nonce-failure cache) carrying its round-1 boost of 10. The popped
	// entry is a detached copy, so later boosts cannot touch it.
	popped := mustPopEntry(t, env.mempool)
	if popped.tx.id != cached.id || popped.cachedPriority != 60 { // 50 + 10
		t.Fatalf("popped = (id %d, prio %d), want cached with prio 60", popped.tx.id, popped.cachedPriority)
	}
	if popped.boost != 10 {
		t.Fatalf("cached boost at pop = %d, want 10", popped.boost)
	}

	// Round 2 boost runs while cached is out: lastIncludedPriority 20 -> delta 5 lifts only the still-queued tx.
	env.mempool.RecordIncludedTx(20)
	env.mempool.StartNewPGARound()
	if popped.boost != 10 {
		t.Fatalf("cached boost while out = %d, want 10 (frozen while cached)", popped.boost)
	}

	// Revive cached. It re-enters with its frozen boost of 10, not the round-2 boost it never received.
	env.mempool.Push(popped)

	first := mustPopEntry(t, env.mempool)
	if first.tx.id != cached.id || first.cachedPriority != 60 { // 50 + 10, unaffected by round 2
		t.Fatalf("revived pop = (id %d, prio %d), want cached with prio 60", first.tx.id, first.cachedPriority)
	}
	second := mustPopEntry(t, env.mempool)
	if second.tx.id != remaining.id || second.cachedPriority != 25 { // 10 + 10 + 5
		t.Fatalf("remaining pop = (id %d, prio %d), want remaining with prio 25", second.tx.id, second.cachedPriority)
	}
	if second.boost != 15 {
		t.Fatalf("remaining boost = %d, want 15 (round 1 + round 2)", second.boost)
	}
}
