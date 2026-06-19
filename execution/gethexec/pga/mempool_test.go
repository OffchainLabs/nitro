// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"container/heap"
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

// newPgaMempoolTestEnv builds an env whose mempool reads from a waiting-list channel of capacity testChanCap.
func newPgaMempoolTestEnv() *pgaMempoolTestEnv {
	ch := make(chan mockTx, testChanCap)
	return &pgaMempoolTestEnv{
		mempool: NewMempool(ch),
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

// forcePushMockTx inserts an entry directly into the heap with an explicit priority (bypassing setPriority and the
// waiting list), assigning it the next id from the env, and returns that id for identification.
func (env *pgaMempoolTestEnv) forcePushMockTx(priority uint64, firstAppearance time.Time) int {
	item, _ := env.makePgaTestItem(context.Background(), constFee(priority), 0, firstAppearance)
	heap.Push(&env.mempool.heap, prioritizedTx[mockTx]{tx: item, priority: priority})
	return item.id
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

func TestPgaMempoolPopOrdersByPriorityDesc(t *testing.T) {
	env := newPgaMempoolTestEnv()
	priorities := []uint64{5, 1, 9, 7, 3}
	idByPriority := make(map[uint64]int, len(priorities))
	for _, p := range priorities {
		idByPriority[p] = env.forcePushMockTx(p, defaultArrival)
	}
	for _, want := range []uint64{9, 7, 5, 3, 1} {
		if env.mempool.Len() == 0 {
			t.Fatalf("queue emptied before popping priority %d", want)
		}
		if got := env.mempool.Pop(); got.id != idByPriority[want] {
			t.Fatalf("popped wrong tx for priority %d", want)
		}
	}
	if env.mempool.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", env.mempool.Len())
	}
}

func TestPgaMempoolPopBreaksTiesByFirstAppearance(t *testing.T) {
	env := newPgaMempoolTestEnv()
	h9 := env.forcePushMockTx(9, defaultArrival)
	h5 := env.forcePushMockTx(5, defaultArrival)
	// Three equal-priority entries pushed out of arrival order.
	h7c := env.forcePushMockTx(7, defaultArrival.Add(2*time.Millisecond))
	h7a := env.forcePushMockTx(7, defaultArrival.Add(0*time.Millisecond))
	h7b := env.forcePushMockTx(7, defaultArrival.Add(1*time.Millisecond))
	want := []int{h9, h7a, h7b, h7c, h5}
	for i, wh := range want {
		got := env.mempool.Pop()
		if got.id != wh {
			t.Fatalf("pop %d: got id %d, want %d", i, got.id, wh)
		}
	}
	if env.mempool.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", env.mempool.Len())
	}
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

	if env.mempool.Len() != 3 {
		t.Fatalf("len = %d, want 3", env.mempool.Len())
	}
	if len(env.ch) != 0 {
		t.Fatalf("waiting list not drained: %d remaining", len(env.ch))
	}
	for _, r := range []chan error{r1, r2, r3} {
		expectNoResult(t, r)
	}
	// pop order by priority: capBound (20), tipBound (5), zeroTip (0)
	for _, want := range []int{capBound.id, tipBound.id, zeroTip.id} {
		if got := env.mempool.Pop(); got.id != want {
			t.Fatalf("popped %d, want %d", got.id, want)
		}
	}

	// second drain on the empty channel must return, not block
	env.mempool.StartNewBlock(base, 1000)
	if env.mempool.Len() != 0 {
		t.Fatalf("len = %d after second StartNewBlock, want 0", env.mempool.Len())
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
	env.mempool.Pop()
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
	if env.mempool.Len() != 1 {
		t.Fatalf("after round 1: len = %d, want 1", env.mempool.Len())
	}

	// Round 2 of the same block: a higher-tip arrival is promoted using the
	// basefee and maxsize StartNewBlock recorded, without a new StartNewBlock.
	round2, _ := env.makePgaTestItem(context.Background(), constFee(50), 10, defaultArrival)
	env.ch <- round2
	env.mempool.StartNewPGARound()
	if env.mempool.Len() != 2 {
		t.Fatalf("after round 2: len = %d, want 2", env.mempool.Len())
	}

	// Highest priority pops first, across rounds.
	if got := env.mempool.Pop(); got.id != round2.id {
		t.Fatalf("first pop = %d, want round2", got.id)
	}
	if got := env.mempool.Pop(); got.id != round1.id {
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

	if env.mempool.Len() != testChanCap {
		t.Fatalf("promoted %d, want exactly the snapshot of %d", env.mempool.Len(), testChanCap)
	}
}

func TestPgaMempoolStartNewBlockRejectsExpiredContext(t *testing.T) {
	env := newPgaMempoolTestEnv()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	bad, badResult := env.makePgaTestItem(canceledCtx, constFee(5), 10, defaultArrival)
	good, goodResult := env.makePgaTestItem(context.Background(), constFee(5), 10, defaultArrival)
	env.ch <- bad
	env.ch <- good

	env.mempool.StartNewBlock(big.NewInt(40), 1000)

	expectResult(t, badResult, context.Canceled)
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	expectNoResult(t, goodResult)
}

func TestPgaMempoolStartNewBlockRejectsOversizedTx(t *testing.T) {
	env := newPgaMempoolTestEnv()
	const maxSize = 100

	oversized, oversizedResult := env.makePgaTestItem(context.Background(), constFee(5), maxSize+1, defaultArrival)
	atLimit, atLimitResult := env.makePgaTestItem(context.Background(), constFee(5), maxSize, defaultArrival) // strict >, so allowed
	env.ch <- oversized
	env.ch <- atLimit

	env.mempool.StartNewBlock(big.NewInt(40), maxSize)

	expectResult(t, oversizedResult, txpool.ErrOversizedData)
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	expectNoResult(t, atLimitResult)
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
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	expectNoResult(t, atBaseResult)
	if got := env.mempool.Pop(); got.id != atBase.id {
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
	top := env.mempool.Pop()
	if top.id != idB {
		t.Fatalf("before re-key: top = %d, want B", top.id)
	}
	env.mempool.Push(top) // restore for the re-key

	// A second StartNewBlock against a higher basefee re-keys the queued txs.
	env.mempool.StartNewBlock(big.NewInt(55), 1000) // base 55: A=10, B=5 -> A first

	expectNoResult(t, resultA)
	expectNoResult(t, resultB)
	if got := env.mempool.Pop(); got.id != idA {
		t.Fatalf("after re-key: top = %d, want A", got.id)
	}
	if got := env.mempool.Pop(); got.id != idB {
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
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	if got := env.mempool.Pop(); got.id != idA {
		t.Fatalf("survivor = %d, want A", got.id)
	}
}

func TestPgaMempoolStartNewBlockSweepsExpiredContexts(t *testing.T) {
	env := newPgaMempoolTestEnv()

	ctxA, cancelA := context.WithCancel(context.Background())
	itemA, resultA := env.makePgaTestItem(ctxA, constFee(10), 10, defaultArrival)
	itemB, resultB := env.makePgaTestItem(context.Background(), constFee(20), 10, defaultArrival)
	idB := itemB.id
	env.ch <- itemA
	env.ch <- itemB
	env.mempool.StartNewBlock(big.NewInt(40), 1000) // A priority 10, B priority 20

	cancelA()
	// A second StartNewBlock sweeps the queued tx whose context expired.
	env.mempool.StartNewBlock(big.NewInt(40), 1000) // same basefee; A swept out by its canceled context

	expectResult(t, resultA, context.Canceled)
	expectNoResult(t, resultB)
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	if got := env.mempool.Pop(); got.id != idB {
		t.Fatalf("survivor = %d, want B", got.id)
	}
}

func TestPgaMempoolStartNewBlockDropsOversizedAfterMaxSizeDecrease(t *testing.T) {
	env := newPgaMempoolTestEnv()
	base := big.NewInt(40)

	small, smallResult := env.makePgaTestItem(context.Background(), constFee(10), 50, defaultArrival)  // 50 bytes
	large, largeResult := env.makePgaTestItem(context.Background(), constFee(20), 100, defaultArrival) // 100 bytes
	idSmall := small.id
	env.ch <- small
	env.ch <- large
	env.mempool.StartNewBlock(base, 100) // both fit under the initial max size of 100
	if env.mempool.Len() != 2 {
		t.Fatalf("after first StartNewBlock: len = %d, want 2", env.mempool.Len())
	}

	// A later block shrinks the max transaction size (it is hot-reloadable). The
	// queued tx that no longer fits is dropped on re-key, even though it was
	// accepted under the old limit.
	env.mempool.StartNewBlock(base, 99) // max size now 99: large (100) drops, small (50) stays

	expectResult(t, largeResult, txpool.ErrOversizedData)
	expectNoResult(t, smallResult)
	if env.mempool.Len() != 1 {
		t.Fatalf("len = %d, want 1", env.mempool.Len())
	}
	if got := env.mempool.Pop(); got.id != idSmall {
		t.Fatalf("survivor = %d, want small", got.id)
	}
}

func TestPgaMempoolPeek(t *testing.T) {
	env := newPgaMempoolTestEnv()

	low, _ := env.makePgaTestItem(context.Background(), constFee(10), 10, defaultArrival)
	high, _ := env.makePgaTestItem(context.Background(), constFee(50), 10, defaultArrival)
	env.ch <- low
	env.ch <- high
	env.mempool.StartNewBlock(big.NewInt(40), 1000)

	// Peek returns the highest-priority tx without removing it.
	if got := env.mempool.Peek(); got.id != high.id {
		t.Fatalf("peek = %d, want high", got.id)
	}
	if env.mempool.Len() != 2 {
		t.Fatalf("len = %d after peek, want 2", env.mempool.Len())
	}
	if got := env.mempool.Pop(); got.id != high.id {
		t.Fatalf("pop after peek = %d, want high", got.id)
	}
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

	top := env.mempool.Pop()
	if top.id != idA {
		t.Fatalf("first pop = %d, want A", top.id)
	}
	env.mempool.Push(top) // push back; re-keyed to the same priority, so it pops first again

	if again := env.mempool.Pop(); again.id != idA {
		t.Fatalf("after push-back = %d, want A", again.id)
	}
	for _, want := range []int{idC, idB} {
		if got := env.mempool.Pop(); got.id != want {
			t.Fatalf("remaining pop = %d, want %d", got.id, want)
		}
	}
	if env.mempool.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", env.mempool.Len())
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
	env.mempool.Push(item)

	expectResult(t, result, errFeeCapTooLow)
	if env.mempool.Len() != 0 {
		t.Fatalf("len = %d, want 0 (a failed re-key should drop the tx, not queue it)", env.mempool.Len())
	}
}

func TestPgaMempoolEmptyOps(t *testing.T) {
	env := newPgaMempoolTestEnv() // empty waiting list
	if env.mempool.Len() != 0 {
		t.Fatalf("fresh len = %d, want 0", env.mempool.Len())
	}
	// StartNewBlock must not block on the empty channel and must not panic.
	env.mempool.StartNewBlock(big.NewInt(40), 1000)
	if env.mempool.Len() != 0 {
		t.Fatalf("len = %d after no-op StartNewBlock, want 0", env.mempool.Len())
	}
}
