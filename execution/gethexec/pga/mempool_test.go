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

// errFeeCapTooLow stands in for the fee-cap-below-basefee error that the real ComputePriorityFee returns.
// The mempool only propagates it, so its identity is all that matters here.
var errFeeCapTooLow = errors.New("fee cap below base fee")

// defaultArrival is the arrival timestamp shared by the test txs. Most tests care only about priority order, not arrival time, so
// they reuse it directly; the tie-break test offsets it where arrival order matters.
var defaultArrival = time.Unix(0, 0)

// priorityFeeFunc computes a transaction's priority fee against a basefee.
type priorityFeeFunc func(baseFee *big.Int) (uint64, error)

func constFee(fee uint64) priorityFeeFunc {
	return func(*big.Int) (uint64, error) { return fee, nil }
}

func failFee(err error) priorityFeeFunc {
	return func(*big.Int) (uint64, error) { return 0, err }
}

// mockTx implements Tx for the mempool tests with value receivers, so the mempool can be instantiated by value (Mempool[mockTx]).
type mockTx struct {
	id              int
	fee             priorityFeeFunc
	size            int
	ctx             context.Context
	firstAppearance time.Time
	resultChan      chan error
	returnedResult  *atomic.Bool
}

func (m mockTx) ComputePriorityFee(baseFee *big.Int) (uint64, error) { return m.fee(baseFee) }

func (m mockTx) ReturnResult(err error) {
	if m.returnedResult.Swap(true) {
		return
	}
	m.resultChan <- err
	close(m.resultChan)
}

func (m mockTx) GetContext() context.Context { return m.ctx }

func (m mockTx) GetSize() int { return m.size }

func (m mockTx) GetFirstAppearance() time.Time { return m.firstAppearance }

// makePgaTestItem builds a mockTx like publishTransactionToQueue builds a txQueueItem, and returns its result channel
// so tests can assert what the mempool sent back.
func makePgaTestItem(ctx context.Context, id int, fee priorityFeeFunc, size int, firstAppearance time.Time) (mockTx, chan error) {
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

// makeMockPusher returns a function that inserts an entry with an explicit priority (bypassing setPriority), assigning
// each a distinct auto-incremented id, and returns that id for identification.
func makeMockPusher() func(m *Mempool[mockTx], priority uint64, firstAppearance time.Time) int {
	nextID := 0
	return func(m *Mempool[mockTx], priority uint64, firstAppearance time.Time) int {
		id := nextID
		nextID++
		item, _ := makePgaTestItem(context.Background(), id, constFee(priority), 0, firstAppearance)
		heap.Push(&m.heap, txItem[mockTx]{tx: item, priority: priority})
		return id
	}
}

// promote starts a block at baseFee and runs its first PGA round, the common setup for these tests.
func promote(m *Mempool[mockTx], baseFee *big.Int, maxTxDataSize int) {
	m.StartNewBlock(baseFee, maxTxDataSize)
	m.StartNewPGARound()
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

func TestPgaTxItemSetPriority(t *testing.T) {
	t.Run("sets priority from the computed fee", func(t *testing.T) {
		item := txItem[mockTx]{tx: mockTx{fee: constFee(42)}}
		if err := item.setPriority(big.NewInt(7)); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if item.priority != 42 {
			t.Fatalf("priority = %d, want 42", item.priority)
		}
	})

	t.Run("forwards the basefee", func(t *testing.T) {
		var seen *big.Int
		item := txItem[mockTx]{tx: mockTx{fee: func(baseFee *big.Int) (uint64, error) {
			seen = baseFee
			return 0, nil
		}}}
		if err := item.setPriority(big.NewInt(99)); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if seen == nil || seen.Int64() != 99 {
			t.Fatalf("basefee passed to ComputePriorityFee = %v, want 99", seen)
		}
	})

	t.Run("propagates the error", func(t *testing.T) {
		item := txItem[mockTx]{tx: mockTx{fee: failFee(errFeeCapTooLow)}}
		if err := item.setPriority(big.NewInt(7)); !errors.Is(err, errFeeCapTooLow) {
			t.Fatalf("err = %v, want errors.Is(errFeeCapTooLow)", err)
		}
	})
}

func TestPgaMempoolPopOrdersByPriorityDesc(t *testing.T) {
	m := NewMempool[mockTx](nil)
	pushRaw := makeMockPusher()
	priorities := []uint64{5, 1, 9, 7, 3}
	idByPriority := make(map[uint64]int, len(priorities))
	for _, p := range priorities {
		idByPriority[p] = pushRaw(m, p, defaultArrival)
	}
	for _, want := range []uint64{9, 7, 5, 3, 1} {
		if m.Len() == 0 {
			t.Fatalf("queue emptied before popping priority %d", want)
		}
		if got := m.Pop(); got.id != idByPriority[want] {
			t.Fatalf("popped wrong tx for priority %d", want)
		}
	}
	if m.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", m.Len())
	}
}

func TestPgaMempoolPopBreaksTiesByFirstAppearance(t *testing.T) {
	m := NewMempool[mockTx](nil)
	pushRaw := makeMockPusher()
	h9 := pushRaw(m, 9, defaultArrival)
	h5 := pushRaw(m, 5, defaultArrival)
	// Three equal-priority entries pushed out of arrival order.
	h7c := pushRaw(m, 7, defaultArrival.Add(2*time.Millisecond))
	h7a := pushRaw(m, 7, defaultArrival.Add(0*time.Millisecond))
	h7b := pushRaw(m, 7, defaultArrival.Add(1*time.Millisecond))
	want := []int{h9, h7a, h7b, h7c, h5}
	for i, wh := range want {
		got := m.Pop()
		if got.id != wh {
			t.Fatalf("pop %d: got id %d, want %d", i, got.id, wh)
		}
	}
	if m.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", m.Len())
	}
}

