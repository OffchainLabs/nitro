package execution

import (
	"container/heap"
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var _ heap.Interface = (*timeBoostableTxs)(nil)

type mockTx struct {
	_id        string
	_timestamp time.Time
	_bid       uint64
	_innerTx   *types.Transaction
}

func (m *mockTx) id() string {
	return m._id
}

func (m *mockTx) bid() uint64 {
	return m._bid
}

func (m *mockTx) timestamp() time.Time {
	return m._timestamp
}

func (m *mockTx) innerTx() *types.Transaction {
	return m._innerTx
}

func TestDiscreteTimeBoost(t *testing.T) {
	txInputFeed := make(chan boostableTx, 10)
	txOutputFeed := make(chan boostableTx, 10)
	srv := newTimeBoostService(
		txInputFeed,
		txOutputFeed,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.run(ctx)

	now := time.Now()

	testCases := []struct {
		name     string
		inputTxs []*mockTx
		wantIds  []string
	}{
		{
			name: "normalization, no bid orders by timestamp",
			inputTxs: []*mockTx{
				{
					_id:        "a",
					_bid:       0,
					_timestamp: now.Add(time.Millisecond * 150),
				},
				{
					_id:        "b",
					_bid:       0,
					_timestamp: now,
				},
				{
					_id:        "c",
					_bid:       0,
					_timestamp: now.Add(time.Millisecond * 100),
				},
				{
					_id:        "d",
					_bid:       0,
					_timestamp: now.Add(time.Millisecond * 200),
				},
			},
			wantIds: []string{"b", "c", "a", "d"},
		},
		{
			name: "order by bid",
			inputTxs: []*mockTx{
				{
					_id:        "a",
					_bid:       0,
					_timestamp: now,
				},
				{
					_id:        "b",
					_bid:       100,
					_timestamp: now.Add(time.Millisecond * 100),
				},
				{
					_id:        "c",
					_bid:       200,
					_timestamp: now.Add(time.Millisecond * 150),
				},
				{
					_id:        "d",
					_bid:       300,
					_timestamp: now.Add(time.Millisecond * 200),
				},
			},
			wantIds: []string{"d", "c", "b", "a"},
		},
		{
			name: "timestamp tiebreakers",
			inputTxs: []*mockTx{
				{
					_id:        "a",
					_bid:       100,
					_timestamp: now.Add(time.Millisecond * 150),
				},
				{
					_id:        "b",
					_bid:       100,
					_timestamp: now,
				},
				{
					_id:        "c",
					_bid:       200,
					_timestamp: now.Add(time.Millisecond * 200),
				},
				{
					_id:        "d",
					_bid:       200,
					_timestamp: now.Add(time.Millisecond * 100),
				},
			},
			wantIds: []string{"d", "c", "b", "a"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			for _, tx := range tt.inputTxs {
				txInputFeed <- tx
			}

			gotTxs := make([]*mockTx, 0)
			for i := 0; i < len(tt.inputTxs); i++ {
				item := <-txOutputFeed
				gotTxs = append(gotTxs, item.(*mockTx))
			}

			for i, tx := range gotTxs {
				if tt.wantIds[i] != tx._id {
					t.Errorf("i=%d, wanted %s, got %s", i, tt.wantIds[i], tx._id)
				}
			}
		})
	}
}

func TestDiscreteTimeBoost_CannotGainAdvantageAcrossRounds(t *testing.T) {
	txInputFeed := make(chan boostableTx, 10)
	txOutputFeed := make(chan boostableTx, 10)
	srv := newTimeBoostService(
		txInputFeed,
		txOutputFeed,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.run(ctx)

	now := time.Now()

	inputTxs := []*mockTx{
		{
			_id:        "a",
			_bid:       0,
			_timestamp: now,
		},
		{
			_id:        "b",
			_bid:       100,
			_timestamp: now.Add(time.Millisecond * 100),
		},
		{
			_id:        "c",
			_bid:       200,
			_timestamp: now.Add(time.Millisecond * 150),
		},
		{
			_id:        "d",
			_bid:       300,
			_timestamp: now.Add(time.Millisecond * 200),
		},
		{
			_id:        "e",
			_bid:       300,
			_timestamp: now.Add(time.Millisecond * 150),
		},
		{
			_id:        "f",
			_bid:       400,
			_timestamp: now.Add(time.Millisecond * 200),
		},
	}

	for _, tx := range inputTxs[0:2] {
		txInputFeed <- tx
	}

	gotTxs := make([]*mockTx, 0)

	for i := 0; i < 2; i++ {
		item := <-txOutputFeed
		gotTxs = append(gotTxs, item.(*mockTx))
	}

	srv.startNextRound()

	for _, tx := range inputTxs[2:4] {
		txInputFeed <- tx
	}

	for i := 0; i < 2; i++ {
		item := <-txOutputFeed
		gotTxs = append(gotTxs, item.(*mockTx))
	}

	srv.startNextRound()

	for _, tx := range inputTxs[4:6] {
		txInputFeed <- tx
	}

	for i := 0; i < 2; i++ {
		item := <-txOutputFeed
		gotTxs = append(gotTxs, item.(*mockTx))
	}

	wantIds := []string{"b", "a", "d", "c", "f", "e"}
	for i, tx := range gotTxs {
		if wantIds[i] != tx._id {
			t.Errorf("i=%d, wanted %s, got %s", i, wantIds[i], tx._id)
		}
	}
}

func TestTxPriorityQueue(t *testing.T) {
	txs := timeBoostableTxs(make([]boostableTx, 0))
	heap.Init(&txs)

	t.Run("order by bid", func(t *testing.T) {
		now := time.Now()
		heap.Push(&txs, &mockTx{
			_bid:       0,
			_timestamp: now,
		})
		heap.Push(&txs, &mockTx{
			_bid:       100,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		got := make([]*mockTx, 0)
		for txs.Len() > 0 {
			tx := heap.Pop(&txs).(*mockTx)
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
		now := time.Now()
		heap.Push(&txs, &mockTx{
			_id:        "a",
			_bid:       100,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		heap.Push(&txs, &mockTx{
			_id:        "b",
			_bid:       100,
			_timestamp: now,
		})
		got := make([]*mockTx, 0)
		for txs.Len() > 0 {
			tx := heap.Pop(&txs).(*mockTx)
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
	t.Run("no bid, order by timestamp", func(t *testing.T) {
		now := time.Now()
		heap.Push(&txs, &mockTx{
			_id:        "a",
			_bid:       0,
			_timestamp: now.Add(time.Millisecond * 100),
		})
		heap.Push(&txs, &mockTx{
			_id:        "b",
			_bid:       0,
			_timestamp: now,
		})
		heap.Push(&txs, &mockTx{
			_id:        "c",
			_bid:       0,
			_timestamp: now.Add(time.Millisecond * 200),
		})
		got := make([]*mockTx, 0)
		for txs.Len() > 0 {
			tx := heap.Pop(&txs).(*mockTx)
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
