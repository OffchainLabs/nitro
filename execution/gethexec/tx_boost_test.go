package gethexec

import (
	"container/heap"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	boostpolicies "github.com/offchainlabs/nitro/execution/gethexec/boost-policies"
)

var _ heap.Interface = (*boostableTxs)(nil)

type mockTx struct {
	_id        string
	_timestamp time.Time
	_bid       uint64
}

func (m *mockTx) timestamp() time.Time {
	return m._timestamp
}

func (m *mockTx) innerTx() *types.Transaction {
	inner := &types.DynamicFeeTx{
		GasTipCap: new(big.Int).SetUint64(m._bid),
	}
	return types.NewTx(inner)
}

func TestTxPriorityQueue(t *testing.T) {
	t.Run("order by score", func(t *testing.T) {
		heap := newTxBoostHeap(&boostpolicies.ExpressLaneScorer{})
		now := time.Now()
		heap.Push(&mockTx{
			_bid:       0,
			_timestamp: now,
		})
		heap.Push(&mockTx{
			_bid:       100,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		got := make([]*mockTx, 0)
		for heap.prioQueue.Len() > 0 {
			tx := heap.Pop().(*mockTx)
			got = append(got, tx)
		}
		if len(got) != 2 {
			t.Fatalf("Wanted %d, got %d", 2, len(got))
		}
		if got[0]._bid != uint64(100) {
			t.Fatalf("Wanted %d, got %d", 100, got[0]._bid)
		}
		if got[1]._bid != uint64(0) {
			t.Fatalf("Wanted %d, got %d", 0, got[1]._bid)
		}
	})
	t.Run("tiebreak by timestamp", func(t *testing.T) {
		heap := newTxBoostHeap(&boostpolicies.ExpressLaneScorer{})
		now := time.Now()
		heap.Push(&mockTx{
			_id:        "a",
			_bid:       100,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		heap.Push(&mockTx{
			_id:        "b",
			_bid:       100,
			_timestamp: now,
		})
		got := make([]*mockTx, 0)
		for heap.prioQueue.Len() > 0 {
			tx := heap.Pop().(*mockTx)
			got = append(got, tx)
		}
		if len(got) != 2 {
			t.Fatalf("Wanted %d, got %d", 2, len(got))
		}
		if got[0]._id != "b" {
			t.Fatalf("Wanted %s, got %s", "b", got[0]._id)
		}
		if got[1]._id != "a" {
			t.Fatalf("Wanted %s, got %s", "a", got[1]._id)
		}
	})
	t.Run("express lane scorer, but no bid set, order by timestamp", func(t *testing.T) {
		heap := newTxBoostHeap(&boostpolicies.ExpressLaneScorer{})
		now := time.Now()
		heap.Push(&mockTx{
			_id:        "a",
			_bid:       0,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		heap.Push(&mockTx{
			_id:        "b",
			_bid:       0,
			_timestamp: now,
		})
		heap.Push(&mockTx{
			_id:        "c",
			_bid:       0,
			_timestamp: now.Add(time.Millisecond * 200),
		})
		got := make([]*mockTx, 0)
		for heap.prioQueue.Len() > 0 {
			tx := heap.Pop().(*mockTx)
			got = append(got, tx)
		}
		if len(got) != 3 {
			t.Fatalf("Wanted %d, got %d", 3, len(got))
		}
		if got[0]._id != "b" {
			t.Fatalf("Wanted %s, got %s", "b", got[0]._id)
		}
		if got[1]._id != "a" {
			t.Fatalf("Wanted %s, got %s", "a", got[1]._id)
		}
		if got[2]._id != "c" {
			t.Fatalf("Wanted %s, got %s", "a", got[2]._id)
		}
	})
	t.Run("bid set but using noop scorer should order by timestamp", func(t *testing.T) {
		heap := newTxBoostHeap(&boostpolicies.NoopScorer{})
		now := time.Now()
		heap.Push(&mockTx{
			_id:        "a",
			_bid:       200,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		heap.Push(&mockTx{
			_id:        "b",
			_bid:       0,
			_timestamp: now,
		})
		heap.Push(&mockTx{
			_id:        "c",
			_bid:       300,
			_timestamp: now.Add(time.Millisecond * 200),
		})
		got := make([]*mockTx, 0)
		for heap.prioQueue.Len() > 0 {
			tx := heap.Pop().(*mockTx)
			got = append(got, tx)
		}
		if len(got) != 3 {
			t.Fatalf("Wanted %d, got %d", 3, len(got))
		}
		if got[0]._id != "b" {
			t.Fatalf("Wanted %s, got %s", "b", got[0]._id)
		}
		if got[1]._id != "a" {
			t.Fatalf("Wanted %s, got %s", "a", got[1]._id)
		}
		if got[2]._id != "c" {
			t.Fatalf("Wanted %s, got %s", "a", got[2]._id)
		}
	})
}