func TestPgaMempoolPromoteDrainsWaitingList(t *testing.T) {
	ch := make(chan mockTx, 8)
	m := NewMempool(ch)
	base := big.NewInt(40)

	tipBound, r1 := makePgaTestItem(context.Background(), 0, constFee(5), 10, defaultArrival)
	capBound, r2 := makePgaTestItem(context.Background(), 1, constFee(20), 10, defaultArrival)
	zeroTip, r3 := makePgaTestItem(context.Background(), 2, constFee(0), 10, defaultArrival)
	ch <- tipBound
	ch <- capBound
	ch <- zeroTip

	promote(m, base, 1000)

	if m.Len() != 3 {
		t.Fatalf("len = %d, want 3", m.Len())
	}
	if len(ch) != 0 {
		t.Fatalf("waiting list not drained: %d remaining", len(ch))
	}
	for _, r := range []chan error{r1, r2, r3} {
		expectNoResult(t, r)
	}
	// pop order by priority: capBound (20), tipBound (5), zeroTip (0)
	for _, want := range []int{capBound.id, tipBound.id, zeroTip.id} {
		if got := m.Pop(); got.id != want {
			t.Fatalf("popped %d, want %d", got.id, want)
		}
	}

	// second drain on the empty channel must return, not block
	promote(m, base, 1000)
	if m.Len() != 0 {
		t.Fatalf("len = %d after second promote, want 0", m.Len())
	}
}

func TestPgaMempoolAreThereTxsForNextRound(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	// Empty mempool and empty channel: nothing for the next round.
	if m.AreThereTxsForNextRound() {
		t.Fatal("empty mempool should report no txs for the next round")
	}

	// A waiting-list arrival counts, even before promotion.
	item, _ := makePgaTestItem(context.Background(), 0, constFee(10), 10, defaultArrival)
	ch <- item
	if !m.AreThereTxsForNextRound() {
		t.Fatal("a waiting-list tx should count for the next round")
	}

	// After promotion it's in the heap and the channel is empty; still counts.
	promote(m, big.NewInt(40), 1000)
	if len(ch) != 0 {
		t.Fatalf("channel not drained: %d", len(ch))
	}
	if !m.AreThereTxsForNextRound() {
		t.Fatal("a queued tx should count for the next round")
	}

	// Drain the queue: nothing left.
	m.Pop()
	if m.AreThereTxsForNextRound() {
		t.Fatal("fully drained mempool should report no txs for the next round")
	}
}

