// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

const testReorderWindow = uint64(100)

func dummyTxQueueItem(id uint64) *txQueueItem {
	return &txQueueItem{
		tx: &types.Transaction{CalldataUnits: id},
	}
}

func TestTxGraphEmpty(t *testing.T) {
	testHarnessSliceSource(t, []reorderItem{}, []int{})
}

func TestTxGraphSpaced(t *testing.T) {
	inTxs := []reorderItem{}
	expectedOrder := []int{}
	numTx := 13
	for i := 0; i < numTx; i++ {
		tx := reorderItem{
			uniqueId:        uint64(i),
			timestampMillis: uint64(1000 * i),
			bid:             big.NewInt(int64(100 * i)),
			queueItem:       dummyTxQueueItem(uint64(i)),
		}
		inTxs = append(inTxs, tx)
		expectedOrder = append(expectedOrder, i)
	}

	testHarnessSliceSource(t, inTxs, expectedOrder)
}

func TestTxGraphDense(t *testing.T) {
	inTxs := []reorderItem{}
	expectedOrder := []int{}
	numTx := 13
	for i := 0; i < numTx; i++ {
		tx := reorderItem{
			uniqueId:        uint64(i),
			timestampMillis: uint64(99 * i),
			bid:             big.NewInt(1000 * int64(numTx-i)),
			queueItem:       dummyTxQueueItem(uint64(i)),
		}
		inTxs = append(inTxs, tx)
		expectedOrder = append(expectedOrder, i)
	}

	testHarnessSliceSource(t, inTxs, expectedOrder)
}

func TestTxByBid(t *testing.T) {
	inTxs := []reorderItem{}
	expectedOrder := []int{}
	numTx := 13
	for i := 0; i < numTx; i++ {
		tx := reorderItem{
			uniqueId:        uint64(i),
			timestampMillis: uint64(i),
			bid:             big.NewInt(int64(100 * i)),
			queueItem:       dummyTxQueueItem(uint64(i)),
		}
		inTxs = append(inTxs, tx)
		expectedOrder = append(expectedOrder, numTx-i-1)
	}

	testHarnessSliceSource(t, inTxs, expectedOrder)
}

func TestTxExample1(t *testing.T) {
	inTxs := []reorderItem{
		{
			uniqueId:        0,
			timestampMillis: 0,
			bid:             common.Big0,
			queueItem:       dummyTxQueueItem(0),
		},
		{
			uniqueId:        1,
			timestampMillis: 67,
			bid:             big.NewInt(100),
			queueItem:       dummyTxQueueItem(1),
		},
		{
			uniqueId:        2,
			timestampMillis: 133,
			bid:             big.NewInt(300),
			queueItem:       dummyTxQueueItem(2),
		},
	}
	expectedOrder := []int{0, 2, 1}
	testHarnessSliceSource(t, inTxs, expectedOrder)
}

func TestTxExample2(t *testing.T) {
	inTxs := []reorderItem{
		{
			uniqueId:        0,
			timestampMillis: 0,
			bid:             common.Big0,
			queueItem:       dummyTxQueueItem(0),
		},
		{
			uniqueId:        1,
			timestampMillis: 67,
			bid:             big.NewInt(200),
			queueItem:       dummyTxQueueItem(1),
		},
		{
			uniqueId:        2,
			timestampMillis: 133,
			bid:             big.NewInt(300),
			queueItem:       dummyTxQueueItem(2),
		},
	}
	expectedOrder := []int{1, 0, 2}
	testHarnessSliceSource(t, inTxs, expectedOrder)
}

func testHarnessSliceSource(t *testing.T, inTxs []reorderItem, expectedOrder []int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testHarnessWithSource(t, ctx, inTxs, expectedOrder)
}

func testHarnessWithSource(t *testing.T, ctx context.Context, inTxs []reorderItem, expectedOrder []int) {
	reo := newTxReorderer(newItemSourceSlice(inTxs), int(testReorderWindow), 16)
	go func() {
		reo.run(ctx)
	}()

	numReturned := 0
	for tx := range reo.outChan {
		if tx.tx.CalldataUnits != inTxs[expectedOrder[numReturned]].queueItem.tx.CalldataUnits {
			t.Fatal(numReturned, expectedOrder[numReturned])
		}
		numReturned++
	}
	if numReturned != len(expectedOrder) {
		t.Fatal(numReturned, len(expectedOrder))
	}
}

type tceTestCase struct {
	fromTs         uint64
	fromBid        int64
	toTs           uint64
	toBid          int64
	expectedDir    Direction
	expectedHeavy  bool
	expectedWeight int64
}

func TestComputeEdge(t *testing.T) {
	testCases := []tceTestCase{
		{0, 0, 13, 0, Direction(ForwardDirection), false, 1},
		{0, 0, 101, 0, Direction(ForwardDirection), true, 0},
		{0, 4, 13, 11, Direction(BackwardDirection), false, 7},
	}
	for _, tc := range testCases {
		fromTx := reorderItem{
			uniqueId:        0,
			timestampMillis: tc.fromTs,
			bid:             big.NewInt(tc.fromBid),
			cumulativeLoss:  common.Big0,
			queueItem:       nil,
		}
		toTx := reorderItem{
			uniqueId:        1,
			timestampMillis: tc.toTs,
			bid:             big.NewInt(tc.toBid),
			cumulativeLoss:  common.Big0,
			queueItem:       nil,
		}
		dir, heavy, weight := computeEdgeAndDirection(fromTx, toTx, 100)
		if dir != tc.expectedDir {
			t.Fatal(dir, tc.expectedDir)
		}
		if heavy != tc.expectedHeavy {
			t.Fatal(heavy, tc.expectedHeavy)
		}
		if weight.Cmp(big.NewInt(tc.expectedWeight)) != 0 {
			t.Fatal(weight, tc.expectedWeight)
		}

		revDir, revHeavy, revWeight := computeEdgeAndDirection(toTx, fromTx, 100)
		if revDir == dir {
			t.Fatal(revDir, dir)
		}
		if revHeavy != heavy {
			t.Fatal(revHeavy, heavy)
		}
		if revWeight.Cmp(weight) != 0 {
			t.Fatal(revWeight, weight)
		}
	}
}
