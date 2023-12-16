package discretetimeboost

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDiscreteTimeBoost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	blockOutputFeed := make(chan *block, 100)
	seq := newSequencer(blockOutputFeed)
	go seq.start(ctx)

	time.Sleep(time.Millisecond * 10)

	// Produce txs.
	go func() {
		id := 0
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for i := 0; i < 10; i++ {
					seq.recv <- &sequencerQueuedTx{
						id:              fmt.Sprintf("%d", id),
						gasToUse:        5,
						firstAppearance: time.Now(),
						gasTipCap:       0,
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Consume blocks from the output feed.
	for blk := range blockOutputFeed {
		t.Logf("Block %+v", blk)
	}
}

// func TestDiscreteTimeBoost(t *testing.T) {
// 	txInputFeed := make(chan boostableTx, 10)
// 	txOutputFeed := make(chan boostableTx, 10)
// 	srv := newTimeBoostService(
// 		txInputFeed,
// 		txOutputFeed,
// 	)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	go srv.run(ctx)

// 	now := time.Now()

// 	testCases := []struct {
// 		name     string
// 		inputTxs []*mockTx
// 		wantIds  []string
// 	}{
// 		{
// 			name: "normalization, no bid orders by timestamp",
// 			inputTxs: []*mockTx{
// 				{
// 					_id:        "a",
// 					_bid:       0,
// 					_timestamp: now.Add(time.Millisecond * 150),
// 				},
// 				{
// 					_id:        "b",
// 					_bid:       0,
// 					_timestamp: now,
// 				},
// 				{
// 					_id:        "c",
// 					_bid:       0,
// 					_timestamp: now.Add(time.Millisecond * 100),
// 				},
// 				{
// 					_id:        "d",
// 					_bid:       0,
// 					_timestamp: now.Add(time.Millisecond * 200),
// 				},
// 			},
// 			wantIds: []string{"b", "c", "a", "d"},
// 		},
// 		{
// 			name: "order by bid",
// 			inputTxs: []*mockTx{
// 				{
// 					_id:        "a",
// 					_bid:       0,
// 					_timestamp: now,
// 				},
// 				{
// 					_id:        "b",
// 					_bid:       100,
// 					_timestamp: now.Add(time.Millisecond * 100),
// 				},
// 				{
// 					_id:        "c",
// 					_bid:       200,
// 					_timestamp: now.Add(time.Millisecond * 150),
// 				},
// 				{
// 					_id:        "d",
// 					_bid:       300,
// 					_timestamp: now.Add(time.Millisecond * 200),
// 				},
// 			},
// 			wantIds: []string{"d", "c", "b", "a"},
// 		},
// 		{
// 			name: "timestamp tiebreakers",
// 			inputTxs: []*mockTx{
// 				{
// 					_id:        "a",
// 					_bid:       100,
// 					_timestamp: now.Add(time.Millisecond * 150),
// 				},
// 				{
// 					_id:        "b",
// 					_bid:       100,
// 					_timestamp: now,
// 				},
// 				{
// 					_id:        "c",
// 					_bid:       200,
// 					_timestamp: now.Add(time.Millisecond * 200),
// 				},
// 				{
// 					_id:        "d",
// 					_bid:       200,
// 					_timestamp: now.Add(time.Millisecond * 100),
// 				},
// 			},
// 			wantIds: []string{"d", "c", "b", "a"},
// 		},
// 	}

// 	for _, tt := range testCases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			for _, tx := range tt.inputTxs {
// 				txInputFeed <- tx
// 			}

// 			gotTxs := make([]*mockTx, 0)
// 			for i := 0; i < len(tt.inputTxs); i++ {
// 				item := <-txOutputFeed
// 				gotTxs = append(gotTxs, item.(*mockTx))
// 			}

// 			for i, tx := range gotTxs {
// 				if tt.wantIds[i] != tx._id {
// 					t.Errorf("i=%d, wanted %s, got %s", i, tt.wantIds[i], tx._id)
// 				}
// 			}
// 		})
// 	}
// }

// func TestDiscreteTimeBoost_CannotGainAdvantageAcrossRounds(t *testing.T) {
// 	txInputFeed := make(chan boostableTx, 10)
// 	txOutputFeed := make(chan boostableTx, 10)
// 	srv := newTimeBoostService(
// 		txInputFeed,
// 		txOutputFeed,
// 	)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	go srv.run(ctx)

// 	now := time.Now()

// 	inputTxs := []*mockTx{
// 		{
// 			_id:        "a",
// 			_bid:       0,
// 			_timestamp: now,
// 		},
// 		{
// 			_id:        "b",
// 			_bid:       100,
// 			_timestamp: now.Add(time.Millisecond * 100),
// 		},
// 		{
// 			_id:        "c",
// 			_bid:       200,
// 			_timestamp: now.Add(time.Millisecond * 150),
// 		},
// 		{
// 			_id:        "d",
// 			_bid:       300,
// 			_timestamp: now.Add(time.Millisecond * 200),
// 		},
// 		{
// 			_id:        "e",
// 			_bid:       300,
// 			_timestamp: now.Add(time.Millisecond * 150),
// 		},
// 		{
// 			_id:        "f",
// 			_bid:       400,
// 			_timestamp: now.Add(time.Millisecond * 200),
// 		},
// 	}

// 	for _, tx := range inputTxs[0:2] {
// 		txInputFeed <- tx
// 	}

// 	gotTxs := make([]*mockTx, 0)

// 	for i := 0; i < 2; i++ {
// 		item := <-txOutputFeed
// 		gotTxs = append(gotTxs, item.(*mockTx))
// 	}

// 	srv.startNextRound()

// 	for _, tx := range inputTxs[2:4] {
// 		txInputFeed <- tx
// 	}

// 	for i := 0; i < 2; i++ {
// 		item := <-txOutputFeed
// 		gotTxs = append(gotTxs, item.(*mockTx))
// 	}

// 	srv.startNextRound()

// 	for _, tx := range inputTxs[4:6] {
// 		txInputFeed <- tx
// 	}

// 	for i := 0; i < 2; i++ {
// 		item := <-txOutputFeed
// 		gotTxs = append(gotTxs, item.(*mockTx))
// 	}

// 	wantIds := []string{"b", "a", "d", "c", "f", "e"}
// 	for i, tx := range gotTxs {
// 		if wantIds[i] != tx._id {
// 			t.Errorf("i=%d, wanted %s, got %s", i, wantIds[i], tx._id)
// 		}
// 	}
// }