func TestPgaMempoolMultipleRoundsPerBlock(t *testing.T) {
	ch := make(chan mockTx, 8)
	m := NewMempool(ch)

	// Round 1: one arrival, drained against the block's basefee and maxsize.
	round1, _ := makePgaTestItem(context.Background(), 0, constFee(10), 10, defaultArrival)
	ch <- round1
	m.StartNewBlock(big.NewInt(40), 1000)
	m.StartNewPGARound()
	if m.Len() != 1 {
		t.Fatalf("after round 1: len = %d, want 1", m.Len())
	}

	// Round 2 of the same block: a higher-tip arrival is promoted using the
	// basefee and maxsize StartNewBlock recorded, without a new StartNewBlock.
	round2, _ := makePgaTestItem(context.Background(), 1, constFee(50), 10, defaultArrival)
	ch <- round2
	m.StartNewPGARound()
	if m.Len() != 2 {
		t.Fatalf("after round 2: len = %d, want 2", m.Len())
	}

	// Highest priority pops first, across rounds.
	if got := m.Pop(); got.id != round2.id {
		t.Fatalf("first pop = %d, want round2", got.id)
	}
	if got := m.Pop(); got.id != round1.id {
		t.Fatalf("second pop = %d, want round1", got.id)
	}
}

func TestPgaMempoolPromoteDrainsOnlySnapshot(t *testing.T) {
	const capacity = 4
	ch := make(chan mockTx, capacity)
	m := NewMempool(ch)
	// Fill the buffer so the snapshot at promotion start is exactly capacity.
	for i := range capacity {
		item, _ := makePgaTestItem(context.Background(), i, constFee(5), 10, defaultArrival)
		ch <- item
	}
	// A producer that keeps offering more: these arrive during promotion and
	// must wait for the next round rather than join the current one.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := capacity; ; i++ {
			item, _ := makePgaTestItem(context.Background(), i, constFee(5), 10, defaultArrival)
			select {
			case ch <- item:
			case <-stop:
				return
			}
		}
	}()

	promote(m, big.NewInt(40), 1000)
	close(stop)
	<-done

	if m.Len() != capacity {
		t.Fatalf("promoted %d, want exactly the snapshot of %d", m.Len(), capacity)
	}
}

func TestPgaMempoolPromoteRejectsExpiredContext(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	bad, badResult := makePgaTestItem(canceledCtx, 0, constFee(5), 10, defaultArrival)
	good, goodResult := makePgaTestItem(context.Background(), 1, constFee(5), 10, defaultArrival)
	ch <- bad
	ch <- good

	promote(m, big.NewInt(40), 1000)

	expectResult(t, badResult, context.Canceled)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	expectNoResult(t, goodResult)
}

func TestPgaMempoolPromoteRejectsOversizedTx(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)
	const maxSize = 100

	oversized, oversizedResult := makePgaTestItem(context.Background(), 0, constFee(5), maxSize+1, defaultArrival)
	atLimit, atLimitResult := makePgaTestItem(context.Background(), 1, constFee(5), maxSize, defaultArrival) // strict >, so allowed
	ch <- oversized
	ch <- atLimit

	promote(m, big.NewInt(40), maxSize)

	expectResult(t, oversizedResult, txpool.ErrOversizedData)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	expectNoResult(t, atLimitResult)
}

func TestPgaMempoolPromoteRejectsFeeCapBelowBaseFee(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)
	base := big.NewInt(40)

	low, lowResult := makePgaTestItem(context.Background(), 0, failFee(errFeeCapTooLow), 10, defaultArrival) // fee cap below base
	atBase, atBaseResult := makePgaTestItem(context.Background(), 1, constFee(0), 10, defaultArrival)        // fee cap == base, priority 0
	ch <- low
	ch <- atBase

	promote(m, base, 1000)

	expectResult(t, lowResult, errFeeCapTooLow)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	expectNoResult(t, atBaseResult)
	if got := m.Pop(); got.id != atBase.id {
		t.Fatalf("survivor = %d, want atBase", got.id)
	}
}

