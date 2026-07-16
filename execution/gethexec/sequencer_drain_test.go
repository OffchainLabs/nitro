// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	testBlockNumber = 100
	testBaseFee     = 100
)

func makeTestQueueItem(t *testing.T, nonce uint64, gasFeeCap int64) (txQueueItem, chan error) {
	t.Helper()
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasFeeCap: big.NewInt(gasFeeCap),
		GasTipCap: big.NewInt(0),
		Gas:       21000,
	})
	resultChan := make(chan error, 1)
	item := newRegularTxQueueItem(context.Background(), tx, nil, resultChan, false, 0)
	return item, resultChan
}

func TestDrainQueueItemsEmptyQueues(t *testing.T) {
	txQueue := make(chan txQueueItem, 1)
	auctionQueue := make(chan txQueueItem, 1)
	retryQueue := &synchronizedTxQueue{}
	items := drainQueueItems(txQueue, retryQueue, auctionQueue)
	if len(items) != 0 {
		t.Errorf("drained %d items from empty queues, want 0", len(items))
	}
}

func TestDrainQueueItemsPriorityOrder(t *testing.T) {
	txQueue := make(chan txQueueItem, 2)
	auctionQueue := make(chan txQueueItem, 2)
	retryQueue := &synchronizedTxQueue{}

	const feeCap = 0 // fee validation is not exercised in this test

	// Nonces identify the expected drain order: auction resolution, then retry, then submitted.
	for _, nonce := range []uint64{4, 5} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		txQueue <- item
	}
	for _, nonce := range []uint64{2, 3} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		retryQueue.Push(item)
	}
	for _, nonce := range []uint64{0, 1} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		auctionQueue <- item
	}

	items := drainQueueItems(txQueue, retryQueue, auctionQueue)
	if len(items) != 6 {
		t.Fatalf("drained %d items, want 6", len(items))
	}
	for i, item := range items {
		if item.tx.Nonce() != uint64(i) { // #nosec G115
			t.Errorf("items[%d] has nonce %d, want %d", i, item.tx.Nonce(), i)
		}
	}
	if len(txQueue) != 0 || retryQueue.Len() != 0 || len(auctionQueue) != 0 {
		t.Error("queues should be empty after draining")
	}
}

func TestValidateQueueItemAcceptsValidItem(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemRejectsCanceledContext(t *testing.T) {
	config := DefaultSequencerConfig
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.ctx = canceledCtx
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, context.Canceled) {
		t.Errorf("validateQueueItem() = %v, want %v", err, context.Canceled)
	}
}

func TestValidateQueueItemRejectsOversizedTx(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.txSize = config.MaxTxDataSize + 1
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, txpool.ErrOversizedData) {
		t.Errorf("validateQueueItem() = %v, want %v", err, txpool.ErrOversizedData)
	}
}

func TestValidateQueueItemRejectsExpiredTimeboostedTx(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true
	item.blockStamp = 1 // testBlockNumber >= 1 + QueueTimeoutInBlocks
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	err := validateQueueItem(&config, header, item)
	if err == nil || !strings.Contains(err.Error(), "block based timeout") {
		t.Errorf("validateQueueItem() = %v, want block-age expiry error", err)
	}
}

func TestValidateQueueItemAcceptsUnexpiredTimeboostedTx(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true
	item.blockStamp = testBlockNumber - 1 // testBlockNumber < blockStamp + QueueTimeoutInBlocks
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemAcceptsTimeboostedTxWithoutBlockStamp(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true // blockStamp stays 0, which exempts the item from expiry
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemRejectsFeeCapBelowBasefee(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee-1)
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, core.ErrFeeCapTooLow) {
		t.Errorf("validateQueueItem() = %v, want %v", err, core.ErrFeeCapTooLow)
	}
}

func TestDrainAndValidateQueueItemsReturnsResultOnRejection(t *testing.T) {
	config := DefaultSequencerConfig
	rejectedItem, rejectedResultChan := makeTestQueueItem(t, 0, testBaseFee)
	rejectedItem.txSize = config.MaxTxDataSize + 1
	validItem, validResultChan := makeTestQueueItem(t, 1, testBaseFee)
	txQueue := make(chan txQueueItem, 2)
	txQueue <- rejectedItem
	txQueue <- validItem
	auctionQueue := make(chan txQueueItem, 1)
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	items := drainAndValidateQueueItems(&config, txQueue, &synchronizedTxQueue{}, auctionQueue, header)

	if len(items) != 1 || items[0].tx.Nonce() != 1 {
		t.Fatalf("drained items = %v, want only the valid item", items)
	}
	select {
	case err := <-rejectedResultChan:
		if !errors.Is(err, txpool.ErrOversizedData) {
			t.Errorf("rejected item got %v, want %v", err, txpool.ErrOversizedData)
		}
	default:
		t.Error("rejected item never received a result")
	}
	select {
	case err := <-validResultChan:
		t.Errorf("valid item got early result %v", err)
	default:
	}
}