func TestPgaMempoolPromoteRecomputesAgainstNewBaseFee(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	// A is tip-bound (constant), B is cap-bound; their order flips with the
	// basefee: B outranks A at base 10 and falls below it at base 55.
	itemA, resultA := makePgaTestItem(context.Background(), 0, constFee(10), 10, defaultArrival)
	itemB, resultB := makePgaTestItem(context.Background(), 1, func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(10)) == 0 {
			return 50, nil
		}
		return 5, nil
	}, 10, defaultArrival)
	idA, idB := itemA.id, itemB.id
	ch <- itemA
	ch <- itemB

	promote(m, big.NewInt(10), 1000) // base 10: A=10, B=50 -> B first
	top := m.Pop()
	if top.id != idB {
		t.Fatalf("before re-key: top = %d, want B", top.id)
	}
	m.Push(top) // restore for the re-key

	// A second Promote against a higher basefee re-keys the queued txs.
	promote(m, big.NewInt(55), 1000) // base 55: A=10, B=5 -> A first

	expectNoResult(t, resultA)
	expectNoResult(t, resultB)
	if got := m.Pop(); got.id != idA {
		t.Fatalf("after re-key: top = %d, want A", got.id)
	}
	if got := m.Pop(); got.id != idB {
		t.Fatalf("after re-key: second = %d, want B", got.id)
	}
}

func TestPgaMempoolPromoteDropsRequeuedFeeCapTooLow(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	itemA, resultA := makePgaTestItem(context.Background(), 0, constFee(10), 10, defaultArrival)
	// B is valid at the first basefee but its fee cap falls below the second.
	itemB, resultB := makePgaTestItem(context.Background(), 1, func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(60)) >= 0 {
			return 0, errFeeCapTooLow
		}
		return 10, nil
	}, 10, defaultArrival)
	idA := itemA.id
	ch <- itemA
	ch <- itemB
	promote(m, big.NewInt(40), 1000) // both valid at base 40

	// A second Promote against a higher basefee drops the requeued tx whose fee
	// cap is now too low.
	promote(m, big.NewInt(60), 1000) // base 60: A stays, B drops

	expectResult(t, resultB, errFeeCapTooLow)
	expectNoResult(t, resultA)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	if got := m.Pop(); got.id != idA {
		t.Fatalf("survivor = %d, want A", got.id)
	}
}

func TestPgaMempoolPromoteSweepsExpiredContexts(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	ctxA, cancelA := context.WithCancel(context.Background())
	itemA, resultA := makePgaTestItem(ctxA, 0, constFee(10), 10, defaultArrival)
	itemB, resultB := makePgaTestItem(context.Background(), 1, constFee(20), 10, defaultArrival)
	idB := itemB.id
	ch <- itemA
	ch <- itemB
	promote(m, big.NewInt(40), 1000) // A priority 10, B priority 20

	cancelA()
	// A second Promote sweeps the queued tx whose context expired.
	promote(m, big.NewInt(40), 1000) // same basefee; A swept out by its canceled context

	expectResult(t, resultA, context.Canceled)
	expectNoResult(t, resultB)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	if got := m.Pop(); got.id != idB {
		t.Fatalf("survivor = %d, want B", got.id)
	}
}

func TestPgaMempoolPromoteDropsOversizedAfterMaxSizeDecrease(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)
	base := big.NewInt(40)

	small, smallResult := makePgaTestItem(context.Background(), 0, constFee(10), 50, defaultArrival)  // 50 bytes
	large, largeResult := makePgaTestItem(context.Background(), 1, constFee(20), 100, defaultArrival) // 100 bytes
	idSmall := small.id
	ch <- small
	ch <- large
	promote(m, base, 100) // both fit under the initial max size of 100
	if m.Len() != 2 {
		t.Fatalf("after first promote: len = %d, want 2", m.Len())
	}

	// A later block shrinks the max transaction size (it is hot-reloadable). The
	// queued tx that no longer fits is dropped on re-key, even though it was
	// accepted under the old limit.
	m.StartNewBlock(base, 99) // max size now 99: large (100) drops, small (50) stays

	expectResult(t, largeResult, txpool.ErrOversizedData)
	expectNoResult(t, smallResult)
	if m.Len() != 1 {
		t.Fatalf("len = %d, want 1", m.Len())
	}
	if got := m.Pop(); got.id != idSmall {
		t.Fatalf("survivor = %d, want small", got.id)
	}
}

func TestPgaMempoolPeek(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	low, _ := makePgaTestItem(context.Background(), 0, constFee(10), 10, defaultArrival)
	high, _ := makePgaTestItem(context.Background(), 1, constFee(50), 10, defaultArrival)
	ch <- low
	ch <- high
	promote(m, big.NewInt(40), 1000)

	// Peek returns the highest-priority tx without removing it.
	if got := m.Peek(); got.id != high.id {
		t.Fatalf("peek = %d, want high", got.id)
	}
	if m.Len() != 2 {
		t.Fatalf("len = %d after peek, want 2", m.Len())
	}
	if got := m.Pop(); got.id != high.id {
		t.Fatalf("pop after peek = %d, want high", got.id)
	}
}

func TestPgaMempoolPushBackKeepsPriority(t *testing.T) {
	ch := make(chan mockTx, 4)
	m := NewMempool(ch)

	itemA, _ := makePgaTestItem(context.Background(), 0, constFee(50), 10, defaultArrival)
	itemB, _ := makePgaTestItem(context.Background(), 1, constFee(10), 10, defaultArrival)
	itemC, _ := makePgaTestItem(context.Background(), 2, constFee(30), 10, defaultArrival)
	idA, idB, idC := itemA.id, itemB.id, itemC.id
	ch <- itemA
	ch <- itemB
	ch <- itemC
	promote(m, big.NewInt(40), 1000)

	top := m.Pop()
	if top.id != idA {
		t.Fatalf("first pop = %d, want A", top.id)
	}
	m.Push(top) // push back; re-keyed to the same priority, so it pops first again

	if again := m.Pop(); again.id != idA {
		t.Fatalf("after push-back = %d, want A", again.id)
	}
	for _, want := range []int{idC, idB} {
		if got := m.Pop(); got.id != want {
			t.Fatalf("remaining pop = %d, want %d", got.id, want)
		}
	}
	if m.Len() != 0 {
		t.Fatalf("queue not empty after draining: len %d", m.Len())
	}
}

// TestPgaMempoolPushDropsOnFeeError covers Push's re-key failure branch: a
// push-back whose ComputePriorityFee errors is dropped with that error rather
// than queued. The basefee is constant within a block, so a re-key that
// succeeded on the way out cannot fail on push-back today; this pins the
// public API's behavior regardless.
func TestPgaMempoolPushDropsOnFeeError(t *testing.T) {
	m := NewMempool[mockTx](nil)
	m.StartNewBlock(big.NewInt(40), 1000) // sets the basefee Push re-keys against

	item, result := makePgaTestItem(context.Background(), 0, failFee(errFeeCapTooLow), 10, defaultArrival)
	m.Push(item)

	expectResult(t, result, errFeeCapTooLow)
	if m.Len() != 0 {
		t.Fatalf("len = %d, want 0 (a failed re-key should drop the tx, not queue it)", m.Len())
	}
}

func TestPgaMempoolEmptyOps(t *testing.T) {
	ch := make(chan mockTx) // unbuffered and empty
	m := NewMempool(ch)
	if m.Len() != 0 {
		t.Fatalf("fresh len = %d, want 0", m.Len())
	}
	// Promote must not block on the empty channel and must not panic.
	promote(m, big.NewInt(40), 1000)
	if m.Len() != 0 {
		t.Fatalf("len = %d after no-op promote, want 0", m.Len())
	}
}
